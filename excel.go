package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// exportDetail mirrors go_to_excel_detail: writes Journal_benevolat.xlsx.
func exportDetail(rows []JournalRow) (string, error) {
	f := excelize.NewFile()
	defer f.Close()
	sheet := "Sheet1"
	tit := []string{"Journal détaillé", "Date", "Nom", "Prénom ", "H.Perm.", "H.ext.", "H.lect.", "H.Rég.", "H.Nat.", "Commentaire"}
	for i, t := range tit {
		f.SetCellValue(sheet, fmt.Sprintf("%s1", colName(i+1)), t)
	}
	for i, r := range rows {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", i+2), nomBS)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", i+2), r.DateFR)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", i+2), r.Nom)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", i+2), r.Prenom)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", i+2), r.HP)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", i+2), r.HD)
		f.SetCellValue(sheet, fmt.Sprintf("G%d", i+2), r.HL)
		f.SetCellValue(sheet, fmt.Sprintf("H%d", i+2), r.HR)
		f.SetCellValue(sheet, fmt.Sprintf("I%d", i+2), r.HN)
		f.SetCellValue(sheet, fmt.Sprintf("J%d", i+2), r.Comment)
	}
	f.SetColWidth(sheet, "A", "A", float64(len(nomBS)+2))
	f.SetColWidth(sheet, "B", "B", 11)
	f.SetColWidth(sheet, "C", "C", 20)
	f.SetColWidth(sheet, "D", "D", 20)
	f.SetColWidth(sheet, "J", "J", 30)
	for _, c := range []string{"E", "F", "G", "H", "I"} {
		f.SetColWidth(sheet, c, c, 6)
	}
	p := filepath.Join(appDir, "Journal_benevolat.xlsx")
	return p, f.SaveAs(p)
}

// exportRecap mirrors go_to_excel_recap: writes Journal_recap.xlsx.
func exportRecap(d1, d2 string, hp, hd, hl, hr, hn, tg int) (string, error) {
	f := excelize.NewFile()
	defer f.Close()
	sheet := "Sheet1"
	tit := []string{"BS de ", "du ", "au ", "H.Perm.", "H.ext.", "H.lect.", "H.Rég.", "H.Nat.", "Total"}
	for i, t := range tit {
		f.SetCellValue(sheet, fmt.Sprintf("%s1", colName(i+1)), t)
	}
	f.SetCellValue(sheet, "A2", nomBS)
	vals := []any{d1, d2, hp, hd, hl, hr, hn, tg}
	for i, v := range vals {
		f.SetCellValue(sheet, fmt.Sprintf("%s2", colName(i+2)), v)
	}
	f.SetColWidth(sheet, "A", "A", 20)
	f.SetColWidth(sheet, "B", "B", 11)
	f.SetColWidth(sheet, "C", "C", 11)
	for _, c := range []string{"D", "E", "F", "G", "H", "I"} {
		f.SetColWidth(sheet, c, c, 8)
	}
	p := filepath.Join(appDir, "Journal_recap.xlsx")
	return p, f.SaveAs(p)
}

