package main

import (
	"testing"
	"time"
)

func freezeToday(t *testing.T, day time.Time) {
	t.Helper()
	previous := today
	today = func() time.Time { return day }
	t.Cleanup(func() { today = previous })
}

func TestParseDate(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr string
	}{
		{"valide", "23/12/2024", ""},
		{"vide", "", "la date est obligatoire"},
		{"trop courte", "1/2/2024", "date erronée : elle doit comporter 10 caractères (jj/mm/aaaa)"},
		{"jour non numérique", "aa/12/2024", "jour erroné"},
		{"mois non numérique", "23/xx/2024", "mois erroné"},
		{"année non numérique", "23/12/aaaa", "année erronée"},
		{"mois hors bornes", "23/13/2024", "mois erroné"},
		{"jour hors bornes", "32/12/2024", "date invraisemblable"},
		{"29 février bissextile", "29/02/2024", ""},
		{"29 février non bissextile", "29/02/2023", "date invraisemblable"},
		{"29 février 2000", "29/02/2000", ""},
		{"29 février 1900", "29/02/1900", "date invraisemblable"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseDate(tc.input)
			switch {
			case tc.wantErr == "" && err != nil:
				t.Fatalf("parseDate(%q) = %v, attendu aucune erreur", tc.input, err)
			case tc.wantErr != "" && err == nil:
				t.Fatalf("parseDate(%q) n'a pas été rejetée", tc.input)
			case tc.wantErr != "" && err.Error() != tc.wantErr:
				t.Fatalf("parseDate(%q) = %q, attendu %q", tc.input, err, tc.wantErr)
			}
		})
	}
}

func TestParseDateRoundTrip(t *testing.T) {
	d, err := parseDate("05/03/2024")
	if err != nil {
		t.Fatal(err)
	}
	if got := formatISO(d); got != "2024-03-05" {
		t.Fatalf("formatISO = %q", got)
	}
	if got := formatFR(d); got != "05/03/2024" {
		t.Fatalf("formatFR = %q", got)
	}
}

func TestParsePastDateRejectsFuture(t *testing.T) {
	freezeToday(t, time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC))

	if _, err := parsePastDate("15/06/2024"); err != nil {
		t.Fatalf("la date du jour doit être acceptée : %v", err)
	}
	_, err := parsePastDate("16/06/2024")
	if err == nil || err.Error() != "la date saisie est postérieure à la date du jour" {
		t.Fatalf("date future : erreur = %v", err)
	}
}

func TestParseDateRange(t *testing.T) {
	freezeToday(t, time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC))

	if _, _, err := parseDateRange("01/06/2024", "15/06/2024"); err != nil {
		t.Fatalf("période valide refusée : %v", err)
	}
	if _, _, err := parseDateRange("20/06/2024", "25/06/2024"); err == nil {
		t.Fatal("une date de départ future doit être refusée")
	}
	if _, _, err := parseDateRange("10/06/2024", "01/06/2024"); err == nil {
		t.Fatal("un départ postérieur à la fin doit être refusé")
	}
	if _, _, err := parseDateRange("10/06/2024", "31/02/2024"); err == nil {
		t.Fatal("une date de fin invraisemblable doit être refusée")
	}
}

func TestParseHours(t *testing.T) {
	if h, err := parseHours(" 3 "); err != nil || h != 3 {
		t.Fatalf("parseHours(\" 3 \") = %d, %v", h, err)
	}
	// Une durée négative est la façon documentée de corriger une erreur.
	if h, err := parseHours("-3"); err != nil || h != -3 {
		t.Fatalf("parseHours(\"-3\") = %d, %v", h, err)
	}
	if h, err := parseHours("0"); err != nil || h != 0 {
		t.Fatalf("parseHours(\"0\") = %d, %v", h, err)
	}
	for _, bad := range []string{"", "trois", "3,5", "3h"} {
		if _, err := parseHours(bad); err == nil {
			t.Fatalf("parseHours(%q) devait être refusée", bad)
		}
	}
}
