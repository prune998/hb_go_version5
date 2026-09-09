package main

import "sort"

// ReportKind is one of the three outputs of "Edition du journal".
type ReportKind int

const (
	ReportDetail ReportKind = iota
	ReportRecap
	ReportIndiv
)

func (k ReportKind) Label() string {
	switch k {
	case ReportDetail:
		return "Détaillé"
	case ReportRecap:
		return "Récapitulatif"
	case ReportIndiv:
		return "Individuel"
	}
	return ""
}

func (k ReportKind) Help() string {
	switch k {
	case ReportDetail:
		return "Toutes les écritures de la période, par ordre chronologique."
	case ReportRecap:
		return "Les totaux d'heures de la période, toutes catégories."
	case ReportIndiv:
		return "Par bénévole : nombre de présences et répartition des heures."
	}
	return ""
}

// Recap totals the hours of a period.
type Recap struct {
	Count int
	HP    int
	HD    int
	HL    int
	HR    int
	HN    int
}

// Total is the grand total of the period.
func (r Recap) Total() int { return r.HP + r.HD + r.HL + r.HR + r.HN }

// BuildRecap sums a list of journal lines.
func BuildRecap(entries []Entry) Recap {
	recap := Recap{Count: len(entries)}
	for _, e := range entries {
		recap.HP += e.HP
		recap.HD += e.HD
		recap.HL += e.HL
		recap.HR += e.HR
		recap.HN += e.HN
	}
	return recap
}

// IndivRow is the activity of one volunteer over a period.
type IndivRow struct {
	Person string
	NbPerm int // number of attended "permanences"
	HP     int
	HD     int
	HL     int
	HR     int
	HN     int
}

// Hours is the total number of hours of the volunteer, all categories.
func (r IndivRow) Hours() int { return r.HP + r.HD + r.HL + r.HR + r.HN }

// BuildIndiv groups the journal by volunteer, sorted by name.
//
// The attendance count follows the rule described in the manual: a line with
// permanence hours counts as one presence, and a negative line cancels one
// (that is how a wrongly recorded attendance is corrected).
func BuildIndiv(entries []Entry) []*IndivRow {
	byPerson := make(map[string]*IndivRow, len(entries))
	for _, e := range entries {
		person := e.Person()
		row := byPerson[person]
		if row == nil {
			row = &IndivRow{Person: person}
			byPerson[person] = row
		}
		switch {
		case e.HP > 0:
			row.NbPerm++
		case e.HP < 0:
			row.NbPerm--
		}
		row.HP += e.HP
		row.HD += e.HD
		row.HL += e.HL
		row.HR += e.HR
		row.HN += e.HN
	}

	rows := make([]*IndivRow, 0, len(byPerson))
	for _, row := range byPerson {
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Person < rows[j].Person })
	return rows
}
