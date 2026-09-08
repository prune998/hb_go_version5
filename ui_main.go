package main

import (
	"fmt"
	"os"

	. "go.hasen.dev/shirei"
	"go.hasen.dev/shirei/widgets"
)

// frame is the single shirei frame builder: main window + current dialog.
func frame() {
	ModAttrs(func(a *AttrSet) {
		a.Animations = 1
	})
	Container(Attrs(Grow(1), Expand, Background(220, 8, 96, 1)), func() {
		mainWindow()
	})

	switch app.screen {
	case scrPermAsk:
		Modal(420, app.dismissTo(scrMain), permAskView)
	case scrPermSelect:
		Modal(1000, app.dismissTo(scrMain), permSelectView)
	case scrIndivSelect:
		Modal(1300, app.dismissTo(scrMain), indivSelectView)
	case scrIndivFields:
		Modal(620, app.dismissTo(scrIndivSelect), indivFieldsView)
	case scrConsSelect:
		Modal(1280, app.dismissTo(scrMain), consSelectView)
	case scrConsFields:
		Modal(600, app.dismissTo(scrConsSelect), consFieldsView)
	case scrJournalAsk:
		Modal(420, app.dismissTo(scrMain), journalAskView)
	case scrJournalView:
		Modal(1200, app.dismissTo(scrMain), journalView)
	case scrInitSelect:
		Modal(760, app.dismissTo(scrMain), initSelectView)
	case scrInjectSelect:
		Modal(760, app.dismissTo(scrMain), injectSelectView)
	case scrNoDB:
		Modal(560, nil, noDBView)
	case scrNoDBFile:
		Modal(760, app.dismissTo(scrNoDB), noDBFileView)
	}

	if app.msg != msgNone {
		Modal(440, func() { app.closeMsg() }, messageView)
	}
	DebugPanel()
}

func mainWindow() {
	Container(Attrs(Expand, Gap(0)), func() {
		// Menu bar: Fichier / Action / A propos ...
		Container(Attrs(Expand, Row, FixHeight(36), Gap(8), Pad2(6, 8), CrossMid,
			Background(0, 0, 55, 1)), func() {
			widgets.MenuButton(widgets.NoIcon, "Fichier", fichierMenu)
			widgets.MenuButton(widgets.NoIcon, "Action", actionMenu)
			widgets.MenuButton(widgets.NoIcon, "A propos ...", aproposMenu)
			Element(Attrs(Grow(1)))
			Label("BS de "+nomBS, FontSize(11), TextColor(0, 0, 92, 1), Fonts(Monospace...))
		})

		// Banner, like the Python Canvas packed on top (image 640x573).
		Container(Attrs(Grow(1), Expand, Center, Background(220, 8, 98, 1)), func() {
			if p := bannerPath(); p != "" {
				Image(p, Vec2{640, 573})
			} else {
				Label("HB VERSION 5", FontSize(24), TextColor(0, 0, 30, 1))
			}
		})

		// Status strip (DB info) — replaces the empty lower part of the
		// original window with something useful.
		Container(Attrs(Expand, Pad(10), Gap(6), Background(220, 8, 98, 1)), func() {
			Label(fmt.Sprintf("%d bénévoles enregistrés — base : %s", len(bens), dbPath),
				FontSize(11), TextColor(0, 0, 40, 1), Fonts(Monospace...))
		})
	})
}

func fichierMenu() {
	if widgets.MenuItem(widgets.NoIcon, "Initialisation du fichier des bénévoles") {
		app.fileSel, app.fileSelIdx = "", -1
		app.screen = scrInitSelect
	}
	if widgets.MenuItem(widgets.NoIcon, "Sauvegarde") {
		runSauvegarde()
	}
	if widgets.MenuItem(widgets.NoIcon, "Restauration") {
		runRestauration()
	}
	if widgets.MenuItem(widgets.NoIcon, "Quitter") {
		quitApp()
	}
}

