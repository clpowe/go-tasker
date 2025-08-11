package main

import (
	"database/sql"
	"errors"
	"strconv"
	"time"
)

func getDailyGoal(db *sql.DB) (int, error) {
	var v string
	if err := db.QueryRow(`SELECT value FROM settings WHERE key='daily_focus_goal_minutes'`).
		Scan(&v); err != nil {
		return 0, err
	}
	return strconv.Atoi(v)
}

func setDailyGoal(db *sql.DB, m int) error {
	if m <= 0 {
		return errors.New("minutes must be > 0")
	}
	_, err := db.Exec(`INSERT INTO settings(key,value) VALUES('daily_focus_goal_minutes',?)
	                   ON CONFLICT(key) DO UPDATE SET value=excluded.value`, strconv.Itoa(m))
	return err
}

func todayBounds() (time.Time, time.Time) {
	now := time.Now().UTC()
	y, m, d := now.Date()
	loc := now.Location()
	return time.Date(y, m, d, 0, 0, 0, 0, loc), time.Date(y, m, d, 23, 59, 59, 0, loc)
}

func getTodayFocusMinutes(db *sql.DB) (int, error) {
	start, end := todayBounds()
	var secs int
	if err := db.QueryRow(`SELECT COALESCE(SUM(duration_seconds),0) FROM pomodoro_sessions WHERE start_time>=? AND start_time<?`,
		start, end).Scan(&secs); err != nil {
		return 0, err
	}
	return secs / 60, nil
}
