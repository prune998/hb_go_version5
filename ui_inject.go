package main

import (
	"fmt"

	. "go.hasen.dev/shirei"
	. "go.hasen.dev/shirei/widgets"
)

// InjectScreen is the state of "Consolidation par journal externe": the
// detailed journal of another installation is read, checked, then merged
// into the local journal.
type InjectScreen struct {
	step   int // 0 = fichier, 1 = contrôle
	path   string
	rows   []*Entry
	recap  Recap
	backup bool
	err    string
}

var injectSteps = []string{"Fichier", "Contrôle", "Injection"}

func (a *App) openInject() {
	a.inject = InjectScreen{backup: true}
	a.screen = ScreenInject
}

// InjectScreenView draws the current step.
func InjectScreenView() {
	in := &app.inject

	PageFill(func() {
		WorkflowHeader(SymExternal, "Consolidation par journal externe", in.subtitle(), injectSteps, in.step)

		if in.step == 0 {
			InjectFileStep(in)
			return
		}
		InjectCheckStep(in)
	})
}

func (in *InjectScreen) subtitle() string {
	if in.step == 0 {
		return "Injecter le journal détaillé Excel enregistré sur un autre poste."
	}
	return fmt.Sprintf("%d écriture(s) lue(s) dans %s", len(in.rows), in.path)
}

// ---------------------------------------------------------- step 1: fichier

func InjectFileStep(in *InjectScreen) {
	Card(func() {
		CardHeader(SymCog, "Étape 1 — Choix du fichier",
			"Le fichier attendu est un « Journal_benevolat.xlsx » produit par l'édition du journal détaillé.")

		InfoBanner("Par prudence : renommez le fichier à injecter pour ne pas injecter votre propre journal, et sauvegardez la base avant l'opération.")

		Field("Fichier à injecter", "la cellule A1 doit contenir « "+detailHeader+" »", func() {
			Container(Attrs(Row, Expand, CrossMid, Gap(10), Wrap), func() {
				if PrimaryButton(SymFolder, "Parcourir …") {
					in.choose()
				}
				if in.path != "" {
					Label(in.path, FontSize(12), TextColorVec(colInkSoft))
				} else {
					Label("Aucun fichier choisi", FontSize(12), TextColorVec(colInkSoft))
				}
			})
		})

		Field("Sauvegarde", "recommandée avant toute injection", func() {
			CheckBox(&in.backup, "Sauvegarder la base avant l'injection")
		})

		if in.err != "" {
			ErrorBanner(in.err)
		}

		Divider()

		Container(Attrs(Row, Expand, CrossMid, Gap(10)), func() {
			if GhostButton(SymCancel, "Annuler") {
				app.GoHome()
			}
			Filler(1)
			if PrimaryButtonEnabled(SymRight, "Contrôler le fichier", in.path != "") {
				in.analyse()
			}
		})
	})
}

// choose opens the shared file picker.
func (in *InjectScreen) choose() {
	app.picker.Open("Sélectionner le journal Excel à injecter", pickerStartDir(), []string{".xlsx"},
		func(path string) {
			in.path = path
			in.err = ""
			in.analyse()
		})
}

// ---------------------------------------------------------- step 2: contrôle

func InjectCheckStep(in *InjectScreen) {
	Card(func() {
		Container(Attrs(Row, Expand, CrossMid, Gap(12), Wrap), func() {
			Container(Attrs(Gap(2)), func() {
				Label(fmt.Sprintf("%d écriture(s) prête(s) à être injectée(s)", len(in.rows)),
					FontSize(14), FontWeight(WeightBold), TextColorVec(colInk))
				Label(fmt.Sprintf("Total : %d heures · fichier %s", in.recap.Total(), in.path),
					FontSize(12), TextColorVec(colInkSoft))
			})
			Filler(1)
			if GhostButton(SymLeft, "Choisir un autre fichier") {
				in.step = 0
			}
			if PrimaryButton(SymDownload, "Injecter dans le journal") {
				in.commit()
			}
		})
		if in.backup {
			Label("Une sauvegarde de la base sera effectuée juste avant l'injection.",
				FontSize(11), TextColorVec(colInkSoft))
		}
	})

	Pane("Contrôle du journal à injecter", nil, func() {
		Container(Attrs(Grow(1), Expand, Extrinsic, Clip), func() {
			TableExt("inject-rows", TableAttrs[*Entry]{
				RowHeight: 28,
				OnRow: func(index int, e *Entry) {
					if index%2 == 1 {
						ModAttrs(BackgroundVec(colZebra))
					}
					if e.IsCorrection() {
						ModAttrs(BackgroundVec(colDangerSoft))
					}
				},
			}, detailColumns(), in.rows, func(e *Entry) any { return e })
		})
	})
}

// ------------------------------------------------------------------ logic

// analyse reads and checks the workbook without touching the database.
func (in *InjectScreen) analyse() {
	entries, err := importJournal(in.path)
	if err != nil {
		in.err = err.Error()
		in.rows = nil
		in.step = 0
		return
	}

	in.err = ""
	in.rows = make([]*Entry, len(entries))
	for i := range entries {
		in.rows[i] = &entries[i]
	}
	in.recap = BuildRecap(entries)
	in.step = 1
}

// commit appends the external journal to the local one.
func (in *InjectScreen) commit() {
	if app.store == nil || len(in.rows) == 0 {
		return
	}

	if in.backup {
		if err := backupDatabase(); err != nil {
			app.NotifyError("Sauvegarde", err.Error())
			return
		}
	}

	entries := make([]Entry, 0, len(in.rows))
	for _, row := range in.rows {
		entries = append(entries, *row)
	}

	if err := app.store.AddEntries(entries); err != nil {
		app.NotifyError("La base n'a pas pu être mise à jour", err.Error())
		return
	}

	app.Reload()
	app.Notify("La base a été mise à jour",
		fmt.Sprintf("%d enregistrement(s) ajouté(s) au journal.", len(entries)))
	app.GoHome()
}
