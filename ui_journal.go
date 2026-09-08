package main

import (
	"fmt"

	. "go.hasen.dev/shirei"
	"go.hasen.dev/shirei/widgets"
)

// journalAskView mirrors choisir_le_journal(): type + two dates + Valider.
func journalAskView() {
	Container(Attrs(Expand, Pad(20), Gap(12), Background(220, 5, 97, 1)), func() {
		Label("EDITION DU JOURNAL\nCHOISISSEZ LE TYPE DE SORTIE :", FontSize(12), TextColor(0, 0, 15, 1))
		widgets.SegmentedControl(&app.jMode,
			widgets.Cell("1- Détaillé", 1),
			widgets.Cell("2- Récapitulatif", 2),
			widgets.Cell("3- Individuel", 3),
		)
		Label("INDIQUEZ LES DATES", FontSize(12), TextColor(0, 0, 15, 1))
		dateRow("Entre le :", &app.jD1)
		dateRow("et le :", &app.jD2)
		Container(Attrs(CrossMid, FixWidth(140)), func() {
			if widgets.Button(widgets.NoIcon, "Valider") {
				validateJournalAsk()
			}
		})
	})
}

func dateRow(label string, dst *string) {
	Container(Attrs(Row, Gap(8), CrossMid), func() {
		Label(label, FontSize(12), TextColor(0, 0, 15, 1))
		Container(Attrs(FixWidth(110), Background(0, 0, 100, 1), Corners(3)), func() {
			widgets.TextInput(dst)
		})
	})
}

// validateJournalAsk mirrors j() up to the query.
func validateJournalAsk() {
	if msg := validateRange(app.jD1, app.jD2); msg != "" {
		app.showErr("ERREUR", msg)
		return
	}
	rows, err := queryJournal(frToISO(app.jD1), frToISO(app.jD2))
	if err != nil {
		app.showErr("ERREUR", "ERREUR SUR LA BASE DE DONNÉES")
		return
	}
	app.jRows = rows
	app.jMembers = map[string][6]int{}
	for _, r := range rows {
		app.jSum[0] += int(r.HP)
		app.jSum[1] += int(r.HD)
		app.jSum[2] += int(r.HL)
		app.jSum[3] += int(r.HR)
		app.jSum[4] += int(r.HN)
		key := r.Nom + " " + r.Prenom
		v := app.jMembers[key]
		if r.HP < 0 {
			v[0]--
		} else if r.HP > 0 {
			v[0]++
		}
		v[1] += int(r.HP)
		v[2] += int(r.HD)
		v[3] += int(r.HL)
		v[4] += int(r.HR)
		v[5] += int(r.HN)
		app.jMembers[key] = v
	}
	app.jSum[5] = app.jSum[0] + app.jSum[1] + app.jSum[2] + app.jSum[3] + app.jSum[4]
	app.jExcelMsg = ""
	app.screen = scrJournalView
}

// journalView mirrors top_aff: title, monospace lines, Excel button.
func journalView() {
	lines := buildJournalLines()
	Container(Attrs(Expand, Pad(16), Gap(10), Background(0, 0, 100, 1)), func() {
		title := "BS de " + nomBS + " : Visualisation du journal entre le " + app.jD1 + " et le  " + app.jD2
		Container(Attrs(Expand, FixHeight(40), Center, Background(220, 5, 97, 1), Corners(3)), func() {
			Label(title, FontSize(13), TextColor(0, 0, 12, 1))
		})

		Container(Attrs(Expand, FixHeight(420), Gap(10)), func() {
			Container(Attrs(Grow(1), Expand, Corners(3), Background(0, 0, 100, 1)), func() {
				widgets.VirtualListView("journal", len(lines),
					func(i int) any { return i },
					func(i int, w float32) float32 { return 17 },
					func(i int, w float32) {
						l := lines[i]
						if l.red {
							Label(l.text, FontSize(11), TextColor(0, 80, 45, 1), Fonts(Monospace...))
						} else {
							Label(l.text, FontSize(11), TextColor(0, 0, 15, 1), Fonts(Monospace...))
						}
					})
			})
		})

		Container(Attrs(Row, Gap(8), CrossMid), func() {
			switch app.jMode {
			case 1:
				if widgets.Button(widgets.NoIcon, "Fichier Excel détail") && len(app.jRows) > 0 {
					runExcelDetail()
				}
			case 2:
				if widgets.Button(widgets.NoIcon, "Fichier Excel récap") && len(app.jRows) > 0 {
					runExcelRecap()
				}
			case 3:
				if widgets.Button(widgets.NoIcon, "Fichier Excel indiv.") && len(app.jMembers) > 0 {
					runExcelIndiv()
				}
			}
			if widgets.Button(widgets.NoIcon, "Fermer") {
				app.screen = scrMain
			}
			if app.jExcelMsg != "" {
				Label(app.jExcelMsg, FontSize(11), TextColor(120, 70, 30, 1), Fonts(Monospace...))
			}
		})
	})
}

