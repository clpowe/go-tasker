package main

import (
	"database/sql"
	"sync"
	"time"

	"github.com/rivo/tview"
)

type task struct {
	ID          int64
	Title       string
	Done        bool
	CreatedAt   time.Time
	CompletedAt sql.NullTime
}

type AppState struct {
	db                   *sql.DB
	app                  *tview.Application
	tasksList            *tview.List
	timerView            *tview.TextView
	footer               *tview.TextView
	infoView             *tview.TextView
	goalMinutes          int
	todayFocusMinutes    int
	workDuration         time.Duration
	breakDuration        time.Duration
	timer                *PomodoroTimer
	mu                   sync.Mutex
	selectedTaskID       *int64
	selectedTaskTitle    string
	lastTaskRefreshError error
}

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
	accrued         time.Duration
	currentTaskID   *int64
	onTick          func()
	onWorkCompleted func(
		taskID *int64,
		startTime time.Time,
		endTime time.Time,
		dur time.Duration,
	)
	onStateChanged func()
	stopCh         chan struct{}
}

func (t *PomodoroTimer) StartWork(taskID *int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.running && t.mode == ModeWork {
		return
	}
	t.mode = ModeWork
	t.remaining = t.workDuration
	t.accrued = 0
	t.currentTaskID = taskID
	t.startTime = time.Now().UTC()
	t.running = true
	t.stopCh = make(chan struct{})
	t.ticker = time.NewTicker(time.Second)
	if t.onStateChanged != nil {
		t.onStateChanged()
	}
}

func (t *PomodoroTimer) PauseOrStop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.running {
		return
	}

	t.running = false
	if t.ticker != nil {
		t.ticker.Stop()
	}

	if t.stopCh != nil {
		close(t.stopCh)
	}

	// Record Partial work if >= 60s
	elapsed := t.workDuration - t.remaining
	if t.mode == ModeWork && elapsed >= time.Minute {
		end := time.Now().UTC()
		start := end.Add(-elapsed)
		if t.onWorkCompleted != nil {
			t.onWorkCompleted(t.currentTaskID, start, end, elapsed)
		}
	}

	t.accrued = 0
	if t.onStateChanged != nil {
		t.onStateChanged()
	}
}

func (t *PomodoroTimer) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.ticker != nil {
		t.ticker.Stop()
	}

	if t.stopCh != nil {
		close(t.stopCh)
	}

	t.running = false
	t.mode = ModeWork
	t.remaining = t.workDuration
	t.accrued = 0
	t.currentTaskID = nil
	if t.onStateChanged != nil {
		t.onStateChanged()
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
			t.mu.Unlock()
			if t.onTick != nil {
				t.onTick()
			}
			if rem <= 0 {
				t.mu.Lock()
				if t.mode == ModeWork {
					end := time.Now().UTC()
					if t.onWorkCompleted != nil {
						t.onWorkCompleted(
							t.currentTaskID,
							t.startTime,
							end,
							t.remaining,
						)
					}
					// Auto switch to break (optional)
					t.mode = ModeBreak
					t.remaining = t.breakDuration
					t.startTime = time.Now().UTC()
					t.currentTaskID = nil
					if t.onStateChanged != nil {
						t.onStateChanged()
					}
				} else {
					// End of break: stop
					t.running = false
					t.ticker.Stop()
					if t.onStateChanged != nil {
						t.onStateChanged()
					}
					t.mu.Unlock()
					return
				}
				t.mu.Unlock()
			}
		}
	}
}

func main() {
	db, err := openDB()
}

/***** DB layer *****/
