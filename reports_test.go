package main

import (
	"testing"
	"time"
)

func day(d int) time.Time { return time.Date(2024, 6, d, 0, 0, 0, 0, time.UTC) }

func sampleEntries() []Entry {
	return []Entry{
		{Date: day(3), Nom: "ABRAHAM", Prenom: "Brigitte", HP: 3},
		{Date: day(3), Nom: "IZZO", Prenom: "Jeanne", HP: 3},
		{Date: day(10), Nom: "ABRAHAM", Prenom: "Brigitte", HP: 3, HD: 2},
		{Date: day(12), Nom: "ABRAHAM", Prenom: "Brigitte", HL: 12},
		// Rectification d'une présence saisie à tort.
		{Date: day(3), Nom: "IZZO", Prenom: "Jeanne", HP: -3, Commentaire: "erreur"},
		{Date: day(15), Nom: "THOMAS", Prenom: "Jacques", HR: 4, HN: 6},
	}
}

func TestBuildRecap(t *testing.T) {
	recap := BuildRecap(sampleEntries())

	if recap.Count != 6 {
		t.Fatalf("Count = %d", recap.Count)
	}
	if recap.HP != 6 || recap.HD != 2 || recap.HL != 12 || recap.HR != 4 || recap.HN != 6 {
		t.Fatalf("totaux inattendus : %+v", recap)
	}
	if got := recap.Total(); got != 30 {
		t.Fatalf("Total = %d, attendu 30", got)
	}
}

func TestBuildRecapEmpty(t *testing.T) {
	recap := BuildRecap(nil)
	if recap.Count != 0 || recap.Total() != 0 {
		t.Fatalf("récapitulatif vide inattendu : %+v", recap)
	}
}

func TestBuildIndiv(t *testing.T) {
	rows := BuildIndiv(sampleEntries())

	if len(rows) != 3 {
		t.Fatalf("%d bénévoles, attendu 3", len(rows))
	}
	// Tri par nom.
	if rows[0].Person != "ABRAHAM Brigitte" || rows[1].Person != "IZZO Jeanne" || rows[2].Person != "THOMAS Jacques" {
		t.Fatalf("ordre inattendu : %q, %q, %q", rows[0].Person, rows[1].Person, rows[2].Person)
	}

	abraham := rows[0]
	if abraham.NbPerm != 2 {
		t.Fatalf("ABRAHAM NbPerm = %d, attendu 2", abraham.NbPerm)
	}
	if abraham.HP != 6 || abraham.HD != 2 || abraham.HL != 12 {
		t.Fatalf("ABRAHAM heures inattendues : %+v", abraham)
	}
	if got := abraham.Hours(); got != 20 {
		t.Fatalf("ABRAHAM total = %d, attendu 20", got)
	}

	// Une ligne négative annule la présence comptée par la ligne positive.
	izzo := rows[1]
	if izzo.NbPerm != 0 || izzo.HP != 0 {
		t.Fatalf("IZZO devait être remis à zéro : %+v", izzo)
	}

	// Les heures sans permanence ne comptent aucune présence.
	if rows[2].NbPerm != 0 {
		t.Fatalf("THOMAS NbPerm = %d, attendu 0", rows[2].NbPerm)
	}
}

func TestEntryHelpers(t *testing.T) {
	e := Entry{Nom: "THOMAS", Prenom: "Jacques"}
	e.SetCategory(CatRegional, 4)
	if e.HR != 4 || e.Category(CatRegional) != 4 {
		t.Fatalf("SetCategory(Régional) = %+v", e)
	}
	e.SetCategory(CatLectureDDV, -2)
	if !e.IsCorrection() {
		t.Fatal("une valeur négative doit être vue comme une rectification")
	}
	if got := e.Hours(); got != 2 {
		t.Fatalf("Hours = %d, attendu 2", got)
	}
	if got := e.Person(); got != "THOMAS Jacques" {
		t.Fatalf("Person = %q", got)
	}
}

func TestTruncateComment(t *testing.T) {
	// 30 caractères accentués : la troncature compte des caractères, pas des octets.
	long := "ééééééééééééééééééééééééééééééFIN"
	got := truncateComment(long)
	if runes := []rune(got); len(runes) != commentMaxLen {
		t.Fatalf("%d caractères, attendu %d", len(runes), commentMaxLen)
	}
	if got := truncateComment("  court  "); got != "court" {
		t.Fatalf("truncateComment = %q", got)
	}
}

func TestBenevoleMatches(t *testing.T) {
	b := Benevole{Nom: "MEZIÈRES", Prenom: "Marie-Claire", Donneur: "DDV"}
	for _, query := range []string{"", "mez", "marie", "MEZ claire", "ddv"} {
		if !b.Matches(query) {
			t.Fatalf("%q devait correspondre", query)
		}
	}
	if b.Matches("izzo") {
		t.Fatal("\"izzo\" ne devait pas correspondre")
	}
	if !b.IsDDV() {
		t.Fatal("IsDDV = false")
	}
}