type journalLine struct {
	text string
	red  bool
}

// buildJournalLines reproduces the exact listbox text of j() for the 3 modes.
func buildJournalLines() []journalLine {
	var out []journalLine
	if len(app.jRows) == 0 {
		return []journalLine{{text: "Aucun enregistrement détecté avec ces critères", red: true}}
	}

	switch app.jMode {
	case 1:
		out = append(out, journalLine{text: fmt.Sprintf("%d  enregistrement(s) détecté(s)", len(app.jRows))})
		out = append(out, journalLine{text: fmt.Sprintf("%-11s%-20s%-20s%6s%6s%6s%6s%6s %s",
			padCenter("Date", 11), padCenter("Nom", 20), padCenter("Prénom", 20),
			"Perm.", "Ext.", "Lect.", "Rég.", "Nat.", "Commentaire ")})
		for _, r := range app.jRows {
			comm := r.Comment
			if comm == "" {
				comm = " "
			}
			out = append(out, journalLine{
				text: fmt.Sprintf("%-11s%-20s%-20s%6d%6d%6d%6d%6d %s",
					r.DateFR, r.Nom, r.Prenom, r.HP, r.HD, r.HL, r.HR, r.HN, trimRight(comm, 30)),
				red: r.HP < 0 || r.HD < 0 || r.HL < 0 || r.HR < 0 || r.HN < 0,
			})
		}
	case 2:
		out = append(out, journalLine{text: "Récapitulatif des heures de bénévolat "})
		out = append(out, journalLine{text: "Entre le " + app.jD1 + " et le " + app.jD2})
		out = append(out, journalLine{text: fmt.Sprintf("%d  enregistrement(s) détecté(s)", len(app.jRows))})
		out = append(out, journalLine{text: fmt.Sprintf("%-17s %5d", "Heures permanence", app.jSum[0])})
		out = append(out, journalLine{text: fmt.Sprintf("%-17s %5d", "Heures extérieur", app.jSum[1])})
		out = append(out, journalLine{text: fmt.Sprintf("%-17s %5d", "Heures lecture", app.jSum[2])})
		out = append(out, journalLine{text: fmt.Sprintf("%-17s %5d", "Heures région", app.jSum[3])})
		out = append(out, journalLine{text: fmt.Sprintf("%-17s %5d", "Heures national", app.jSum[4])})
		out = append(out, journalLine{text: fmt.Sprintf("%-17s %5d", "Total général", app.jSum[5])})
	case 3:
		out = append(out, journalLine{text: "Comptages individuels :"})
		out = append(out, journalLine{text: fmt.Sprintf("%-30s %6s %6s %6s %6s %6s %6s %s",
			padCenter("NOM - Prénom", 30), "Nb per", "H.perm", "H.ext.", "H.lect.", "H.rég.", "H.nat.", "TOT.")})
		keys := make([]string, 0, len(app.jMembers))
		for k := range app.jMembers {
			keys = append(keys, k)
		}
		sortStrings(keys)
		for _, k := range keys {
			v := app.jMembers[k]
			tot := v[1] + v[2] + v[3] + v[4] + v[5]
			out = append(out, journalLine{
				text: fmt.Sprintf("%-30s %6d %6d %6d %6d %6d %6d %6d",
					k, v[0], v[1], v[2], v[3], v[4], v[5], tot)})
		}
	}
	return out
}

func runExcelDetail() {
	p, err := exportDetail(app.jRows)
	if err != nil {
		app.jExcelMsg = "IMPOSSIBLE D'ÉCRIRE LE FICHIER '" + p + "'"
		return
	}
	app.jExcelMsg = "ÉCRIT : " + p
}

func runExcelRecap() {
	p, err := exportRecap(app.jD1, app.jD2, app.jSum[0], app.jSum[1], app.jSum[2], app.jSum[3], app.jSum[4], app.jSum[5])
	if err != nil {
		app.jExcelMsg = "IMPOSSIBLE D'ÉCRIRE LE FICHIER '" + p + "'"
		return
	}
	app.jExcelMsg = "ÉCRIT : " + p
}

func runExcelIndiv() {
	p, err := exportIndiv(app.jD1, app.jD2, app.jMembers)
	if err != nil {
		app.jExcelMsg = "IMPOSSIBLE D'ÉCRIRE LE FICHIER '" + p + "'"
		return
	}
	app.jExcelMsg = "ÉCRIT : " + p
}

func trimRight(s string, w int) string {
	if len(s) > w {
		return s[:w]
	}
	return s
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
