package main

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func openDB() (*sql.DB, error) {
	dir, _ := os.UserConfigDir()
	if dir == "" {
		dir = "."
	}
	appDir := filepath.Join(dir, "tasker")
	_ = os.MkdirAll(appDir, 0755)
	db, err := sql.Open("sqlite", filepath.Join(appDir, "tasker.db"))
	if err != nil {
		return nil, err
	}
	_, _ = db.Exec("PRAGMA busy_timeout = 2000;")
	return db, nil
}

func migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS tasks(
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      title TEXT NOT NULL,
      done INTEGER NOT NULL DEFAULT 0,
      created_at TIMESTAMP NOT NULL DEFAULT (datetime('now')),
      completed_at TIMESTAMP
    );`,
		`CREATE TABLE IF NOT EXISTS pomodoro_sessions(
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      task_id INTEGER,
      start_time TIMESTAMP NOT NULL,
      end_time TIMESTAMP NOT NULL,
      duration_seconds INTEGER NOT NULL,
      FOREIGN KEY(task_id) REFERENCES tasks(id) ON DELETE SET NULL
    );`,
		`CREATE TABLE IF NOT EXISTS settings(
      key TEXT PRIMARY KEY,
      value TEXT NOT NULL
    );`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	_, _ = db.Exec(`INSERT INTO settings(key,value)
      VALUES('daily_focus_goal_minutes','120')
      ON CONFLICT(key) DO NOTHING`)
	return nil
}
