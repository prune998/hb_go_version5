package main

import (
	"fmt"
	"strconv"
	"strings"

	. "go.hasen.dev/shirei"
	"go.hasen.dev/shirei/widgets"
)

func parseDuree(s string) (int, error) {
	return strconv.Atoi(strings.TrimSpace(s))
}

// benLine reproduces the left listbox row: " nom<20  prenom<20 donneur:3".
func benLine(b Benevole) string {
	return " " + padRightRunes(b.Nom, 20) + "  " + padRightRunes(b.Prenom, 20) + " " + padRightRunes(b.Donneur, 3)
}

// benList renders the BEN list; a click on a row calls onPick with its index.
func benList(key string, onPick func(int)) {
	Container(Attrs(FixWidth(330), Expand, Corners(3), Background(220, 8, 98, 1)), func() {
		widgets.VirtualListView(key, len(bens),
			func(i int) any { return i },
			func(i int, w float32) float32 { return 16 },
			func(i int, w float32) {
				Container(Attrs(Expand, CrossMid, Pad2(2, 8), FixHeight(16), NoAnimate), func() {
					if IsHovered() {
						ModAttrs(Background(220, 20, 90, 1))
					}
					if PressAction() {
						onPick(i)
					}
					Label(benLine(bens[i]), FontSize(11), TextColor(0, 0, 15, 1), Fonts(Monospace...))
				})
			})
	})
}

// linesBox renders the "selected" listbox (blue monospace lines, red allowed).
func linesBox(key string, lines []journalLine) {
	Container(Attrs(Grow(1), Expand, Corners(3), Background(220, 8, 98, 1)), func() {
		widgets.VirtualListView(key, len(lines),
			func(i int) any { return i },
			func(i int, w float32) float32 { return 16 },
			func(i int, w float32) {
				Container(Attrs(Expand, Pad2(4, 8), FixHeight(16), NoAnimate), func() {
					l := lines[i]
					if l.red {
						Label(l.text, FontSize(11), TextColor(0, 80, 45, 1), Fonts(Monospace...))
					} else {
						Label(l.text, FontSize(11), TextColor(225, 95, 45, 1), Fonts(Monospace...))
					}
				})
			})
	})
}

// permAskView mirrors a(): date + durée standard + Valider.
func permAskView() {
	Container(Attrs(Expand, Pad(20), Gap(12), Background(220, 5, 97, 1)), func() {
		Label("Indiquez la date choisie : \n (par défaut : la date du jour )", FontSize(12), TextColor(0, 0, 15, 1))
		Container(Attrs(CrossMid, FixWidth(140)), func() {
			Container(Attrs(FixWidth(110), Background(0, 0, 100, 1), Corners(3)), func() {
				widgets.TextInput(&app.permDate)
			})
		})
		Label("Indiquez la durée standard\npour la permanence ( en heures entières): \n (par défaut : 3 h)", FontSize(12), TextColor(0, 0, 15, 1))
		Container(Attrs(CrossMid, FixWidth(140)), func() {
			Container(Attrs(FixWidth(60), Background(0, 0, 100, 1), Corners(3)), func() {
				widgets.TextInput(&app.permDuree)
			})
		})
		Container(Attrs(CrossMid, FixWidth(140)), func() {
			if widgets.Button(widgets.NoIcon, "Valider") {
				validatePermAsk()
			}
		})
	})
}

// validatePermAsk mirrors verification_saisie().
func validatePermAsk() {
	if msg := validateDateFR(app.permDate); msg != "" {
		app.showErr("ERREUR", msg)
		return
	}
	n, err := parseDuree(app.permDuree)
	if err != nil {
		app.showErr("ERREUR", "SAISIE DURÉE ERRONÉE")
		return
	}
	app.permDureeInt = n
	app.permDateISO = frToISO(app.permDate)
	app.permSel = nil
	app.screen = scrPermSelect
}

// permSelectView mirrors permanence(): titles, BEN list, selected list, buttons.
func permSelectView() {
	Container(Attrs(Expand, Pad(16), Gap(10), Background(0, 0, 100, 1)), func() {
		Container(Attrs(Expand, FixHeight(56), Center, Background(220, 5, 97, 1), Corners(3)), func() {
			Label("BS de "+nomBS+" \nSaisie des heures de permanence \n Sélectionnez les bénévoles concernés dans la liste de gauche...",
				FontSize(13), TextColor(0, 0, 12, 1))
		})
		Container(Attrs(Expand, Row, FixHeight(300), Gap(10)), func() {
			benList("perm-bens", pickPerm)
			linesBox("perm-sel", permSelLines())
		})
		Container(Attrs(Expand, Row, Gap(12), CrossMid, Background(220, 5, 97, 1), Corners(3), Pad2(10, 6)), func() {
			if widgets.Button(widgets.NoIcon, "Valider") {
				runPermGo()
			}
			if widgets.Button(widgets.NoIcon, "Effacer la sélection") {
				app.permSel = nil
			}
		})
	})
}

// pickPerm mirrors selection(): appends the member unless already present.
func pickPerm(i int) {
	for _, v := range app.permSel {
		if v == i {
			return
		}
	}
	app.permSel = append(app.permSel, i)
}

// permSelLines reproduces the exact listbox text of permanence()/selection().
func permSelLines() []journalLine {
	out := []journalLine{
		{text: "LES BÉNÉVOLES SÉLECTIONNÉS"},
		{text: "SERONT CRÉDITÉS DE " + app.permDuree + " HEURES"},
		{text: " "},
		{text: " "},
		{text: "\n"},
	}
	for _, idx := range app.permSel {
		b := bens[idx]
		out = append(out, journalLine{
			text: fmt.Sprintf("%s %s %s %2.0f h.",
				padRightRunes(b.Nom, 20), padRightRunes(b.Prenom, 15), padRightRunes(app.permDateISO, 15), float64(app.permDureeInt)),
			red: app.permDureeInt < 0,
		})
	}
	return out
}

// runPermGo mirrors go() + maj_journal().
func runPermGo() {
	var err error
	if len(app.permSel) > 0 {
		err = insertPermanence(app.permDateISO, app.permDureeInt, app.permSel)
	}
	app.permSel = nil
	app.screen = scrMain
	if err != nil {
		app.showErr("ERREUR", "LA BASE N'A PAS PU ÊTRE MISE À JOUR. ")
		return
	}
	app.showInfo("MESSAGE", "LA BASE A ÉTÉ MISE À JOUR. ")
}
