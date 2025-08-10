package main

import (
	"sync"
	"time"
)

type PomodoroMode int

const (
	ModeWork PomodoroMode = iota
	ModeBreak
)

type PomodoroTimer struct {
	mode            PomodoroMode
	workDuration    time.Duration
	breakDuration   time.Duration
	remaining       time.Duration
	running         bool
	ticker          *time.Ticker
	mu              sync.Mutex
	startTime       time.Time
	currentTaskID   *int64
	onTick          func()
	onWorkCompleted func(taskID *int64, start, endTime time.Time, dur time.Duration)
	onStateChanged  func()
	stopCh          chan struct{}
}

func (t *PomodoroTimer) StartWork(taskID *int64) {
	// Acquire lock to update state, then release before invoking callbacks
	t.mu.Lock()
	if t.running && t.mode == ModeWork {
		t.mu.Unlock()
		return
	}
	t.mode = ModeWork
	t.remaining = t.workDuration
	t.currentTaskID = taskID
	t.startTime = time.Now().UTC()
	t.running = true
	t.stopCh = make(chan struct{})
	t.ticker = time.NewTicker(time.Second)
	t.mu.Unlock()

	if t.onStateChanged != nil {
		go t.onStateChanged()
	}
	go t.run()
}

func (t *PomodoroTimer) PauseOrStop() {
	// Stop the timer, then release the lock before invoking callbacks
	t.mu.Lock()
	if !t.running {
		t.mu.Unlock()
		return
	}
	t.running = false
	if t.ticker != nil {
		t.ticker.Stop()
	}
	if t.stopCh != nil {
		close(t.stopCh)
	}
	// Capture values needed for callbacks before unlocking
	mode := t.mode
	elapsed := t.workDuration - t.remaining
	taskID := t.currentTaskID
	t.mu.Unlock()
	if mode == ModeWork && elapsed >= time.Minute && t.onWorkCompleted != nil {
		end := time.Now().UTC()
		// fire completion callback asynchronously
		go t.onWorkCompleted(taskID, end.Add(-elapsed), end, elapsed)
	}
	if t.onStateChanged != nil {
		// fire state change asynchronously
		go t.onStateChanged()
	}
}

func (t *PomodoroTimer) Reset() {
	// Reset internal state, then release lock before notifying listeners
	t.mu.Lock()
	if t.ticker != nil {
		t.ticker.Stop()
	}
	if t.stopCh != nil {
		close(t.stopCh)
	}
	t.running = false
	t.mode = ModeWork
	t.remaining = t.workDuration
	t.currentTaskID = nil
	t.mu.Unlock()

	if t.onStateChanged != nil {
		go t.onStateChanged()
	}
}

func (t *PomodoroTimer) run() {
	for {
		select {
		case <-t.stopCh:
			return
		case <-t.ticker.C:
			t.mu.Lock()
			if !t.running {
				t.mu.Unlock()
				return
			}
			if t.remaining > 0 {
				t.remaining -= time.Second
			}
			rem := t.remaining
			mode := t.mode
			t.mu.Unlock()

			if t.onTick != nil {
				t.onTick()
			}

			if rem <= 0 && mode == ModeWork {
				// Transition to break - capture data for callbacks before locking
				t.mu.Lock()
				end := time.Now().UTC()
				dur := t.workDuration
				taskID := t.currentTaskID
				onWorkCompleted := t.onWorkCompleted
				onStateChanged := t.onStateChanged
				
				t.mode = ModeBreak
				t.remaining = t.breakDuration
				t.currentTaskID = nil
				t.mu.Unlock()

				// Call callbacks after releasing the mutex
				if onWorkCompleted != nil {
					onWorkCompleted(taskID, end.Add(-dur), end, dur)
				}
				if onStateChanged != nil {
					go onStateChanged()
				}
			} else if rem <= 0 && mode == ModeBreak {
				// End of break: stop the timer
				t.PauseOrStop()
			}
		}
	}
}
