package main

import (
	"database/sql"
	"sync"
	"time"

	"github.com/rivo/tview"
)

type Task struct {
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
