package main

import (
	"fmt"

	. "go.hasen.dev/shirei"
	. "go.hasen.dev/shirei/widgets"
)

// EntryKind is one of the three hour entry workflows of the "Action" menu.
// They share this screen, and therefore share their workflow: parameters,
// then selection of the volunteers, then validation.
type EntryKind int

const (
	KindPermanence EntryKind = iota
	KindIndividuelle
	KindConsolidation
)

func (k EntryKind) Title() string {
	switch k {
	case KindPermanence:
		return "Saisie heures permanence"
	case KindIndividuelle:
		return "Saisie heures individuelles"
	case KindConsolidation:
		return "Saisie globale de consolidation"
	}
	return ""
}

func (k EntryKind) Icon() IconGlyph {
	switch k {
	case KindPermanence:
		return SymGroup
	case KindIndividuelle:
		return SymUser
	case KindConsolidation:
		return SymGrid
	}
	return NoIcon
}

func (k EntryKind) Hint() string {
	switch k {
	case KindPermanence:
		return "Tous les bénévoles présents sont crédités de la même durée."
	case KindIndividuelle:
		return "Heures hors permanence, une catégorie à la fois."
	case KindConsolidation:
		return "Report d'un récapitulatif d'activité, toutes catégories en une saisie."
	}
	return ""
}

// entryForm holds the text fields of the parameters step and of the line
// being prepared, kept as strings so what the user typed is what is
// validated.
type entryForm struct {
	date    string
	hours   string
	cat     Category
	hp      string
	hd      string
	hl      string
	hr      string
	hn      string
	comment string
}

// PendingLine is one line waiting to be written to the journal.
type PendingLine struct {
	Ben   *Benevole
	Cat   Category
	Entry Entry
}

// EntryScreen is the state of the three entry workflows.
type EntryScreen struct {
	kind   EntryKind
	step   int // 0 = paramètres, 1 = saisie
	params entryForm
	err    string

	filter string
	lines  []*PendingLine

	formOpen bool
	formBen  *Benevole
	form     entryForm
	formErr  string
}

var entrySteps = []string{"Paramètres", "Saisie", "Enregistrement"}

// openEntry starts one of the entry workflows.
func (a *App) openEntry(kind EntryKind) {
	a.entry = EntryScreen{
		kind: kind,
		params: entryForm{
			date:  formatFR(today()),
			hours: "3",
			hp:    "0", hd: "0", hl: "0", hr: "0", hn: "0",
		},
	}
	a.screen = ScreenEntry
}

// EntryScreenView draws the current step of the workflow.
func EntryScreenView() {
	e := &app.entry

	PageFill(func() {
		WorkflowHeader(e.kind.Icon(), e.kind.Title(), e.subtitle(), entrySteps, e.step)

		if e.step == 0 {
			EntryParamsStep(e)
			return
		}
		EntrySelectionStep(e)
	})

	EntryFormDialog(e)
}

// subtitle recalls the chosen parameters once the first step is done.
func (e *EntryScreen) subtitle() string {
	if e.step == 0 {
		return e.kind.Hint()
	}
	switch e.kind {
	case KindPermanence:
		return fmt.Sprintf("Permanence du %s · %s h par bénévole", e.params.date, e.params.hours)
	case KindIndividuelle:
		return fmt.Sprintf("Valeurs proposées : %s · %s · %s h", e.params.date, e.params.cat.Label(), e.params.hours)
	default:
		return "Date proposée : " + e.params.date
	}
}

// ------------------------------------------------------- step 1: paramètres

// EntryParamsStep collects what applies to the whole entry session.
func EntryParamsStep(e *EntryScreen) {
	Card(func() {
		CardHeader(SymCog, "Étape 1 — Paramètres", entryParamsHint(e.kind))

		Field("Date", "jour de l'activité, au format jj/mm/aaaa", func() {
			DateField(&e.params.date)
		})

		switch e.kind {
		case KindPermanence:
			Field("Durée de la permanence", "en heures entières, 3 h par défaut", func() {
				HoursField(&e.params.hours)
			})
		case KindIndividuelle:
			Field("Catégorie proposée", e.params.cat.Help(), func() {
				SegmentedControl(&e.params.cat, categoryCells()...)
			})
			Field("Durée proposée", "en heures entières", func() {
				HoursField(&e.params.hours)
			})
		}

		if e.err != "" {
			ErrorBanner(e.err)
		}

		Divider()

		Container(Attrs(Row, Expand, CrossMid, Gap(10)), func() {
			if GhostButton(SymCancel, "Annuler") {
				app.GoHome()
			}
			Filler(1)
			if PrimaryButton(SymRight, "Valider et saisir") {
				e.startSelection()
			}
		})
	})
}

