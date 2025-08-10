package main

import (
	"log"

	"github.com/rivo/tview"
)

func main() {
	app := tview.NewApplication()
	root := tview.NewTextView().
		SetBorder(true).
		SetTitle("Pomodoro")
	if err := app.SetRoot(root, true).Run(); err != nil {
		log.Fatal(err)
	}
}
