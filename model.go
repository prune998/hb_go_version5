package main

import (
	"strings"
	"time"
)

// commentMaxLen is the length of the JOURNAL.commentaire column; longer
// comments are truncated, as documented in the manual.
const commentMaxLen = 30

// Benevole is one volunteer, as imported from the Orphée report.
type Benevole struct {
	ID      int
	Nom     string
	Prenom  string
	Donneur string // "DDV" for a "donneur de voix", empty otherwise
}

// IsDDV reports whether the volunteer is a "donneur de voix"; only they may
// be credited with reading hours.
func (b Benevole) IsDDV() bool {
	return strings.EqualFold(strings.TrimSpace(b.Donneur), "DDV")
}

// Person is the display name used everywhere in the interface.
func (b Benevole) Person() string {
	return strings.TrimSpace(b.Nom + " " + b.Prenom)
}

// Matches reports whether the volunteer matches a free text search.
func (b Benevole) Matches(query string) bool {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return true
	}
	hay := strings.ToLower(b.Nom + " " + b.Prenom + " " + b.Donneur)
	for _, word := range strings.Fields(query) {
		if !strings.Contains(hay, word) {
			return false
		}
	}
	return true
}

// Category is one of the four "hors permanence" hour categories, in the same
// order as the combo box of the original program.
type Category int

const (
	CatExterieur Category = iota
	CatLectureDDV
	CatRegional
	CatNational
)

var allCategories = []Category{CatExterieur, CatLectureDDV, CatRegional, CatNational}

// Label is the full category name.
func (c Category) Label() string {
	switch c {
	case CatExterieur:
		return "Extérieur"
	case CatLectureDDV:
		return "Lecture DDV"
	case CatRegional:
		return "Régional"
	case CatNational:
		return "National"
	}
	return ""
}

// Short is the column heading used in the tables and the workbooks.
func (c Category) Short() string {
	switch c {
	case CatExterieur:
		return "Ext."
	case CatLectureDDV:
		return "Lect."
	case CatRegional:
		return "Rég."
	case CatNational:
		return "Nat."
	}
	return ""
}

// Help is the one line explanation taken from the manual.
func (c Category) Help() string {
	switch c {
	case CatExterieur:
		return "Hors permanence : travail à domicile, vérification, forums, expositions, points-relais…"
	case CatLectureDDV:
		return "Lecture de livres pour les DDV (durée du livre × 4, arrondie à l'heure)."
	case CatRegional:
		return "Heures passées au niveau de la région (DR, DRA, formateurs, chargés de mission)."
	case CatNational:
		return "Heures passées au niveau national ADV."
	}
	return ""
}

// Entry is one line of the JOURNAL table. Hours are whole hours and may be
// negative: the manual documents a negative entry as the way to cancel a
// mistake (it also decreases the attendance count of the volunteer).
type Entry struct {
	Date        time.Time
	Nom         string
	Prenom      string
	HP          int // heures de permanence
	HD          int // heures extérieur
	HL          int // heures de lecture DDV
	HR          int // heures régionales
	HN          int // heures nationales
	Commentaire string
}

// Person is the display name of the volunteer the line belongs to.
func (e Entry) Person() string { return strings.TrimSpace(e.Nom + " " + e.Prenom) }

// Hours is the total number of hours the line carries.
func (e Entry) Hours() int { return e.HP + e.HD + e.HL + e.HR + e.HN }

// IsCorrection reports whether the line cancels a previous one; those are
// shown in red, on screen and in the journal.
func (e Entry) IsCorrection() bool {
	return e.HP < 0 || e.HD < 0 || e.HL < 0 || e.HR < 0 || e.HN < 0
}

// Category returns the hours stored for one category.
func (e Entry) Category(c Category) int {
	switch c {
	case CatExterieur:
		return e.HD
	case CatLectureDDV:
		return e.HL
	case CatRegional:
		return e.HR
	case CatNational:
		return e.HN
	}
	return 0
}

// SetCategory stores hours in one category, leaving the others untouched.
func (e *Entry) SetCategory(c Category, hours int) {
	switch c {
	case CatExterieur:
		e.HD = hours
	case CatLectureDDV:
		e.HL = hours
	case CatRegional:
		e.HR = hours
	case CatNational:
		e.HN = hours
	}
}

// truncateComment keeps the comment within the column width, counting
// characters and not bytes so accented text is not cut in half.
func truncateComment(s string) string {
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) <= commentMaxLen {
		return s
	}
	return string(runes[:commentMaxLen])
}
