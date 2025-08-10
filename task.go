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

func addTask(db *sql.DB, title string) error {
	_, err := db.Exec(`INSERT INTO tasks(title, done) VALUES(?,0)`, title)
	return err
}

func toggleTaskDone(db *sql.DB, id int64, done bool) error {
	if done {
		_, err := db.Exec(`UPDATE tasks SET done=1,completed_at=datetime('now') WHERE id=?`, id)
		return err
	}
	_, err := db.Exec(`UPDATE tasks SET done=0,completed_at=NULL WHERE id=?`, id)
	return err
}
