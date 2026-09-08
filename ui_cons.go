package main

import (
	"fmt"

	. "go.hasen.dev/shirei"
	"go.hasen.dev/shirei/widgets"
)

// consSelectView mirrors indiv_consolidation(): titles, lists, buttons.
func consSelectView() {
	Container(Attrs(Expand, Pad(16), Gap(10), Background(0, 0, 100, 1)), func() {
		Container(Attrs(Expand, FixHeight(56), Center, Background(220, 5, 97, 1), Corners(3)), func() {
			Label("BS de "+nomBS+" \nSaisie globale de consolidation \n Sélectionnez l'adhérent concerné dans la liste de gauche...",
				FontSize(13), TextColor(0, 0, 12, 1))
		})
		Container(Attrs(Expand, Row, FixHeight(300), Gap(10)), func() {
			benList("cons-bens", pickCons)
			linesBox("cons-sel", consSelLines())
		})
		Container(Attrs(Expand, Row, Gap(12), CrossMid, Background(220, 5, 97, 1), Corners(3), Pad2(10, 6)), func() {
			if widgets.Button(widgets.NoIcon, "Valider") {
				runConsGo()
			}
			if widgets.Button(widgets.NoIcon, "Effacer la sélection") {
				app.consRows = nil
			}
		})
	})
}

// pickCons mirrors selection_indiv_consolidation(): opens the fields dialog.
func pickCons(i int) {
	app.consPending = i
	app.consDate = todayFR()
	app.consHP, app.consHD, app.consHL, app.consHR, app.consHN = "0", "0", "0", "0", "0"
	app.consCom = ""
	app.screen = scrConsFields
}

// consSelLines reproduces the exact listbox text of selection_i_consolidation().
func consSelLines() []journalLine {
	out := []journalLine{
		{text: "BÉNÉVOLES SÉLECTIONNÉS"},
		{text: "   "},
		{text: "\n"},
	}
	if len(app.consRows) > 0 {
		out = append(out, journalLine{text: padRightRunes("Nom", 20) + " " + padRightRunes("Prénom", 15) + " " +
			padRightRunes("Date", 11) + "   " + padRightRunes("Per.", 4) + "   " + padRightRunes("Ext.", 4) + "    " +
			padRightRunes("DDV", 4) + "    " + padRightRunes("Rég.", 4) + "   " + padRightRunes("Nat.", 4) + " " +
			padRightRunes("Commentaire", 30)})
	}
	for _, e := range app.consRows {
		b := bens[e.idx]
		out = append(out, journalLine{
			text: fmt.Sprintf("%s %s %s  %4d    %4d    %4d    %4d  %4d  %s",
				padRightRunes(b.Nom, 20), padRightRunes(b.Prenom, 15), padRightRunes(e.dateISO, 11),
				e.hp, e.hd, e.hl, e.hr, e.hn, padRightRunes(cutRunes(e.comm, 30), 30)),
		})
	}
	return out
}

// consFieldsView mirrors selection_indiv_consolidation().
func consFieldsView() {
	if app.consPending < 0 || app.consPending >= len(bens) {
		app.screen = scrConsSelect
		return
	}
	b := bens[app.consPending]
	Container(Attrs(Expand, Pad(20), Gap(12), Background(220, 5, 97, 1)), func() {
		Container(Attrs(Expand, Center, Background(0, 0, 100, 1), Corners(3), Pad2(6, 4)), func() {
			Label(b.Nom+"  "+b.Prenom, FontSize(14), TextColor(0, 0, 12, 1))
		})
		dateRow("Date  :", &app.consDate)
		Label("Saisie multiple de consolidation \n Indiquez une durée par catégorie (en heures entières) : ",
			FontSize(12), TextColor(0, 0, 15, 1))
		dateRow("Permanence:", &app.consHP)
		dateRow("Extérieur:", &app.consHD)
		dateRow("Heures DDV:", &app.consHL)
		dateRow("Régional:", &app.consHR)
		dateRow("National:", &app.consHN)
		Container(Attrs(Row, Gap(8), CrossMid), func() {
			Label("Commentaire facultatif", FontSize(12), TextColor(0, 0, 15, 1))
			Container(Attrs(FixWidth(220), Background(0, 0, 100, 1), Corners(3)), func() {
				widgets.TextInput(&app.consCom)
			})
		})
		Container(Attrs(CrossMid, FixWidth(140)), func() {
			if widgets.Button(widgets.NoIcon, "VALIDER") {
				validateConsFields()
			}
		})
	})
}

// validateConsFields mirrors verif_saisie_indiv_consolidation() (all-zero check fixed).
func validateConsFields() {
	if msg := validateDateFR(app.consDate); msg != "" {
		app.showErr("ERREUR", msg)
		return
	}
	hp, err1 := parseDuree(app.consHP)
	hd, err2 := parseDuree(app.consHD)
	hl, err3 := parseDuree(app.consHL)
	hr, err4 := parseDuree(app.consHR)
	hn, err5 := parseDuree(app.consHN)
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil {
		app.showErr("ERREUR", "UNE DURÉE AU MOINS  EST ERRONÉE")
		return
	}
	b := bens[app.consPending]
	if app.consHL != "0" && b.Donneur != "DDV" {
		app.showErr("ERREUR", "POUR LES HEURES DE DDV LE BÉNÉVOLE DOIT ÊTRE DDV.\n RECTIFIEZ SVP.")
		return
	}
	if hp == 0 && hd == 0 && hl == 0 && hr == 0 && hn == 0 {
		app.showErr("ERREUR", "VOUS N'AVEZ SAISI AUCUNE VALEUR.\n RECOMMENCEZ.")
		return
	}
	app.consRows = append(app.consRows, selEntry{
		idx:     app.consPending,
		dateISO: frToISO(app.consDate),
		hp:      hp,
		hd:      hd,
		hl:      hl,
		hr:      hr,
		hn:      hn,
		comm:    app.consCom,
	})
	app.screen = scrConsSelect
}

// runConsGo mirrors go_i_consolidation() + maj_journal_i_consolidation().
func runConsGo() {
	var err error
	if len(app.consRows) > 0 {
		entries := make([]consInsert, 0, len(app.consRows))
		for _, e := range app.consRows {
			entries = append(entries, consInsert{
				idx:     e.idx,
				dateISO: e.dateISO,
				hp:      e.hp,
				hd:      e.hd,
				hl:      e.hl,
				hr:      e.hr,
				hn:      e.hn,
				comm:    cutRunes(e.comm, 30),
			})
		}
		err = insertConsList(entries)
	}
	app.consRows = nil
	app.screen = scrMain
	if err != nil {
		app.showErr("ERREUR", "LA BASE N'A PAS PU ÊTRE MISE À JOUR.\n RECOMMENCEZ ")
		return
	}
	app.showInfo("MESSAGE", "LA BASE A ÉTÉ MISE À JOUR")
}
