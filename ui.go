package main

import (
	"database/sql"
	"strconv"
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

func progressBar(width int, pct float64) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}
	filled := int(pct * float64(width))
	b := make([]rune, width)
	for i := 0; i < width; i++ {
		if i < filled {
			b[i] = '#'
		} else {
			b[i] = '-'
		}
	}
	return "[" + string(b) + "]"
}

func promptSetGoal(app *tview.Application, db *sql.DB, current int, onSave func()) {
	in := tview.NewInputField().SetLabel("Daily goal (minutes): ").SetText(strconv.Itoa(current))
	form := tview.NewForm().AddFormItem(in).
		AddButton("Save", func() {
			if m, err := strconv.Atoi(strings.TrimSpace(in.GetText())); err == nil && m > 0 {
				_ = setDailyGoal(db, m)
			}
			onSave()
		}).AddButton("Cancel", func() { onSave() })
	form.SetBorder(true).SetTitle(" Daily Focus Goal ")
	app.SetRoot(centered(60, 7, form), true).SetFocus(form)
}

func centered(w, h int, p tview.Primitive) tview.Primitive {
	return tview.NewFlex().AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).AddItem(p, h, 1, true).AddItem(nil, 0, 1, false), w, 1, true).
		AddItem(nil, 0, 1, false)
}
