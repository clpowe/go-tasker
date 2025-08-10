package main

import (
	"log"

	"github.com/gdamore/tcell/v2"
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

	var data []Task

	unchecked := "☐"
	checked := "☑︎"

	refresh := func() {
		list.Clear()
		data, _ = listTasks(db)
		for _, t := range data {
			mark := unchecked
			if t.Done {
				mark = checked
			}
			list.AddItem(mark+" "+t.Title, "", 0, nil)
		}
	}

	refresh()

	app.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		switch ev.Rune() {
		case 'e':
			i := list.GetCurrentItem()
			if i >= 0 && i < len(data) {
				_ = toggleTaskDone(db, data[i].ID, !data[i].Done)
				refresh()
			}
			return nil
		case 'a':
			promptAddTask(app, db, func() {
				refresh()
				app.SetRoot(root, true)
			})
			return nil
		}
		return ev
	})

	if err := app.SetRoot(root, true).Run(); err != nil {
		log.Fatal(err)
	}
}
