package main

import (
	"strconv"
	"time"
	"unicode/utf8"
)

// qMois reproduces the Python q_mois leap-year day tables (index 0 = common, 1 = leap).
var qMois = [2][13]int{
	{0, 31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31},
	{0, 31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31},
}

func today() time.Time { return time.Now() }

// todayFR returns dd/mm/yyyy, the format the app uses for date entries.
func todayFR() string { return today().Format("02/01/2006") }

// todayISO returns yyyy-mm-dd, the format stored in JOURNAL.Date.
func todayISO() string { return today().Format("2006-01-02") }

// frToISO converts dd/mm/yyyy to yyyy-mm-dd (the Python slice trick).
func frToISO(s string) string {
	if len(s) != 10 {
		return ""
	}
	return s[6:10] + "-" + s[3:5] + "-" + s[0:2]
}

func isLeap(y int) bool { return y%4 == 0 && y%100 != 0 || y%400 == 0 }

// validateDateFR mirrors the Python single-date checks. It returns the French
// uppercase error message, or "" when the date is valid.
func validateDateFR(s string) string {
	if len(s) != 10 {
		return "SAISIE DATE ERRONÉE\n NOMBRE DE CARACTÈRES DIFFÉRENT DE 10"
	}
	if s == "" {
		return "LA DATE SAISIE NE PEUT PAS ÊTRE ÉGALE À 0"
	}
	j, err := strconv.Atoi(s[0:2])
	if err != nil {
		return "JOUR ERRONÉ"
	}
	m, err := strconv.Atoi(s[3:5])
	if err != nil {
		return "MOIS ERRONÉ"
	}
	a, err := strconv.Atoi(s[6:10])
	if err != nil {
		return "ANNÉE ERRONÉE"
	}
	if m < 1 || m > 12 {
		return "MOIS ERRONÉ"
	}
	leap := 0
	if isLeap(a) {
		leap = 1
	}
	if j < 1 || j > qMois[leap][m] {
		return "DATE INVRAISEMBLABLE"
	}
	if frToISO(s) > todayISO() {
		return "LA DATE SAISIE EST POSTÉRIEURE\nÀ LA DATE DU JOUR"
	}
	return ""
}

// validateRange mirrors the Python j() checks on the two journal dates.
func validateRange(d1, d2 string) string {
	if len(d1) != 10 || len(d2) != 10 {
		return "UNE AU MOINS DES DATES EST ERRONÉE\n NOMBRE DE CARACTÈRES DIFFÉRENT DE 10"
	}
	if d1 == "" || d2 == " " {
		return "LA DATE SAISIE NE PEUT PAS ÊTRE ÉGALE À 0"
	}
	jd, err := strconv.Atoi(d1[0:2])
	if err != nil {
		return "JOUR ERRONÉ"
	}
	ja, err := strconv.Atoi(d2[0:2])
	if err != nil {
		return "JOUR ERRONÉ"
	}
	md, err := strconv.Atoi(d1[3:5])
	if err != nil {
		return "MOIS ERRONÉ"
	}
	ma, err := strconv.Atoi(d2[3:5])
	if err != nil {
		return "MOIS ERRONÉ"
	}
	ad, err := strconv.Atoi(d1[6:10])
	if err != nil {
		return "ANNÉE ERRONÉE"
	}
	aa, err := strconv.Atoi(d2[6:10])
	if err != nil {
		return "ANNÉE ERRONÉE"
	}
	if md < 1 || md > 12 || ma < 1 || ma > 12 {
		return "MOIS ERRONÉ"
	}
	if jd < 1 || jd > qMois[boolInt(isLeap(ad))][md] {
		return "DATE DE DÉPART INVRAISEMBLABLE"
	}
	if ja < 1 || ja > qMois[boolInt(isLeap(aa))][ma] {
		return "DATE DE FIN INVRAISEMBLABLE"
	}
	limiteb := frToISO(d1)
	limiteh := frToISO(d2)
	if limiteb > todayISO() {
		return "LA DATE DE DÉPART EST POSTÉRIEURE\nÀ LA DATE DU JOUR"
	}
	if limiteb > limiteh {
		return "LA DATE DE DÉPART EST POSTÉRIEURE\nÀ LA DATE DE FIN"
	}
	return ""
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func padRight(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + repeat(" ", w-len(s))
}

func padCenter(s string, w int) string {
	if len(s) >= w {
		return s
	}
	left := (w - len(s)) / 2
	return repeat(" ", left) + s + repeat(" ", w-len(s)-left)
}

func repeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}

// padRightRunes is the rune-aware twin of padRight (accents are single cells).
func padRightRunes(s string, w int) string {
	if utf8.RuneCountInString(s) >= w {
		return s
	}
	return s + repeat(" ", w-utf8.RuneCountInString(s))
}

// cutRunes mirrors the Python text[0:30] truncation on a rune basis.
func cutRunes(s string, n int) string {
	rs := []rune(s)
	if len(rs) <= n {
		return s
	}
	return string(rs[:n])
}
