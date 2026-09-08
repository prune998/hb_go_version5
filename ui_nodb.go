package main

import (
	. "go.hasen.dev/shirei"
	"go.hasen.dev/shirei/widgets"
)

// noDBView is shown at startup when no usable database exists: either locate
// a local .db file or start from an empty one. It cannot be dismissed.
func noDBView() {
	Container(Attrs(Expand, Pad(20), Gap(12), Background(0, 0, 100, 1)), func() {
		Label("BASE NON TROUVÉE...", FontSize(14), FontWeight(WeightBold), TextColor(0, 0, 15, 1))
		Label("VEUILLEZ RESTAURER UNE SAUVEGARDE\nOU RÉINITIALISER", FontSize(12),
			TextColor(0, 0, 25, 1), Fonts(Monospace...))
		Container(Attrs(Expand, Row, Gap(10), CrossMid), func() {
			if widgets.Button(widgets.NoIcon, "CHOISIR UNE BASE LOCALE") {
				app.fileSel, app.fileSelIdx = "", -1
				app.screen = scrNoDBFile
			}
			if widgets.Button(widgets.NoIcon, "DEPART AVEC UNE BASE VIDE") {
				runEmptyStart()
			}
		})
	})
}

func noDBFileView() {
	filePickView("SELECTIONNER LE FICHIER DE BASE ( .db )", []string{"db"}, runAdoptFile, scrNoDB)
}

func runEmptyStart() {
	if err := createEmptyDB(); err != nil {
		app.showErr("ERREUR", "LA BASE VIDE N'A PAS PU ÊTRE CRÉÉE\n RECOMMENCEZ")
		return
	}
	app.screen = scrMain
	app.showInfo("MESSAGE", "BASE VIDE CRÉÉE :\n FICHIER "+dbPath)
}

func runAdoptFile(p string) {
	if err := adoptDB(p); err != nil {
		app.screen = scrNoDB
		app.showErr("ERREUR", "LE FICHIER CHOISI N'EST PAS UNE BASE VALIDE\n RECOMMENCEZ")
		return
	}
	app.screen = scrMain
	app.showInfo("MESSAGE", "BASE CHARGÉE :\n FICHIER "+p)
}
