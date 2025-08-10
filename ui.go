package main

import (
	"database/sql"
	"strings"

	"github.com/rivo/tview"
)

func promptAddTask(app *tview.Application, db *sql.DB, rebuild func()) {
	in := tview.NewInputField().SetLabel("Title: ")
	form := tview.NewForm()
	form.AddFormItem(in).AddButton("Add", func() {
		title := strings.TrimSpace(in.GetText())
		if title != "" {
			_ = addTask(db, title)
		}
		rebuild()
	}).AddButton("Cancel", func() { rebuild() })
	form.SetBorder(true).SetTitle(" New Task ")
	app.SetRoot(centered(60, 7, form), true).SetFocus(form)
}

func confirmDelete(app *tview.Application, title string, onOK func(), onCancel func()) {
	m := tview.NewModal().SetText(`DELETE "` + title + `"?`).
		AddButtons([]string{"Delete", "Cancel"}).
		SetDoneFunc(func(i int, l string) {
			if l == "Delete" {
				onOK()
				return
			}
			// Treat any non-Delete selection (including Cancel) as cancel
			onCancel()
		})
	app.SetRoot(m, true).SetFocus(m)
}

func centered(w, h int, p tview.Primitive) tview.Primitive {
	return tview.NewFlex().AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).AddItem(p, h, 1, true).AddItem(nil, 0, 1, false), w, 1, true).
		AddItem(nil, 0, 1, false)
}