func entryParamsHint(kind EntryKind) string {
	switch kind {
	case KindPermanence:
		return "Les dates erronées sont refusées ; ne changez la durée que si elle diffère de 3 heures."
	case KindIndividuelle:
		return "Ces valeurs sont proposées pour chaque bénévole ; elles restent modifiables ligne à ligne."
	default:
		return "La date est proposée pour chaque récapitulatif ; elle reste modifiable ligne à ligne."
	}
}

// startSelection validates the parameters and moves to the entry step.
func (e *EntryScreen) startSelection() {
	if _, err := parsePastDate(e.params.date); err != nil {
		e.err = err.Error()
		return
	}
	if e.kind != KindConsolidation {
		if _, err := parseHours(e.params.hours); err != nil {
			e.err = err.Error()
			return
		}
	}
	e.err = ""
	e.step = 1
}

// ---------------------------------------------------------- step 2: saisie

// EntrySelectionStep is the two pane screen: the volunteers on the left, the
// lines waiting to be recorded on the right.
func EntrySelectionStep(e *EntryScreen) {
	InfoBanner(entrySelectionHint(e.kind))

	Container(Attrs(Row, Grow(1), Expand, Extrinsic, Clip, Gap(16)), func() {
		Container(Attrs(FixWidth(360), Expand, Clip), func() {
			BenevolesPane(e)
		})
		Container(Attrs(Grow(1), Expand, Extrinsic, Clip), func() {
			PendingLinesPane(e)
		})
	})

	Card(func() {
		Container(Attrs(Row, Expand, CrossMid, Gap(10), Wrap), func() {
			if GhostButton(SymLeft, "Modifier les paramètres") {
				e.step = 0
			}
			if DangerButton(SymDelete, "Effacer la sélection") {
				e.lines = nil
			}
			Filler(1)
			Label(fmt.Sprintf("%d ligne(s) · %d heure(s)", len(e.lines), e.totalHours()),
				FontSize(12), TextColorVec(colInkSoft))
			if PrimaryButtonEnabled(SymPass, "Valider l'enregistrement", len(e.lines) > 0) {
				e.commit()
			}
		})
	})
}

func entrySelectionHint(kind EntryKind) string {
	switch kind {
	case KindPermanence:
		return "Cliquez sur chaque bénévole présent : il est ajouté à droite. Un même bénévole ne peut être saisi qu'une fois par permanence."
	case KindIndividuelle:
		return "Cliquez sur un bénévole pour saisir sa catégorie et sa durée. Un bénévole peut être saisi plusieurs fois."
	default:
		return "Cliquez sur un bénévole pour reporter son récapitulatif d'heures. Un bénévole peut être saisi plusieurs fois."
	}
}

// BenevolesPane lists the volunteers, filtered by the search field.
func BenevolesPane(e *EntryScreen) {
	matches := filterBenevoles(app.data.Benevoles, e.filter)

	Pane(fmt.Sprintf("Bénévoles (%d)", len(matches)), nil, func() {
		Container(Attrs(Expand, Pad2(8, 12), BackgroundVec(colSurface)), func() {
			SearchField(&e.filter)
		})
		Divider()

		if len(matches) == 0 {
			EmptyState(SymSearch, "Aucun bénévole ne correspond à cette recherche.")
			return
		}

		Container(Attrs(Grow(1), Expand, Extrinsic, Clip), func() {
			VirtualListView("entry-benevoles", len(matches),
				func(i int) any { return matches[i] },
				func(i int, _ float32) float32 { return 34 },
				func(i int, _ float32) { BenevoleRow(e, matches[i]) },
			)
		})
	})
}

