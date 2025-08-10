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
	root := tview.NewTextView().
		SetBorder(true).
		SetTitle("Pomodoro")
	if err := app.SetRoot(root, true).Run(); err != nil {
		log.Fatal(err)
	}
}
