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

	timerView := tview.NewTextView().SetBorder(true).SetTitle(" Pomodoro ")
	infoView := tview.NewTextView().SetBorder(true).SetTitle(" Today ")
	footer := tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter).
		SetText("[a] Add  [e] Toggle  [d] Delete  [p] Start/Stop  [r] Reset  [g] Goal  [q] Quit")

	left := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(list, 0, 1, true).AddItem(footer, 1, 0, false)
	right := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(timerView, 8, 0, false).AddItem(infoView, 0, 1, false)
	root := tview.NewFlex().AddItem(left, 0, 2, true).AddItem(right, 0, 3, false)

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
		case 'd':
			i := list.GetCurrentItem()
			if i >= 0 && i < len(data) {
				t := data[i]
				confirmDelete(app, t.Title, func() {
					_ = deleteTask(db, t.ID)
					refresh()
					app.SetRoot(root, true)
				})
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