// BenevoleRow is one clickable volunteer.
func BenevoleRow(e *EntryScreen, b *Benevole) {
	already := e.kind == KindPermanence && e.contains(b)

	Container(Attrs(Row, Expand, CrossMid, Gap(8), Pad2(0, 12), FixHeight(34), NoAnimate), func() {
		switch {
		case already:
			ModAttrs(BackgroundVec(colZebra))
		case IsHovered():
			ModAttrs(BackgroundVec(colAccentSoft))
		}
		if IsClicked() {
			e.pick(b)
		}

		Container(Attrs(Row, Grow(1), Expand, Extrinsic, Clip, CrossMid, Gap(6)), func() {
			Label(b.Nom, FontSize(12), FontWeight(WeightBold), TextColorVec(colInk))
			Label(b.Prenom, FontSize(12), TextColorVec(colInkSoft))
		})
		if b.IsDDV() {
			Tag("DDV", colAccentSoft, colAccent)
		}
		if already {
			Icon(SymPass, FontSize(12), TextColorVec(colOK))
		}
	})
}

// PendingLinesPane shows what will be written to the journal.
func PendingLinesPane(e *EntryScreen) {
	title := fmt.Sprintf("Lignes à enregistrer (%d)", len(e.lines))

	Pane(title, nil, func() {
		if len(e.lines) == 0 {
			EmptyState(SymList, "Sélectionnez un bénévole dans la liste de gauche.")
			return
		}
		Container(Attrs(Grow(1), Expand, Extrinsic, Clip), func() {
			TableExt("entry-lines", TableAttrs[*PendingLine]{
				RowHeight: 30,
				OnRow: func(index int, line *PendingLine) {
					if index%2 == 1 {
						ModAttrs(BackgroundVec(colZebra))
					}
					if line.Entry.IsCorrection() {
						ModAttrs(BackgroundVec(colDangerSoft))
					}
				},
			}, e.columns(), e.lines, func(l *PendingLine) any { return l })
		})
	})
}

// columns describes the pending table, one layout per workflow.
func (e *EntryScreen) columns() []TableColumn[*PendingLine] {
	name := TableColumn[*PendingLine]{
		Label:  "Bénévole",
		Render: func(l *PendingLine) { Label(l.Ben.Person(), FontSize(12), TextColorVec(colInk)) },
	}
	date := TableColumn[*PendingLine]{
		Label: "Date", Width: 90,
		Render: func(l *PendingLine) {
			Label(formatFR(l.Entry.Date), FontSize(12), TextColorVec(colInk), Fonts(Monospace...))
		},
	}
	comment := TableColumn[*PendingLine]{
		Label: "Commentaire", Width: 190,
		Render: func(l *PendingLine) {
			Label(l.Entry.Commentaire, FontSize(11), TextColorVec(colInkSoft))
		},
	}
	remove := TableColumn[*PendingLine]{
		Label: "", Width: 44,
		Render: func(l *PendingLine) {
			if ButtonExt("", ButtonAttrs{Icon: SymDelete, Accent: colDanger, TextSize: 11}, DefaultCtrlButtonLook()) {
				e.remove(l)
			}
		},
	}

	hours := func(label string, width float32, value func(*PendingLine) int) TableColumn[*PendingLine] {
		return TableColumn[*PendingLine]{
			Label: label, Width: width,
			Render: func(l *PendingLine) { HoursLabel(value(l)) },
		}
	}

	switch e.kind {
	case KindPermanence:
		return []TableColumn[*PendingLine]{
			name, date,
			hours("Heures", 70, func(l *PendingLine) int { return l.Entry.HP }),
			remove,
		}
	case KindIndividuelle:
		return []TableColumn[*PendingLine]{
			name, date,
			{
				Label: "Catégorie", Width: 110,
				Render: func(l *PendingLine) { Label(l.Cat.Label(), FontSize(12), TextColorVec(colInk)) },
			},
			hours("Heures", 70, func(l *PendingLine) int { return l.Entry.Category(l.Cat) }),
			comment, remove,
		}
	default:
		return []TableColumn[*PendingLine]{
			name, date,
			hours("Perm.", 60, func(l *PendingLine) int { return l.Entry.HP }),
			hours(CatExterieur.Short(), 60, func(l *PendingLine) int { return l.Entry.HD }),
			hours(CatLectureDDV.Short(), 60, func(l *PendingLine) int { return l.Entry.HL }),
			hours(CatRegional.Short(), 60, func(l *PendingLine) int { return l.Entry.HR }),
			hours(CatNational.Short(), 60, func(l *PendingLine) int { return l.Entry.HN }),
			comment, remove,
		}
	}
}

