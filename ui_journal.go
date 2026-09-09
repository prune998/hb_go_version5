package main

import (
	"fmt"
	"time"

	. "go.hasen.dev/shirei"
	. "go.hasen.dev/shirei/widgets"
)

// JournalScreen is the state of "Edition du journal": the period and the
// kind of output, then the result on screen and the workbook on disk.
type JournalScreen struct {
	step int // 0 = paramètres, 1 = édition
	kind ReportKind
	from string
	to   string
	err  string

	rows      []*Entry
	recap     Recap
	indiv     []*IndivRow
	excelPath string
	excelErr  string
}

var journalSteps = []string{"Paramètres", "Consultation", "Fichier Excel"}

// openJournal starts the journal edition, over the current month by default.
func (a *App) openJournal() {
	now := today()
	a.journal = JournalScreen{
		kind: ReportDetail,
		from: formatFR(time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)),
		to:   formatFR(now),
	}
	a.screen = ScreenJournal
}

// JournalScreenView draws the current step.
func JournalScreenView() {
	j := &app.journal

	PageFill(func() {
		current := j.step
		if j.step == 1 && j.excelPath != "" {
			current = 2
		}
		WorkflowHeader(SymFile, "Edition du journal", j.subtitle(), journalSteps, current)

		if j.step == 0 {
			JournalParamsStep(j)
			return
		}
		JournalResultStep(j)
	})
}

func (j *JournalScreen) subtitle() string {
	if j.step == 0 {
		return "Choisissez le type de sortie et l'encadrement de dates."
	}
	return fmt.Sprintf("%s · du %s au %s", j.kind.Label(), j.from, j.to)
}

// ------------------------------------------------------- step 1: paramètres

func JournalParamsStep(j *JournalScreen) {
	Card(func() {
		CardHeader(SymCog, "Étape 1 — Paramètres", "La vraisemblance des dates d'encadrement est contrôlée.")

		Field("Type de sortie", j.kind.Help(), func() {
			SegmentedControl(&j.kind,
				Cell(ReportDetail.Label(), ReportDetail),
				Cell(ReportRecap.Label(), ReportRecap),
				Cell(ReportIndiv.Label(), ReportIndiv),
			)
		})
		Field("Entre le", "jj/mm/aaaa", func() { DateField(&j.from) })
		Field("Et le", "jj/mm/aaaa", func() { DateField(&j.to) })

		if j.err != "" {
			ErrorBanner(j.err)
		}

		Divider()

		Container(Attrs(Row, Expand, CrossMid, Gap(10)), func() {
			if GhostButton(SymCancel, "Annuler") {
				app.GoHome()
			}
			Filler(1)
			if PrimaryButton(SymRight, "Éditer") {
				j.generate()
			}
		})
	})
}

// ------------------------------------------------------- step 2: édition

func JournalResultStep(j *JournalScreen) {
	Card(func() {
		Container(Attrs(Row, Expand, CrossMid, Gap(12), Wrap), func() {
			Container(Attrs(Gap(2)), func() {
				Label(fmt.Sprintf("%d écriture(s) détectée(s)", len(j.rows)),
					FontSize(14), FontWeight(WeightBold), TextColorVec(colInk))
				Label("BS de "+app.BSName()+" · du "+j.from+" au "+j.to,
					FontSize(12), TextColorVec(colInkSoft))
			})
			Filler(1)
			if GhostButton(SymLeft, "Nouvelle édition") {
				j.step = 0
			}
			if j.excelPath != "" {
				if PrimaryButton(TypExport, j.excelButtonLabel()) {
					if err := openInDesktop(j.excelPath); err != nil {
						app.NotifyError("Fichier Excel", "Impossible d'ouvrir « "+j.excelPath+" ».")
					}
				}
			}
		})

		if j.excelErr != "" {
			ErrorBanner(j.excelErr)
		} else if j.excelPath != "" {
			Label("Fichier écrit : "+j.excelPath, FontSize(11), TextColorVec(colInkSoft))
		}
	})

	if len(j.rows) == 0 {
		Card(func() {
			ErrorBanner("Aucun enregistrement détecté avec ces critères.")
		})
		return
	}

	switch j.kind {
	case ReportRecap:
		JournalRecapView(j)
	case ReportIndiv:
		JournalIndivView(j)
	default:
		JournalDetailView(j)
	}
}

