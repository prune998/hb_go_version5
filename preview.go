package main

import (
	"fmt"
	"strings"
	"time"

	. "go.hasen.dev/shirei"
)

// This file is development tooling: `hb_go_version5 -png out.png -screen X`
// renders one screen headlessly, which is how the layout is checked without
// opening a window. It never touches the database.

// A screen name prefixed with "db-" opens the real database instead of the
// sample data: `-screen db-home`, `-screen db-journal`.
const previewDBPrefix = "db-"

var previewScreens = []string{
	"home", "empty",
	"perm-params", "perm-saisie",
	"indiv-saisie", "indiv-form", "cons-form",
	"inject-fichier", "inject-controle",
	"journal-params", "journal-detail", "journal-recap", "journal-indiv",
	"about", "confirm",
}

func renderScreenshot(path, screen string) error {
	preparePreview(screen)
	return RenderToPNG(path, windowWidth, windowHeight, RootView)
}

// preparePreview fills the application state with sample data and opens the
// requested screen.
func preparePreview(screen string) {
	if real, ok := strings.CutPrefix(screen, previewDBPrefix); ok {
		app.Setup()
		previewScreen(real)
		return
	}

	paths, _ := NewPaths()
	app.paths = paths
	app.data = previewReference()
	previewScreen(screen)
}

// previewScreen opens one screen on the data already loaded in the state.
func previewScreen(screen string) {
	if screen == "empty" {
		app.data = Reference{}
		app.screen = ScreenHome
		return
	}

	switch screen {
	case "perm-params":
		app.openEntry(KindPermanence)
	case "perm-saisie":
		app.openEntry(KindPermanence)
		app.entry.step = 1
		for _, b := range previewPicks(4) {
			app.entry.addPermanence(b)
		}
	case "indiv-saisie":
		app.openEntry(KindIndividuelle)
		app.entry.step = 1
		previewIndivLines()
	case "indiv-form":
		app.openEntry(KindIndividuelle)
		app.entry.step = 1
		previewIndivLines()
		app.entry.pick(previewPicks(1)[0])
	case "cons-form":
		app.openEntry(KindConsolidation)
		app.entry.step = 1
		app.entry.pick(previewPicks(1)[0])
	case "inject-fichier":
		app.openInject()
		app.inject.path = app.paths.Export(fileJournalDetail)
	case "inject-controle":
		app.openInject()
		app.inject.path = app.paths.Export(fileJournalDetail)
		app.inject.rows = previewEntries()
		app.inject.recap = BuildRecap(derefEntries(app.inject.rows))
		app.inject.step = 1
	case "journal":
		app.openJournal()
		app.journal.from = formatFR(app.data.FirstDate)
		app.journal.to = formatFR(app.data.LastDate)
		app.journal.generate()
	case "journal-params":
		app.openJournal()
	case "journal-detail":
		previewJournal(ReportDetail)
	case "journal-recap":
		previewJournal(ReportRecap)
	case "journal-indiv":
		previewJournal(ReportIndiv)
	case "about":
		app.screen = ScreenHome
		app.about = true
	case "confirm":
		app.screen = ScreenHome
		app.confirm.Ask("Initialisation du fichier des bénévoles",
			"42 bénévoles ont été lus dans « benevoles.xlsx », pour la BS « MANOSQUE - 04M ».\n\n"+
				"La liste actuelle (42 bénévoles) sera remplacée. Le journal des heures n'est pas modifié.",
			"Remplacer la liste", true, func() {})
	default:
		app.screen = ScreenHome
	}
}

func previewReference() Reference {
	names := [][3]string{
		{"ABRAHAM", "Brigitte", "DDV"},
		{"AURAN", "Annie", "DDV"},
		{"BARATIER", "Jean-Louis", ""},
		{"BARBIER", "Corinne", "DDV"},
		{"BARTOLI", "Michèle", "DDV"},
		{"CHARPENTIER", "Paul", ""},
		{"DESMETTRE", "Lucette", ""},
		{"IZZO", "Jeanne", "DDV"},
		{"MEZIÈRES", "Marie-Claire", ""},
		{"THOMAS", "Jacques", "DDV"},
	}

	snap := Reference{
		BS:           "MANOSQUE - 04M",
		JournalCount: 1540,
		FirstDate:    time.Date(2022, 12, 8, 0, 0, 0, 0, time.UTC),
		LastDate:     today(),
	}
	for i, n := range names {
		snap.Benevoles = append(snap.Benevoles, Benevole{ID: i + 1, Nom: n[0], Prenom: n[1], Donneur: n[2]})
	}
	return snap
}

func previewPicks(n int) []*Benevole {
	picks := make([]*Benevole, 0, n)
	for i := 0; i < n && i < len(app.data.Benevoles); i++ {
		picks = append(picks, &app.data.Benevoles[i])
	}
	return picks
}

func previewIndivLines() {
	e := &app.entry
	for i, b := range previewPicks(3) {
		entry := Entry{Date: today(), Nom: b.Nom, Prenom: b.Prenom}
		cat := allCategories[i%len(allCategories)]
		entry.SetCategory(cat, 3+i)
		entry.Commentaire = fmt.Sprintf("exemple %d", i+1)
		e.lines = append(e.lines, &PendingLine{Ben: b, Cat: cat, Entry: entry})
	}
}

func previewEntries() []*Entry {
	base := today().AddDate(0, 0, -20)
	samples := []Entry{
		{Date: base, Nom: "ABRAHAM", Prenom: "Brigitte", HP: 3},
		{Date: base, Nom: "AURAN", Prenom: "Annie", HP: 3, Commentaire: "remplacement"},
		{Date: base.AddDate(0, 0, 3), Nom: "BARTOLI", Prenom: "Michèle", HL: 12},
		{Date: base.AddDate(0, 0, 5), Nom: "THOMAS", Prenom: "Jacques", HD: 4, HR: 2},
		{Date: base.AddDate(0, 0, 7), Nom: "IZZO", Prenom: "Jeanne", HP: -3, Commentaire: "présence saisie à tort"},
		{Date: base.AddDate(0, 0, 9), Nom: "DESMETTRE", Prenom: "Lucette", HP: 3, HN: 6},
	}
	rows := make([]*Entry, len(samples))
	for i := range samples {
		rows[i] = &samples[i]
	}
	return rows
}

func derefEntries(rows []*Entry) []Entry {
	out := make([]Entry, 0, len(rows))
	for _, r := range rows {
		out = append(out, *r)
	}
	return out
}

func previewJournal(kind ReportKind) {
	app.openJournal()
	j := &app.journal
	j.kind = kind
	j.step = 1
	j.rows = previewEntries()
	entries := derefEntries(j.rows)
	j.recap = BuildRecap(entries)
	j.indiv = BuildIndiv(entries)
	switch kind {
	case ReportRecap:
		j.excelPath = app.paths.Export(fileJournalRecap)
	case ReportIndiv:
		j.excelPath = app.paths.Export(fileJournalIndiv)
	default:
		j.excelPath = app.paths.Export(fileJournalDetail)
	}
}