// ------------------------------------------------------------- line editor

// EntryFormDialog is the per volunteer form of the individual and
// consolidation workflows.
func EntryFormDialog(e *EntryScreen) {
	if !e.formOpen || e.formBen == nil {
		return
	}
	ben := e.formBen

	Modal(e.formWidth(), e.closeForm, func() {
		Container(Attrs(Row, CrossMid, Gap(10)), func() {
			Icon(e.kind.Icon(), FontSize(18), TextColorVec(colAccent))
			Label(ben.Person(), FontSize(16), FontWeight(WeightBold), TextColorVec(colInk))
			if ben.IsDDV() {
				Tag("DDV", colAccentSoft, colAccent)
			}
		})
		Divider()

		Field("Date", "jj/mm/aaaa", func() { DateField(&e.form.date) })

		if e.kind == KindIndividuelle {
			Field("Catégorie", e.form.cat.Help(), func() {
				SegmentedControl(&e.form.cat, categoryCells()...)
			})
			Field("Durée", "en heures entières, négative pour corriger une erreur", func() {
				HoursField(&e.form.hours)
			})
		} else {
			Field("Heures par catégorie", "en heures entières, négatives pour corriger une erreur", func() {
				Container(Attrs(Row, Gap(10), Wrap, MaxWidth(480)), func() {
					hoursBox("Permanence", &e.form.hp)
					hoursBox("Extérieur", &e.form.hd)
					hoursBox("Lecture DDV", &e.form.hl)
					hoursBox("Régional", &e.form.hr)
					hoursBox("National", &e.form.hn)
				})
			})
		}

		Field("Commentaire", "facultatif, 30 caractères au plus", func() {
			TextField(&e.form.comment, 320, "")
		})

		if e.formErr != "" {
			ErrorBanner(e.formErr)
		}

		Container(Attrs(Row, Expand, CrossMid, Gap(10), Pad4(6, 0, 0, 0)), func() {
			Filler(1)
			if GhostButton(SymCancel, "Annuler") {
				e.closeForm()
				return
			}
			if PrimaryButton(SymPlus, "Ajouter à la liste") {
				e.submitForm()
			}
		})
	})
}

// hoursBox is one labelled hour field of the consolidation form.
func hoursBox(label string, buf *string) {
	Container(Attrs(Gap(4), FixWidth(88)), func() {
		Label(label, FontSize(11), FontWeight(WeightBold), TextColorVec(colInkSoft))
		HoursField(buf)
	})
}

// formWidth: the consolidation form holds five hour fields side by side and
// needs a wider card than the individual one.
func (e *EntryScreen) formWidth() float32 {
	if e.kind == KindConsolidation {
		return 780
	}
	return 620
}

func categoryCells() []SegmentedCell[Category] {
	cells := make([]SegmentedCell[Category], 0, len(allCategories))
	for _, c := range allCategories {
		cells = append(cells, Cell(c.Label(), c))
	}
	return cells
}

// ------------------------------------------------------------------ logic

// pick reacts to a click on a volunteer: the permanence workflow records the
// line straight away, the two others open the line editor.
func (e *EntryScreen) pick(b *Benevole) {
	if e.kind == KindPermanence {
		e.addPermanence(b)
		return
	}
	e.formOpen = true
	e.formBen = b
	e.formErr = ""
	e.form = e.params
}

func (e *EntryScreen) closeForm() {
	e.formOpen = false
	e.formBen = nil
	e.formErr = ""
}