func (j *JournalScreen) excelButtonLabel() string {
	switch j.kind {
	case ReportRecap:
		return "Fichier Excel récap"
	case ReportIndiv:
		return "Fichier Excel indiv."
	default:
		return "Fichier Excel détail"
	}
}

// JournalDetailView lists every line of the period.
func JournalDetailView(j *JournalScreen) {
	Pane("Journal détaillé", func() {
		Label("Les lignes en rouge sont des rectifications.", FontSize(11), TextColorVec(colInkSoft))
	}, func() {
		Container(Attrs(Grow(1), Expand, Extrinsic, Clip), func() {
			TableExt("journal-detail", TableAttrs[*Entry]{
				RowHeight: 28,
				OnRow: func(index int, e *Entry) {
					if index%2 == 1 {
						ModAttrs(BackgroundVec(colZebra))
					}
					if e.IsCorrection() {
						ModAttrs(BackgroundVec(colDangerSoft))
					}
				},
			}, detailColumns(), j.rows, func(e *Entry) any { return e })
		})
	})
}

func detailColumns() []TableColumn[*Entry] {
	hours := func(label string, value func(*Entry) int) TableColumn[*Entry] {
		return TableColumn[*Entry]{
			Label: label, Width: 62, DefaultDesc: true,
			Render: func(e *Entry) { HoursLabel(value(e)) },
			Less:   func(a, b *Entry) bool { return value(a) < value(b) },
		}
	}

	return []TableColumn[*Entry]{
		{
			Label: "Date", Width: 92,
			Render: func(e *Entry) {
				Label(formatFR(e.Date), FontSize(12), TextColorVec(colInk), Fonts(Monospace...))
			},
			Less: func(a, b *Entry) bool { return a.Date.Before(b.Date) },
		},
		{
			Label: "Nom", Width: 180,
			Render: func(e *Entry) { Label(e.Nom, FontSize(12), FontWeight(WeightBold), TextColorVec(colInk)) },
			Less:   func(a, b *Entry) bool { return a.Nom < b.Nom },
		},
		{
			Label: "Prénom", Width: 150,
			Render: func(e *Entry) { Label(e.Prenom, FontSize(12), TextColorVec(colInk)) },
			Less:   func(a, b *Entry) bool { return a.Prenom < b.Prenom },
		},
		hours("Perm.", func(e *Entry) int { return e.HP }),
		hours(CatExterieur.Short(), func(e *Entry) int { return e.HD }),
		hours(CatLectureDDV.Short(), func(e *Entry) int { return e.HL }),
		hours(CatRegional.Short(), func(e *Entry) int { return e.HR }),
		hours(CatNational.Short(), func(e *Entry) int { return e.HN }),
		{
			Label:  "Commentaire",
			Render: func(e *Entry) { Label(e.Commentaire, FontSize(11), TextColorVec(colInkSoft)) },
		},
	}
}

// JournalRecapView shows the totals of the period.
func JournalRecapView(j *JournalScreen) {
	Card(func() {
		CardHeader(SymChartBar, "Récapitulatif des heures de bénévolat", "")
		Container(Attrs(Row, Expand, Gap(12), Wrap), func() {
			StatTile("Heures permanence", fmt.Sprintf("%d", j.recap.HP), "")
			StatTile("Heures extérieur", fmt.Sprintf("%d", j.recap.HD), "")
			StatTile("Heures lecture", fmt.Sprintf("%d", j.recap.HL), "")
			StatTile("Heures région", fmt.Sprintf("%d", j.recap.HR), "")
			StatTile("Heures national", fmt.Sprintf("%d", j.recap.HN), "")
		})
		Divider()
		Container(Attrs(Row, Expand, CrossMid, Gap(12)), func() {
			Label("Total général", FontSize(15), FontWeight(WeightBold), TextColorVec(colInk))
			Filler(1)
			Label(fmt.Sprintf("%d heures", j.recap.Total()),
				FontSize(22), FontWeight(WeightBold), TextColorVec(colAccent))
		})
	})
}

