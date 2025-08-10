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
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.running && t.mode == ModeWork {
		return
	}
	t.mode = ModeWork
	t.remaining = t.workDuration
	t.currentTaskID = taskID
	t.startTime = time.Now().UTC()
	t.running = true
	t.stopCh = make(chan struct{})
	t.ticker = time.NewTicker(time.Second)
	if t.onStateChanged != nil {
		t.onStateChanged()
	}
	go t.run()
}

func (t *PomodoroTimer) PauseOrStop() {
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
	elapsed := t.workDuration - t.remaining
	if t.mode == ModeWork && elapsed >= time.Minute && t.onWorkCompleted != nil {
		end := time.Now().UTC()
		t.onWorkCompleted(t.currentTaskID, end.Add(-elapsed), end, elapsed)
	}
	if t.onStateChanged != nil {
		t.onStateChanged()
	}
	t.mu.Unlock()
}

func (t *PomodoroTimer) Reset() {
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
	if t.onStateChanged != nil {
		t.onStateChanged()
	}
	t.mu.Unlock()
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
				t.mu.Lock()
				end := time.Now().UTC()
				if t.onWorkCompleted != nil {
					dur := t.workDuration
					t.onWorkCompleted(t.currentTaskID, end.Add(-dur), end, dur)
				}
				t.mode = ModeBreak
				t.remaining = t.breakDuration
				t.currentTaskID = nil
				if t.onStateChanged != nil {
					t.onStateChanged()
				}
				t.mu.Unlock()
			} else if rem <= 0 && mode == ModeBreak {
				t.PauseOrStop()
			}
		}
	}
}
