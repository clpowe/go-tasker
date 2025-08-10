package main

import "database/sql"

type Task struct {
	ID    int64
	Title string
	Done  bool
}

func listTasks(db *sql.DB) ([]Task, error) {
	rows, err := db.Query(`SELECT id,title,done FROM tasks ORDER BY done ASC, created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Task
	for rows.Next() {
		var t Task
		var done int
		if err := rows.Scan(&t.ID, &t.Title, &done); err != nil {
			return nil, err
		}
		t.Done = done == 1
		out = append(out, t)
	}
	return out, rows.Err()
}
