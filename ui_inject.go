package main

import (
	"fmt"
	"os"

	. "go.hasen.dev/shirei"
	"go.hasen.dev/shirei/widgets"
)

// initSelectView / injectSelectView mirror initialise()/injection().
func initSelectView() {
	filePickView("SELECTIONNER LE FICHIER ORPHEE", []string{"xlsx"}, runInitFile, scrMain)
}

func injectSelectView() {
	filePickView("SELECTIONNER LE FICHIER EXCEL à INJECTER", []string{"xlsx"}, runInjectFile, scrMain)
}

// filePickView wraps the FileBrowserPanel; accepting a file calls onFile,
// Fermer/Escape go back to the given screen.
func filePickView(title string, exts []string, onFile func(string), back screen) {
	if app.fileCwd == "" {
		if h, err := os.UserHomeDir(); err == nil {
			app.fileCwd = h
		} else {
			app.fileCwd = "."
		}
	}
	Container(Attrs(Expand, Pad(16), Gap(10), Background(0, 0, 100, 1)), func() {
		Label(title, FontSize(14), FontWeight(WeightBold), TextColor(0, 0, 15, 1))
		if ok := widgets.FileBrowserPanel(&app.fileCwd, &app.fileFilter, &app.fileSelIdx, &app.fileSel,
			widgets.FileBrowserAttrs{Title: title, Width: 700, Files: true, Exts: exts}); ok {
			p := app.fileSel
			app.fileSel, app.fileSelIdx = "", -1
			onFile(p)
			return
		}
		Container(Attrs(Row, Expand, CrossMid), func() {
			Element(Attrs(Grow(1)))
			if widgets.Button(widgets.NoIcon, "Fermer") {
				app.fileSel, app.fileSelIdx = "", -1
				app.screen = back
			}
		})
	})
}

// runInitFile mirrors initialise() after the file dialog.
func runInitFile(p string) {
	app.screen = scrMain
	bs, rows, err := readInitXLSX(p)
	if err != nil {
		app.msgQuit = true
		app.showErr("ERREUR", "LE FICHIER EXCEL CHOISI N'EST PAS CONFORME\n RECOMMENCEZ")
		return
	}
	if err := rebuildBEN(bs, rows); err != nil {
		app.msgQuit = true
		app.showErr("ERREUR", "PROBLÈME LORS DE LA CRÉATION DE LA BASE ")
		return
	}
	app.msgQuit = true
	app.showInfo("INFO", "MISE À JOUR TERMINÉE\nL'APPLICATION VA SE FERMER \nVEUILLEZ RELANCER SVP")
}

// runInjectFile mirrors injection() after the file dialog.
func runInjectFile(p string) {
	app.screen = scrMain
	rows, err := readJournalXLSX(p)
	if err != nil {
		app.msgQuit = true
		app.showErr("ERREUR", "LE FICHIER EXCEL CHOISI N'EST PAS CONFORME\n RECOMMENCEZ")
		return
	}
	if err := insertInjection(rows); err != nil {
		app.msgQuit = true
		app.showErr("ERREUR", "LA BASE N'A PAS PU ÊTRE MISE À JOUR")
		return
	}
	app.showInfo("INFO", "LA BASE a été MISE à Jour\n   "+fmt.Sprintf("%d", len(rows))+"  ENREGISTREMENTS AJOUTÉS")
}
