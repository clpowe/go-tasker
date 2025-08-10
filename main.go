package main

import (
	"log"

	"github.com/rivo/tview"
)

func main() {
	db, err := openDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := migrate(db); err != nil {
		log.Fatal(err)
	}

	app := tview.NewApplication()

	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(" Tasks ")
	root := tview.NewFlex().AddItem(list, 0, 1, true)

	tasks, _ := listTasks(db)
	for _, t := range tasks {
		p := "[ ]"
		if t.Done {
			p = "[x]"
		}
		list.AddItem(p+" "+t.Title, "", 0, nil)
	}

	if err := app.SetRoot(root, true).Run(); err != nil {
		log.Fatal(err)
	}
}
