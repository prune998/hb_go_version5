package main

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// detailHeader is the first cell of the detailed journal. The injection
// feature checks it, so the two must stay in sync: it is how the program
// recognises one of its own exports.
const detailHeader = "Journal détaillé"

// ---------------------------------------------------------------- exports

// exportDetail writes the detailed journal, the workbook that can be
// injected in another installation of the program.
func exportDetail(path, bs string, entries []Entry) error {
	book := newBook([]string{
		detailHeader, "Date", "Nom", "Prénom ", "H.Perm.", "H.ext.", "H.lect.", "H.Rég.", "H.Nat.", "Commentaire",
	})
	defer book.file.Close()

	for i, e := range entries {
		row := i + 2
		book.set(1, row, bs)
		book.set(2, row, formatFR(e.Date))
		book.set(3, row, e.Nom)
		book.set(4, row, e.Prenom)
		book.set(5, row, e.HP)
		book.set(6, row, e.HD)
		book.set(7, row, e.HL)
		book.set(8, row, e.HR)
		book.set(9, row, e.HN)
		book.set(10, row, e.Commentaire)
	}

	book.widths(map[string]float64{
		"A": math.Max(12, float64(len([]rune(bs))+2)),
		"B": 11, "C": 20, "D": 20,
		"E": 6, "F": 6, "G": 6, "H": 6, "I": 6,
		"J": 30,
	})
	return book.save(path)
}

// exportRecap writes the totals of the period.
func exportRecap(path, bs, fromFR, toFR string, recap Recap) error {
	book := newBook([]string{
		"BS de ", "du ", "au ", "H.Perm.", "H.ext.", "H.lect.", "H.Rég.", "H.Nat.", "Total",
	})
	defer book.file.Close()

	values := []any{bs, fromFR, toFR, recap.HP, recap.HD, recap.HL, recap.HR, recap.HN, recap.Total()}
	for i, v := range values {
		book.set(i+1, 2, v)
	}

	book.widths(map[string]float64{
		"A": 20, "B": 11, "C": 11,
		"D": 8, "E": 8, "F": 8, "G": 8, "H": 8, "I": 8,
	})
	return book.save(path)
}

// exportIndiv writes the per volunteer summary of the period.
func exportIndiv(path, bs, fromFR, toFR string, rows []*IndivRow) error {
	book := newBook([]string{
		"BS de ", "du ", "au ", "Nom Prénom", "NB perm.", "H.Perm.", "H.ext.", "H.lect.", "H.Rég.", "H.Nat.", "Total",
	})
	defer book.file.Close()

	book.set(1, 2, bs)
	book.set(2, 2, fromFR)
	book.set(3, 2, toFR)

	for i, r := range rows {
		row := i + 2
		book.set(4, row, r.Person)
		book.set(5, row, r.NbPerm)
		book.set(6, row, r.HP)
		book.set(7, row, r.HD)
		book.set(8, row, r.HL)
		book.set(9, row, r.HR)
		book.set(10, row, r.HN)
		book.set(11, row, r.Hours())
	}

	book.widths(map[string]float64{
		"A": 15, "B": 11, "C": 11, "D": 30,
		"E": 8, "F": 8, "G": 8, "H": 8, "I": 8, "J": 8, "K": 8,
	})
	return book.save(path)
}

// book is a thin helper around excelize: one sheet, a bold header row
// repeated when printing, and cells addressed by column and row numbers.
type book struct {
	file  *excelize.File
	sheet string
	err   error
}

func newBook(header []string) *book {
	f := excelize.NewFile()
	b := &book{file: f, sheet: f.GetSheetName(0)}

	for i, title := range header {
		b.set(i+1, 1, title)
	}
	if style, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}}); err == nil {
		last, _ := excelize.ColumnNumberToName(len(header))
		b.record(f.SetCellStyle(b.sheet, "A1", last+"1", style))
	}
	b.record(f.SetDefinedName(&excelize.DefinedName{
		Name:     "Print_Titles",
		RefersTo: fmt.Sprintf("%s!$1:$1", b.sheet),
		Scope:    b.sheet,
	}))
	return b
}

func (b *book) set(col, row int, value any) {
	name, err := excelize.CoordinatesToCellName(col, row)
	if err != nil {
		b.record(err)
		return
	}
	b.record(b.file.SetCellValue(b.sheet, name, value))
}

func (b *book) widths(widths map[string]float64) {
	for col, w := range widths {
		b.record(b.file.SetColWidth(b.sheet, col, col, w))
	}
}

func (b *book) record(err error) {
	if err != nil && b.err == nil {
		b.err = err
	}
}

