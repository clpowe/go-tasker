package main

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strconv"
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

func openDB() (*sql.DB, error) {
	// Put DB in a config dir if possible, else current dir
	dir, err := os.UserConfigDir()
	if err != nil || dir == "" {
		dir = "."
	}

	appDir := filepath.Join(dir, "tasker")
	_ = os.MkdirAll(appDir, 0755)
	path := filepath.Join(appDir, "tasker.db")

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	// Busy timeout to reduce lock errors
	_, _ = db.Exec("PRAGMA busy_timeout = 2000;")
	return db, nil
}

func migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS tasks (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      title TEXT NOT NULL,
      done INTEGER NOT NULL DEFAULT 0,
      created_at TIMESTAMP NOT NULL DEFAULT (datetime('now')),
      completed_at TIMESTAMP
    );`,
		`CREATE TABLE IF NOT EXISTS pomodoro_sessions (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      task_id INTEGER,
      start_time TIMESTAMP NOT NULL,
      end_time TIMESTAMP NOT NULL,
      duration_seconds INTEGER NOT NULL,
      FOREIGN KEY(task_id) REFERENCES tasks(id) ON DELETE SET NULL
    );`,
		`CREATE TABLE IF NOT EXISTS settings (
      key TEXT PRIMARY KEY,
      value TEXT NOT NULL
    );`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	// Ensure default goal
	var cnt int
	_ = db.QueryRow(
		"SELECT COUNT(*) FROM settings WHERE key = 'daily_focus_goal_minutes'",
	).Scan(&cnt)
	if cnt == 0 {
		_, _ = db.Exec(
			"INSERT INTO settings(key,value) VALUES(?,?)",
			"daily_focus_goal_minutes",
			"120",
		)
	}
	return nil
}

func addTask(db *sql.DB, title string) error {
	_, err := db.Exec(
		"INSERT INTO tasks(title, done) VALUES(?,0)",
		title,
	)
	return err
}

func listTask(db *sql.DB) ([]Task, error) {
	rows, err := db.Query(
		"SELECT id, title, done , created_at, completed_at" +
			"FROM tasks ORDER BY done ASC, created_at DESC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []Task
	for rows.Next() {
		var t Task
		var done int
		if err := rows.Scan(
			&t.ID,
			&t.Title,
			&done,
			&t.CreatedAt,
			&t.CompletedAt,
		); err != nil {
			return nil, err
		}
		t.Done = done == 1
		res = append(res, t)
	}
	return res, rows.Err()
}

func toggleTaskDone(db *sql.DB, id int64, done bool) error {
	if done {
		_, err := db.Exec(
			"UPDATE tasks SET done=1, completed_at=datetime('now') WHERE id=?",
			id,
		)
		return err
	}
	_, err := db.Exec(
		"UPDATE tasks SET done=0, completed_at=NULL WHERE id=?",
		id,
	)
	return err
}

func deleteTask(db *sql.DB, id int64) error {
	_, err := db.Exec("DELETE FROM tasks WHERE id=?", id)
	return err
}

func insertSession(
	db *sql.DB,
	taskID *int64,
	start time.Time,
	end time.Time,
	dur time.Duration,
) error {
	var tid interface{}
	if taskID == nil {
		tid = nil
	} else {
		tid = *taskID
	}
	_, err := db.Exec(
		"INSERT INTO pomodoro_sessions(task_id, start_time, end_time, "+
			"duration_seconds) VALUES(?,?,?,?)",
		tid,
		start.UTC(),
		end.UTC(),
		int(dur.Seconds()),
	)
	return err
}

func getTodayFocusMinutes(db *sql.DB) (int, error) {
	start, end := todayBounds()
	var secs int
	err := db.QueryRow(
		"SELECT COALESCE(SUM(duration_seconds),0) FROM pomodoro_sessions"+
			"WHERE start_time >= ? AND start_time < ?",
		start.UTC(),
		end.UTC(),
	).Scan(&secs)
	if err != nil {
		return 0, err
	}
	return secs / 60, nil
}

func countTodaySessions(db *sql.DB) (int, error) {
	start, end := todayBounds()
	var cnt int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM pomodoro_sessions WHERE start_time >= ? "+
			"AND start_time < ?",
		start.UTC(),
		end.UTC(),
	).Scan(&cnt)
	return cnt, err
}

func getDailyGoal(db *sql.DB) (int, error) {
	var val string
	err := db.QueryRow(
		"SELECT value FROM settings WHERE key=?",
		"daily_focus_goal_minutes",
	).Scan(&val)
	if err != nil {
		return 0, err
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func setDailyGoal(db *sql.DB, minutes int) error {
	if minutes <= 0 {
		return errors.New("minutes must be > 0")
	}
	_, err := db.Exec(
		"INSERT INTO settings(key,value) VALUES(?,?) "+
			"ON CONFLICT(key) DO UPDATE SET value=excluded.value",
		"daily_focus_goal_minutes",
		strconv.Itoa(minutes),
	)
	return err
}

/***** helpers *****/

func todayBounds() (time.Time, time.Time) {
	now := time.Now().UTC()
	y, m, d := now.Date()
	loc := now.Location()
	start := time.Date(y, m, d, 0, 0, 0, 0, loc)
	end := time.Date(y, m, d, 23, 59, 59, 0, loc)
	return start, end
}