// exportIndiv mirrors go_to_excel_indiv: writes Journal_recapindiv.xlsx.
// v layout per member: [nbPer, HP, HD, HL, HR, HN].
func exportIndiv(d1, d2 string, members map[string][6]int) (string, error) {
	f := excelize.NewFile()
	defer f.Close()
	sheet := "Sheet1"
	tit := []string{"BS de ", "du ", "au ", "Nom Prénom", "NB perm.", "H.Perm.", "H.ext.", "H.lect.", "H.Rég.", "H.Nat.", "Total"}
	for i, t := range tit {
		f.SetCellValue(sheet, fmt.Sprintf("%s1", colName(i+1)), t)
	}
	f.SetCellValue(sheet, "A2", nomBS)
	f.SetCellValue(sheet, "B2", d1)
	f.SetCellValue(sheet, "C2", d2)
	keys := make([]string, 0, len(members))
	for k := range members {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	l := 3
	for _, k := range keys {
		v := members[k]
		f.SetCellValue(sheet, fmt.Sprintf("D%d", l), k)
		tot := 0
		for z := 0; z < len(v); z++ {
			tot += v[z]
			f.SetCellValue(sheet, fmt.Sprintf("%s%d", colName(5+z), l), v[z])
		}
		f.SetCellValue(sheet, fmt.Sprintf("K%d", l), tot-v[0])
		l++
	}
	f.SetColWidth(sheet, "A", "A", 15)
	f.SetColWidth(sheet, "B", "B", 11)
	f.SetColWidth(sheet, "C", "C", 11)
	f.SetColWidth(sheet, "D", "D", 30)
	for _, c := range []string{"E", "F", "G", "H", "I", "J", "K"} {
		f.SetColWidth(sheet, c, c, 8)
	}
	p := filepath.Join(appDir, "Journal_recapindiv.xlsx")
	return p, f.SaveAs(p)
}

func colName(n int) string {
	s := ""
	for n > 0 {
		n--
		s = string(rune('A'+n%26)) + s
		n /= 26
	}
	return s
}

// readJournalXLSX mirrors the injection() read of an external
// "Journal détaillé" workbook.
func readJournalXLSX(path string) ([]JournalRow, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sheet := f.GetSheetName(0)
	cell, err := f.GetCellValue(sheet, "A1")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cell) != "Journal détaillé" {
		return nil, fmt.Errorf("non conforme")
	}
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, err
	}
	var out []JournalRow
	for i := 1; i < len(rows); i++ {
		r := rows[i]
		if len(r) < 10 {
			continue
		}
		var jr JournalRow
		if err := scanCell(r[1], &jr.DateFR); err != nil {
			return nil, err
		}
		jr.DateISO = frToISO(jr.DateFR)
		if err := scanCell(r[2], &jr.Nom); err != nil {
			return nil, err
		}
		if err := scanCell(r[3], &jr.Prenom); err != nil {
			return nil, err
		}
		var hp, hd, hl, hr, hn int
		var err2 error
		if hp, err2 = scanInt(r[4]); err2 != nil {
			return nil, err2
		}
		if hd, err2 = scanInt(r[5]); err2 != nil {
			return nil, err2
		}
		if hl, err2 = scanInt(r[6]); err2 != nil {
			return nil, err2
		}
		if hr, err2 = scanInt(r[7]); err2 != nil {
			return nil, err2
		}
		if hn, err2 = scanInt(r[8]); err2 != nil {
			return nil, err2
		}
		jr.HP, jr.HD, jr.HL, jr.HR, jr.HN = int64(hp), int64(hd), int64(hl), int64(hr), int64(hn)
		jr.Comment = r[9]
		out = append(out, jr)
	}
	return out, nil
}

// readInitXLSX mirrors initialise(): reads the orphelinat volunteer file.
// E1 holds the BS name; rows with B=="A" are skipped; C=nom, D=prenom, F=DDV.
func readInitXLSX(path string) (bsName string, out []Benevole, err error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return "", nil, err
	}
	defer f.Close()
	sheet := f.GetSheetName(0)
	if bsName, err = f.GetCellValue(sheet, "E1"); err != nil {
		return "", nil, err
	}
	rows, err := f.GetRows(sheet)
	if err != nil {
		return "", nil, err
	}
	for i, r := range rows {
		if i == 0 {
			continue
		}
		if len(r) < 6 {
			continue
		}
		if strings.TrimSpace(r[1]) == "A" {
			continue
		}
		var b Benevole
		b.Nom = r[2]
		b.Prenom = r[3]
		b.Donneur = r[5]
		out = append(out, b)
	}
	return bsName, out, nil
}

func scanCell(s string, dst *string) error {
	*dst = strings.TrimSpace(s)
	return nil
}

func scanInt(s string) (int, error) {
	return strconv.Atoi(strings.TrimSpace(s))
}