func (b *book) save(path string) error {
	if b.err != nil {
		return fmt.Errorf("écriture du fichier Excel : %w", b.err)
	}
	if err := b.file.SaveAs(path); err != nil {
		return fmt.Errorf("impossible d'écrire « %s » : %w\nLe fichier est peut-être ouvert dans Excel", path, err)
	}
	return nil
}

// ---------------------------------------------------------------- imports

// OrpheeReport is the content of the "Bénévoles_liste pour heures" report
// exported from Orphée.
type OrpheeReport struct {
	BS        string
	Benevoles []Benevole
}

// importOrphee reads the volunteer list. Column B holds the membership type
// ("A" marks a line to ignore), C the surname, D the first name and F the
// "DDV" flag; the name of the BS is in E1.
func importOrphee(path string) (OrpheeReport, error) {
	rows, err := sheetRows(path)
	if err != nil {
		return OrpheeReport{}, err
	}
	if len(rows) == 0 {
		return OrpheeReport{}, errors.New("le fichier Excel est vide")
	}

	report := OrpheeReport{BS: strings.TrimSpace(cell(rows[0], 4))}
	for _, row := range rows {
		if strings.EqualFold(strings.TrimSpace(cell(row, 1)), "A") {
			continue
		}
		nom := strings.TrimSpace(cell(row, 2))
		prenom := strings.TrimSpace(cell(row, 3))
		if nom == "" && prenom == "" {
			continue
		}
		report.Benevoles = append(report.Benevoles, Benevole{
			ID:      len(report.Benevoles) + 1,
			Nom:     nom,
			Prenom:  prenom,
			Donneur: strings.TrimSpace(cell(row, 5)),
		})
	}

	if len(report.Benevoles) == 0 {
		return OrpheeReport{}, errors.New("aucun bénévole trouvé : le fichier choisi n'est pas un rapport Orphée")
	}
	return report, nil
}

// importJournal reads a detailed journal exported by another installation of
// the program, so its lines can be merged into this one.
func importJournal(path string) ([]Entry, error) {
	rows, err := sheetRows(path)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 || strings.TrimSpace(cell(rows[0], 0)) != detailHeader {
		return nil, fmt.Errorf("le fichier choisi n'est pas conforme : la cellule A1 doit contenir « %s »", detailHeader)
	}
	if len(rows[0]) < 10 {
		return nil, errors.New("le fichier choisi n'est pas conforme : 10 colonnes sont attendues")
	}

	var entries []Entry
	for i, row := range rows[1:] {
		line := i + 2
		if strings.TrimSpace(strings.Join(row, "")) == "" {
			continue // trailing empty line
		}

		date, err := parseDate(cell(row, 1))
		if err != nil {
			return nil, fmt.Errorf("ligne %d : %w", line, err)
		}
		nom := strings.TrimSpace(cell(row, 2))
		prenom := strings.TrimSpace(cell(row, 3))
		if nom == "" && prenom == "" {
			return nil, fmt.Errorf("ligne %d : nom et prénom manquants", line)
		}

		hours := make([]int, 5)
		for k := range hours {
			hours[k], err = hoursCell(row, 4+k)
			if err != nil {
				return nil, fmt.Errorf("ligne %d : %w", line, err)
			}
		}

		entries = append(entries, Entry{
			Date:        date,
			Nom:         nom,
			Prenom:      prenom,
			HP:          hours[0],
			HD:          hours[1],
			HL:          hours[2],
			HR:          hours[3],
			HN:          hours[4],
			Commentaire: truncateComment(cell(row, 9)),
		})
	}

	if len(entries) == 0 {
		return nil, errors.New("le fichier ne contient aucune écriture à injecter")
	}
	return entries, nil
}

// sheetRows returns the rows of the first sheet of a workbook.
func sheetRows(path string) ([][]string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("impossible d'ouvrir « %s » : %w", path, err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, errors.New("le fichier Excel ne contient aucune feuille")
	}
	sheet := sheets[0]
	if active := f.GetSheetName(f.GetActiveSheetIndex()); active != "" {
		sheet = active
	}

	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("lecture de la feuille « %s » : %w", sheet, err)
	}
	return rows, nil
}

func cell(row []string, i int) string {
	if i < 0 || i >= len(row) {
		return ""
	}
	return row[i]
}

// hoursCell reads an hour count written by Excel, which may come back as
// "3", "3.0" or an empty string.
func hoursCell(row []string, i int) (int, error) {
	raw := strings.TrimSpace(cell(row, i))
	if raw == "" {
		return 0, nil
	}
	if n, err := strconv.Atoi(raw); err == nil {
		return n, nil
	}
	if f, err := strconv.ParseFloat(strings.Replace(raw, ",", ".", 1), 64); err == nil {
		return int(math.Round(f)), nil
	}
	return 0, fmt.Errorf("« %s » n'est pas un nombre d'heures", raw)
}
