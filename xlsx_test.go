package main

import (
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

// TestDetailRoundTrip is the guarantee the injection feature relies on: a
// journal exported here can be injected back on another machine.
func TestDetailRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), fileJournalDetail)
	entries := sampleEntries()

	if err := exportDetail(path, "MANOSQUE - 04M", entries); err != nil {
		t.Fatalf("exportDetail : %v", err)
	}

	back, err := importJournal(path)
	if err != nil {
		t.Fatalf("importJournal : %v", err)
	}
	if len(back) != len(entries) {
		t.Fatalf("%d écritures relues, attendu %d", len(back), len(entries))
	}
	for i := range entries {
		want, got := entries[i], back[i]
		if !got.Date.Equal(want.Date) || got.Nom != want.Nom || got.Prenom != want.Prenom {
			t.Fatalf("ligne %d : %+v, attendu %+v", i, got, want)
		}
		if got.HP != want.HP || got.HD != want.HD || got.HL != want.HL || got.HR != want.HR || got.HN != want.HN {
			t.Fatalf("ligne %d : heures %+v, attendu %+v", i, got, want)
		}
		if got.Commentaire != want.Commentaire {
			t.Fatalf("ligne %d : commentaire %q, attendu %q", i, got.Commentaire, want.Commentaire)
		}
	}
}

func TestDetailHeaderCell(t *testing.T) {
	path := filepath.Join(t.TempDir(), fileJournalDetail)
	if err := exportDetail(path, "BS", sampleEntries()); err != nil {
		t.Fatal(err)
	}

	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	got, err := f.GetCellValue(f.GetSheetName(0), "A1")
	if err != nil {
		t.Fatal(err)
	}
	if got != detailHeader {
		t.Fatalf("A1 = %q, attendu %q", got, detailHeader)
	}
}

func TestImportJournalRejectsForeignFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "autre.xlsx")

	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	f.SetCellValue(sheet, "A1", "Liste des adhérents")
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	f.Close()

	if _, err := importJournal(path); err == nil {
		t.Fatal("un fichier non conforme doit être refusé")
	}
}

func TestImportJournalRejectsBadDate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.xlsx")

	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	for col, value := range []string{detailHeader, "Date", "Nom", "Prénom ", "H.Perm.", "H.ext.", "H.lect.", "H.Rég.", "H.Nat.", "Commentaire"} {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		f.SetCellValue(sheet, cell, value)
	}
	f.SetCellValue(sheet, "A2", "BS")
	f.SetCellValue(sheet, "B2", "32/13/2024")
	f.SetCellValue(sheet, "C2", "IZZO")
	f.SetCellValue(sheet, "D2", "Jeanne")
	f.SetCellValue(sheet, "E2", 3)
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	f.Close()

	if _, err := importJournal(path); err == nil {
		t.Fatal("une date invraisemblable doit être refusée")
	}
}

func TestImportOrphee(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orphee.xlsx")

	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	// Ligne d'en-tête du rapport : la colonne B vaut "A", elle est ignorée.
	f.SetCellValue(sheet, "B1", "A")
	f.SetCellValue(sheet, "E1", "MANOSQUE - 04M")
	rows := [][]any{
		{"", "B", "ABRAHAM", "Brigitte", "", "DDV"},
		{"", "B", "BARATIER", "Jean-Louis", "", ""},
		{"", "A", "IGNORÉ", "Adhérent", "", ""},
		{"", "B", "", "", "", ""}, // ligne vide en fin de rapport
	}
	for i, row := range rows {
		for col, value := range row {
			cell, _ := excelize.CoordinatesToCellName(col+1, i+2)
			f.SetCellValue(sheet, cell, value)
		}
	}
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	f.Close()

	report, err := importOrphee(path)
	if err != nil {
		t.Fatalf("importOrphee : %v", err)
	}
	if report.BS != "MANOSQUE - 04M" {
		t.Fatalf("BS = %q", report.BS)
	}
	if len(report.Benevoles) != 2 {
		t.Fatalf("%d bénévoles, attendu 2 : %+v", len(report.Benevoles), report.Benevoles)
	}
	if report.Benevoles[0].Nom != "ABRAHAM" || !report.Benevoles[0].IsDDV() {
		t.Fatalf("premier bénévole inattendu : %+v", report.Benevoles[0])
	}
	if report.Benevoles[1].ID != 2 {
		t.Fatalf("identifiants non séquentiels : %+v", report.Benevoles)
	}
}

func TestImportOrpheeRejectsEmptyReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vide.xlsx")

	f := excelize.NewFile()
	f.SetCellValue(f.GetSheetName(0), "A1", "rien")
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	f.Close()

	if _, err := importOrphee(path); err == nil {
		t.Fatal("un rapport sans bénévole doit être refusé")
	}
}

func TestExportRecapAndIndiv(t *testing.T) {
	dir := t.TempDir()
	entries := sampleEntries()

	recapPath := filepath.Join(dir, fileJournalRecap)
	if err := exportRecap(recapPath, "BS", "01/06/2024", "30/06/2024", BuildRecap(entries)); err != nil {
		t.Fatalf("exportRecap : %v", err)
	}
	indivPath := filepath.Join(dir, fileJournalIndiv)
	if err := exportIndiv(indivPath, "BS", "01/06/2024", "30/06/2024", BuildIndiv(entries)); err != nil {
		t.Fatalf("exportIndiv : %v", err)
	}

	f, err := excelize.OpenFile(recapPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if got, _ := f.GetCellValue(f.GetSheetName(0), "I2"); got != "30" {
		t.Fatalf("total du récapitulatif = %q, attendu 30", got)
	}

	g, err := excelize.OpenFile(indivPath)
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	if got, _ := g.GetCellValue(g.GetSheetName(0), "D2"); got != "ABRAHAM Brigitte" {
		t.Fatalf("premier bénévole du comptage individuel = %q", got)
	}
	if got, _ := g.GetCellValue(g.GetSheetName(0), "K2"); got != "20" {
		t.Fatalf("total individuel = %q, attendu 20", got)
	}
}

func TestHoursCell(t *testing.T) {
	row := []string{"3", "", " 4 ", "5.0", "-2", "x"}
	cases := []struct {
		index int
		want  int
		bad   bool
	}{{0, 3, false}, {1, 0, false}, {2, 4, false}, {3, 5, false}, {4, -2, false}, {5, 0, true}}

	for _, tc := range cases {
		got, err := hoursCell(row, tc.index)
		if tc.bad {
			if err == nil {
				t.Fatalf("colonne %d devait être refusée", tc.index)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Fatalf("colonne %d = %d, %v", tc.index, got, err)
		}
	}
}