func actionMenu() {
	if widgets.MenuItem(widgets.NoIcon, "Saisie heures permanence") {
		app.permDate = todayFR()
		app.permDuree = "3"
		app.permDureeInt = 3
		app.permSel = nil
		app.screen = scrPermAsk
	}
	if widgets.MenuItem(widgets.NoIcon, "Saisie heures individuelles") {
		app.indivRows = nil
		app.indivDate = todayFR()
		app.indivDuree = "3"
		app.indivCat = 1
		app.indivCom = ""
		app.screen = scrIndivSelect
	}
	if widgets.MenuItem(widgets.NoIcon, "Saisie globale de consolidation") {
		app.consRows = nil
		app.consDate = todayFR()
		app.consHP, app.consHD, app.consHL, app.consHR, app.consHN = "0", "0", "0", "0", "0"
		app.consCom = ""
		app.screen = scrConsSelect
	}
	if widgets.MenuItem(widgets.NoIcon, "Consolidation par journal externe") {
		app.fileSel, app.fileSelIdx = "", -1
		app.screen = scrInjectSelect
	}
	if widgets.MenuItem(widgets.NoIcon, "Edition du journal") {
		app.jMode = 1
		app.jD1 = todayFR()
		app.jD2 = todayFR()
		app.screen = scrJournalAsk
	}
}

func aproposMenu() {
	if widgets.MenuItem(widgets.NoIcon, "Infos") {
		app.showInfo("MESSAGE", "HB_version5-- Elie Couzinié")
	}
	if widgets.MenuItem(widgets.NoIcon, "Lisez-moi ...") {
		openHelp()
	}
}

// messageView renders the INFO / ERREUR overlay (Go twin of messagebox).
func messageView() {
	err := app.msg == msgErr
	bgH, bgS, bgL := float32(0), float32(0), float32(97)
	if err {
		bgH, bgS, bgL = 0, 70, 94
	}
	Container(Attrs(Expand, Pad(20), Gap(12), Background(bgH, bgS, bgL, 1)), func() {
		title := app.msgTitle
		if title == "" {
			title = "MESSAGE"
		}
		Label(title, FontSize(14), TextColor(0, 0, 15, 1))
		if err {
			Label(app.msgText, FontSize(12), TextColor(0, 0, 25, 1), Fonts(Monospace...))
		} else {
			Label(app.msgText, FontSize(12), TextColor(0, 0, 15, 1))
		}
		Container(Attrs(CrossMid, FixWidth(140)), func() {
			if widgets.Button(widgets.NoIcon, "OK") {
				app.closeMsg()
			}
		})
	})
}

func runSauvegarde() {
	p, err := sauvegarde()
	if err != nil {
		app.showErr("ERREUR", "LA SAUVEGARDE\n N'A PAS PU ÊTRE EXÉCUTÉE")
		return
	}
	app.showInfo("MESSAGE", "SAUVEGARDE RÉUSSIE :\n FICHIER "+p)
}

func runRestauration() {
	dateFR, liveExists, err := restauration()
	if err != nil {
		app.showErr("ERREUR", "FICHIER DE SAUVEGARDE\n NON TROUVÉ")
		return
	}
	newBkp := appDir + "/sauvegarde.db"
	app.showInfo("MESSAGE", "LA SAUVEGARDE "+newBkp+"\nEST DATÉE DU "+dateFR)
	if liveExists {
		app.showInfo("MESSAGE", "LE FICHIER '"+dbPath+"'\nEST DÉJÀ PRÉSENT")
		return
	}
	src, err := os.Open(newBkp)
	if err != nil {
		app.showErr("MESSAGE", "ERREUR LORS DE LA RESTAURATION")
		return
	}
	defer src.Close()
	dst, err := os.Create(dbPath)
	if err != nil {
		app.showErr("MESSAGE", "ERREUR LORS DE LA RESTAURATION")
		return
	}
	defer dst.Close()
	if _, err := dst.ReadFrom(src); err != nil {
		app.showErr("MESSAGE", "ERREUR LORS DE LA RESTAURATION")
		return
	}
	app.showInfo("MESSAGE", "RESTAURATION RÉUSSIE\n FICHIER '"+newBkp+"'\nA ÉTÉ RESTAURÉ VERS "+dbPath)
}
