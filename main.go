package main

import (
	"fmt"
	"log"
	"time"

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

	timerView := tview.NewTextView()
	timerView.SetBorder(true)
	timerView.SetTitle(" Pomodoro ")

	infoView := tview.NewTextView()
	infoView.SetBorder(true)
	infoView.SetTitle(" Today ")

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

	workDur, breakDur := 25*time.Minute, 5*time.Minute
	timer := &PomodoroTimer{
		workDuration:  workDur,
		breakDuration: breakDur,
		remaining:     workDur,
	}

	timer.onWorkCompleted = func(taskID *int64, start, end time.Time, dur time.Duration) {
		_ = insertSession(db, taskID, start, end, dur)
	}

	goal, _ := getDailyGoal(db)
	today, _ := getTodayFocusMinutes(db)

	renderInfo := func() {
		infoView.Clear()
		fmt.Fprintf(infoView, "Work sessions today: (see sessions)\n")
	}

	renderTimer := func() {
		timerView.Clear()
		timer.mu.Lock()
		rem := timer.remaining
		mode := timer.mode
		running := timer.running
		timer.mu.Unlock()
		mm, ss := int(rem.Minutes()), int(rem.Seconds())%60
		modeS := map[PomodoroMode]string{ModeWork: "Work", ModeBreak: "Break"}[mode]
		state := "paused"
		if running {
			state = "running"
		}
		pct := 0.0
		if goal > 0 {
			pct = float64(today) / float64(goal)
			if pct > 1 {
				pct = 1
			}
		}
		fmt.Fprintf(timerView, "Mode: %s (%s)\nRemaining: %02d:%02d\nToday: %d/%d min %s\n",
			modeS, state, mm, ss, today, goal, progressBar(20, pct))
	}

	timer.onWorkCompleted = func(taskID *int64, start, end time.Time, dur time.Duration) {
		_ = insertSession(db, taskID, start, end, dur)
		today, _ = getTodayFocusMinutes(db)
		app.QueueUpdateDraw(func() { renderTimer(); renderInfo() })
	}
	// Use draw-queuing updates (safe from background goroutines)
	timer.onTick = func() { app.QueueUpdateDraw(renderTimer) }
	timer.onStateChanged = func() { app.QueueUpdateDraw(renderTimer) }

	app.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		switch ev.Rune() {
		case 'g':
			promptSetGoal(app, db, goal, func() { goal, _ = getDailyGoal(db); app.SetRoot(root, true); renderTimer() })
			return nil
		case 'p':
			if timer.running {
				timer.PauseOrStop()
			} else {
				timer.StartWork(nil)
			}
			// Immediately reflect state change without queuing (we're already in UI loop)
			renderTimer()
			return nil
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
				}, func() {
					// Cancel: just return to the main layout
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

	renderTimer()

	if err := app.SetRoot(root, true).Run(); err != nil {
		log.Fatal(err)
	}
}