// addPermanence credits one volunteer with the standard duration.
func (e *EntryScreen) addPermanence(b *Benevole) {
	if e.contains(b) {
		app.NotifyInfo("Bénévole déjà saisi",
			b.Person()+" est déjà crédité de cette permanence.")
		return
	}
	date, err := parsePastDate(e.params.date)
	if err != nil {
		e.step = 0
		e.err = err.Error()
		return
	}
	hours, err := parseHours(e.params.hours)
	if err != nil {
		e.step = 0
		e.err = err.Error()
		return
	}

	e.lines = append(e.lines, &PendingLine{
		Ben: b,
		Entry: Entry{
			Date:   date,
			Nom:    b.Nom,
			Prenom: b.Prenom,
			HP:     hours,
		},
	})
}

// submitForm validates the line editor and adds the line.
func (e *EntryScreen) submitForm() {
	ben := e.formBen
	if ben == nil {
		return
	}

	date, err := parsePastDate(e.form.date)
	if err != nil {
		e.formErr = err.Error()
		return
	}

	entry := Entry{
		Date:        date,
		Nom:         ben.Nom,
		Prenom:      ben.Prenom,
		Commentaire: truncateComment(e.form.comment),
	}

	if e.kind == KindIndividuelle {
		hours, err := parseHours(e.form.hours)
		if err != nil {
			e.formErr = err.Error()
			return
		}
		if e.form.cat == CatLectureDDV && !ben.IsDDV() {
			e.formErr = "Les heures de lecture sont réservées aux DDV : " + ben.Person() + " ne l'est pas."
			return
		}
		entry.SetCategory(e.form.cat, hours)
	} else {
		values := make([]int, 5)
		for i, text := range []string{e.form.hp, e.form.hd, e.form.hl, e.form.hr, e.form.hn} {
			values[i], err = parseHours(text)
			if err != nil {
				e.formErr = "Une durée au moins est erronée : indiquez des nombres d'heures entiers."
				return
			}
		}
		if values[2] != 0 && !ben.IsDDV() {
			e.formErr = "Les heures de lecture DDV sont réservées aux DDV : " + ben.Person() + " ne l'est pas."
			return
		}
		if values[0] == 0 && values[1] == 0 && values[2] == 0 && values[3] == 0 && values[4] == 0 {
			e.formErr = "Vous n'avez saisi aucune valeur."
			return
		}
		entry.HP, entry.HD, entry.HL, entry.HR, entry.HN = values[0], values[1], values[2], values[3], values[4]
	}

	e.lines = append(e.lines, &PendingLine{Ben: ben, Cat: e.form.cat, Entry: entry})
	e.closeForm()
}

// commit writes every pending line in one transaction.
func (e *EntryScreen) commit() {
	if len(e.lines) == 0 || app.store == nil {
		return
	}

	entries := make([]Entry, 0, len(e.lines))
	for _, line := range e.lines {
		entries = append(entries, line.Entry)
	}

	if err := app.store.AddEntries(entries); err != nil {
		app.NotifyError("La base n'a pas pu être mise à jour", err.Error())
		return
	}

	app.Reload()
	app.Notify("La base a été mise à jour",
		fmt.Sprintf("%d ligne(s) enregistrée(s) au journal.", len(entries)))
	app.GoHome()
}

func (e *EntryScreen) remove(line *PendingLine) {
	for i, l := range e.lines {
		if l == line {
			e.lines = append(e.lines[:i], e.lines[i+1:]...)
			return
		}
	}
}

// contains reports whether a volunteer is already in the pending list; the
// permanence workflow uses it to refuse a double entry.
func (e *EntryScreen) contains(b *Benevole) bool {
	for _, l := range e.lines {
		if l.Ben == b || (l.Entry.Nom == b.Nom && l.Entry.Prenom == b.Prenom) {
			return true
		}
	}
	return false
}

func (e *EntryScreen) totalHours() int {
	total := 0
	for _, l := range e.lines {
		total += l.Entry.Hours()
	}
	return total
}

// filterBenevoles returns the volunteers matching a search, as pointers so
// the list rows keep a stable identity.
func filterBenevoles(list []Benevole, query string) []*Benevole {
	matches := make([]*Benevole, 0, len(list))
	for i := range list {
		if list[i].Matches(query) {
			matches = append(matches, &list[i])
		}
	}
	return matches
}
