package main

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	store, err := OpenStore(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("OpenStore : %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestOpenStoreCreatesSchema(t *testing.T) {
	store := testStore(t)

	ref, err := store.Load()
	if err != nil {
		t.Fatalf("Load : %v", err)
	}
	if ref.Initialised() {
		t.Fatal("une base neuve ne doit pas être vue comme initialisée")
	}
	if ref.BS != "" || ref.JournalCount != 0 {
		t.Fatalf("base neuve inattendue : %+v", ref)
	}
}

func TestReplaceBenevoles(t *testing.T) {
	store := testStore(t)

	list := []Benevole{
		{Nom: "IZZO", Prenom: "Jeanne", Donneur: "DDV"},
		{Nom: "ABRAHAM", Prenom: "Brigitte"},
	}
	if err := store.ReplaceBenevoles("MANOSQUE - 04M", list); err != nil {
		t.Fatalf("ReplaceBenevoles : %v", err)
	}

	ref, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !ref.Initialised() || ref.BS != "MANOSQUE - 04M" {
		t.Fatalf("référence inattendue : %+v", ref)
	}
	// Load trie par nom puis prénom.
	if ref.Benevoles[0].Nom != "ABRAHAM" || ref.Benevoles[1].Nom != "IZZO" {
		t.Fatalf("ordre inattendu : %+v", ref.Benevoles)
	}
	if !ref.Benevoles[1].IsDDV() {
		t.Fatal("le marqueur DDV a été perdu")
	}

	// Une seconde initialisation remplace la liste et le nom de la BS.
	if err := store.ReplaceBenevoles("NARBONNE - 11N", list[:1]); err != nil {
		t.Fatalf("seconde ReplaceBenevoles : %v", err)
	}
	ref, err = store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(ref.Benevoles) != 1 || ref.BS != "NARBONNE - 11N" {
		t.Fatalf("la seconde initialisation n'a pas remplacé : %+v", ref)
	}
}

func TestAddEntriesAndRead(t *testing.T) {
	store := testStore(t)

	entries := []Entry{
		{Date: day(3), Nom: "ABRAHAM", Prenom: "Brigitte", HP: 3},
		{Date: day(10), Nom: "IZZO", Prenom: "Jeanne", HL: 12, Commentaire: "trois livres"},
		{Date: day(20), Nom: "THOMAS", Prenom: "Jacques", HP: -3, Commentaire: "erreur"},
	}
	if err := store.AddEntries(entries); err != nil {
		t.Fatalf("AddEntries : %v", err)
	}

	got, err := store.EntriesBetween(day(1), day(30))
	if err != nil {
		t.Fatalf("EntriesBetween : %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("%d écritures, attendu 3", len(got))
	}
	if !got[0].Date.Equal(day(3)) || got[0].HP != 3 {
		t.Fatalf("première écriture inattendue : %+v", got[0])
	}
	if got[1].Commentaire != "trois livres" || got[1].HL != 12 {
		t.Fatalf("commentaire ou heures perdus : %+v", got[1])
	}
	if !got[2].IsCorrection() {
		t.Fatalf("la rectification n'a pas été relue : %+v", got[2])
	}

	// Les bornes sont incluses, ce qui est en dehors ne remonte pas.
	window, err := store.EntriesBetween(day(3), day(10))
	if err != nil {
		t.Fatal(err)
	}
	if len(window) != 2 {
		t.Fatalf("%d écritures dans la période, attendu 2", len(window))
	}

	ref, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if ref.JournalCount != 3 || !ref.FirstDate.Equal(day(3)) || !ref.LastDate.Equal(day(20)) {
		t.Fatalf("compteurs du journal inattendus : %+v", ref)
	}
}

func TestAddEntriesEmptyIsNoop(t *testing.T) {
	store := testStore(t)
	if err := store.AddEntries(nil); err != nil {
		t.Fatalf("AddEntries(nil) : %v", err)
	}
}

func TestAddEntriesTruncatesComment(t *testing.T) {
	store := testStore(t)

	long := "commentaire beaucoup trop long pour la colonne"
	err := store.AddEntries([]Entry{{Date: day(1), Nom: "A", Prenom: "B", HP: 1, Commentaire: long}})
	if err != nil {
		t.Fatal(err)
	}

	got, err := store.EntriesBetween(day(1), day(1))
	if err != nil {
		t.Fatal(err)
	}
	if runes := []rune(got[0].Commentaire); len(runes) != commentMaxLen {
		t.Fatalf("commentaire de %d caractères : %q", len(runes), got[0].Commentaire)
	}
}

// TestMigrateAddsCommentColumn checks the upgrade path of a journal written
// by version 4, which had no comment column.
func TestMigrateAddsCommentColumn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v4.db")

	legacy, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = legacy.Exec(`create table JOURNAL ([Date] DATE NOT NULL,
		[nom] NVARCHAR(20) NOT NULL, [prenom] NVARCHAR(20) NOT NULL,
		[HP] INTEGER, [HL] INTEGER, [HD] INTEGER, [HR] INTEGER, [HN] INTEGER)`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := legacy.Exec(`insert into JOURNAL values ('2023-01-03', 'IZZO', 'Jeanne', 3, null, null, null, null)`); err != nil {
		t.Fatal(err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}

	store, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore sur une base version 4 : %v", err)
	}
	defer store.Close()

	got, err := store.EntriesBetween(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2023, 1, 31, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("lecture après migration : %v", err)
	}
	if len(got) != 1 || got[0].HP != 3 || got[0].Commentaire != "" {
		t.Fatalf("écriture migrée inattendue : %+v", got)
	}
}
