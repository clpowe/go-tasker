package main

import (
	"database/sql"
	"time"
)

func insertSession(db *sql.DB, taskID *int64, start, end time.Time, dur time.Duration) error {
	var tid any
	if taskID != nil {
		tid = *taskID
	}
	_, err := db.Exec(`INSERT INTO pomodoro_sessions(task_id, start_time, end_time, duration_seconds) VALUES(?,?,?,?)`, tid, start.UTC(), end.UTC(), int(dur.Seconds()))
	return err
}
