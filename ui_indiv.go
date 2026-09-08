package main

import (
	"fmt"

	. "go.hasen.dev/shirei"
	"go.hasen.dev/shirei/widgets"
)

// catShort mirrors the Python choix[3:6] slice ("1- Extérieur"[3:6] = "Ext", ...).
func catShort(cat int) string {
	switch cat {
	case 1:
		return "Ext"
	case 2:
		return "Le "
	case 3:
		return "Rég"
	default:
		return "Nat"
	}
}

// indivSelectView mirrors indiv(): titles, BEN list, selected list, buttons.
func indivSelectView() {
	Container(Attrs(Expand, Pad(16), Gap(10), Background(0, 0, 100, 1)), func() {
		Container(Attrs(Expand, FixHeight(56), Center, Background(220, 5, 97, 1), Corners(3)), func() {
			Label("BS de "+nomBS+" \nSaisie des heures individuelles \n Sélectionnez l'adhérent concerné dans la liste de gauche...",
				FontSize(13), TextColor(0, 0, 12, 1))
		})
		Container(Attrs(Expand, Row, FixHeight(300), Gap(10)), func() {
			benList("indiv-bens", pickIndiv)
			linesBox("indiv-sel", indivSelLines())
		})
		Container(Attrs(Expand, Row, Gap(12), CrossMid, Background(220, 5, 97, 1), Corners(3), Pad2(10, 6)), func() {
			if widgets.Button(widgets.NoIcon, "Valider") {
				runIndivGo()
			}
			if widgets.Button(widgets.NoIcon, "Effacer la sélection") {
				app.indivRows = nil
			}
		})
	})
}

// pickIndiv mirrors selection_indiv(): opens the fields dialog for the member.
func pickIndiv(i int) {
	app.indivPending = i
	app.indivDate = todayFR()
	app.indivDuree = "3"
	app.indivCat = 1
	app.indivCom = ""
	app.screen = scrIndivFields
}

// indivSelLines reproduces the exact listbox text of indiv()/selection_i().
func indivSelLines() []journalLine {
	out := []journalLine{
		{text: "BÉNÉVOLES SÉLECTIONNÉS"},
		{text: "   "},
		{text: "\n"},
	}
	for _, e := range app.indivRows {
		b := bens[e.idx]
		out = append(out, journalLine{
			text: fmt.Sprintf("%s %s %s %s %2.0f h.  %s ",
				padRightRunes(b.Nom, 20), padRightRunes(b.Prenom, 15), padRightRunes(e.dateISO, 15),
				padRightRunes(catShort(e.cat), 6), float64(e.duree), padRightRunes(e.comm, 30)),
			red: e.duree < 0,
		})
	}
	return out
}

// indivFieldsView mirrors selection_indiv(): header, date, catégorie, durée, com.
func indivFieldsView() {
	if app.indivPending < 0 || app.indivPending >= len(bens) {
		app.screen = scrIndivSelect
		return
	}
	b := bens[app.indivPending]
	Container(Attrs(Expand, Pad(20), Gap(12), Background(220, 5, 97, 1)), func() {
		Container(Attrs(Expand, Center, Background(0, 0, 100, 1), Corners(3), Pad2(6, 4)), func() {
			Label(b.Nom+"  "+b.Prenom, FontSize(14), TextColor(0, 0, 12, 1))
		})
		dateRow("Date  :", &app.indivDate)
		Container(Attrs(Row, Gap(8), CrossMid), func() {
			Label("Catégorie  :", FontSize(12), TextColor(0, 0, 15, 1))
			widgets.SegmentedControl(&app.indivCat,
				widgets.Cell("1- Extérieur", 1),
				widgets.Cell("2- Lecture DDV", 2),
				widgets.Cell("3- Régional", 3),
				widgets.Cell("4- National", 4),
			)
		})
		dateRow("Durée  : (en heures entières)", &app.indivDuree)
		Container(Attrs(Row, Gap(8), CrossMid), func() {
			Label("Commentaire \n(facultatif)", FontSize(12), TextColor(0, 0, 15, 1))
			Container(Attrs(FixWidth(220), Background(0, 0, 100, 1), Corners(3)), func() {
				widgets.TextInput(&app.indivCom)
			})
		})
		Container(Attrs(CrossMid, FixWidth(140)), func() {
			if widgets.Button(widgets.NoIcon, "VALIDER") {
				validateIndivFields()
			}
		})
	})
}

// validateIndivFields mirrors verif_saisie_indiv().
func validateIndivFields() {
	if msg := validateDateFR(app.indivDate); msg != "" {
		app.showErr("ERREUR", msg)
		return
	}
	n, err := parseDuree(app.indivDuree)
	if err != nil {
		app.showErr("ERREUR", "SAISIE DURÉE ERRONÉE")
		return
	}
	b := bens[app.indivPending]
	if app.indivCat == 2 && b.Donneur != "DDV" {
		app.showErr("ERREUR", "LE BÉNÉVOLE DOIT ÊTRE DDV")
		return
	}
	app.indivRows = append(app.indivRows, selEntry{
		idx:     app.indivPending,
		dateISO: frToISO(app.indivDate),
		duree:   n,
		cat:     app.indivCat,
		comm:    app.indivCom,
		red:     n < 0,
	})
	app.screen = scrIndivSelect
}

// runIndivGo mirrors go_i() + maj_journal_i().
func runIndivGo() {
	var err error
	if len(app.indivRows) > 0 {
		entries := make([]indivInsert, 0, len(app.indivRows))
		for _, e := range app.indivRows {
			entries = append(entries, indivInsert{
				idx:     e.idx,
				dateISO: e.dateISO,
				heure:   e.duree,
				cat:     e.cat,
				comm:    cutRunes(e.comm, 30),
			})
		}
		err = insertIndiv(entries)
	}
	app.indivRows = nil
	app.screen = scrMain
	if err != nil {
		app.showErr("ERREUR", "LA BASE N'A PAS PU ÊTRE MISE À JOUR.\n RECOMMENCEZ ")
		return
	}
	app.showInfo("MESSAGE", "LA BASE A ÉTÉ MISE À JOUR")
}