// JournalIndivView shows one line per volunteer.
func JournalIndivView(j *JournalScreen) {
	Pane(fmt.Sprintf("Comptages individuels (%d bénévoles)", len(j.indiv)), nil, func() {
		Container(Attrs(Grow(1), Expand, Extrinsic, Clip), func() {
			TableExt("journal-indiv", TableAttrs[*IndivRow]{
				RowHeight: 28,
				OnRow: func(index int, _ *IndivRow) {
					if index%2 == 1 {
						ModAttrs(BackgroundVec(colZebra))
					}
				},
			}, indivColumns(), j.indiv, func(r *IndivRow) any { return r })
		})
	})
}

func indivColumns() []TableColumn[*IndivRow] {
	hours := func(label string, value func(*IndivRow) int) TableColumn[*IndivRow] {
		return TableColumn[*IndivRow]{
			Label: label, Width: 66, DefaultDesc: true,
			Render: func(r *IndivRow) { HoursLabel(value(r)) },
			Less:   func(a, b *IndivRow) bool { return value(a) < value(b) },
		}
	}

	return []TableColumn[*IndivRow]{
		{
			Label:  "Nom Prénom",
			Render: func(r *IndivRow) { Label(r.Person, FontSize(12), TextColorVec(colInk)) },
			Less:   func(a, b *IndivRow) bool { return a.Person < b.Person },
		},
		{
			Label: "Nb perm.", Width: 80, DefaultDesc: true,
			Render: func(r *IndivRow) { HoursLabel(r.NbPerm) },
			Less:   func(a, b *IndivRow) bool { return a.NbPerm < b.NbPerm },
		},
		hours("Perm.", func(r *IndivRow) int { return r.HP }),
		hours(CatExterieur.Short(), func(r *IndivRow) int { return r.HD }),
		hours(CatLectureDDV.Short(), func(r *IndivRow) int { return r.HL }),
		hours(CatRegional.Short(), func(r *IndivRow) int { return r.HR }),
		hours(CatNational.Short(), func(r *IndivRow) int { return r.HN }),
		{
			Label: "Total", Width: 80, DefaultDesc: true,
			Render: func(r *IndivRow) {
				Label(fmt.Sprintf("%d", r.Hours()), FontSize(12), FontWeight(WeightBold),
					TextColorVec(colAccent), Fonts(Monospace...))
			},
			Less: func(a, b *IndivRow) bool { return a.Hours() < b.Hours() },
		},
	}
}

// ------------------------------------------------------------------ logic

// generate reads the journal for the period, builds the requested output and
// writes the matching workbook.
func (j *JournalScreen) generate() {
	if app.store == nil {
		return
	}

	from, to, err := parseDateRange(j.from, j.to)
	if err != nil {
		j.err = err.Error()
		return
	}

	entries, err := app.store.EntriesBetween(from, to)
	if err != nil {
		j.err = err.Error()
		return
	}

	j.err = ""
	j.rows = make([]*Entry, len(entries))
	for i := range entries {
		j.rows[i] = &entries[i]
	}
	j.recap = BuildRecap(entries)
	j.indiv = BuildIndiv(entries)
	j.excelPath, j.excelErr = "", ""
	j.step = 1

	if len(entries) == 0 {
		return
	}

	var (
		path     string
		exportEr error
	)
	switch j.kind {
	case ReportRecap:
		path = app.paths.Export(fileJournalRecap)
		exportEr = exportRecap(path, app.data.BS, j.from, j.to, j.recap)
	case ReportIndiv:
		path = app.paths.Export(fileJournalIndiv)
		exportEr = exportIndiv(path, app.data.BS, j.from, j.to, j.indiv)
	default:
		path = app.paths.Export(fileJournalDetail)
		exportEr = exportDetail(path, app.data.BS, entries)
	}

	if exportEr != nil {
		j.excelErr = exportEr.Error()
		return
	}
	j.excelPath = path
}
