package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Dates are typed by the user as "jj/mm/aaaa" and stored as "aaaa-mm-jj",
// which is the format the JOURNAL table has always used (and what SQLite
// needs for its date comparisons).
const (
	dateFormatFR  = "02/01/2006"
	dateFormatISO = "2006-01-02"
)

// today is a variable so the tests can freeze the clock.
var today = func() time.Time {
	n := time.Now()
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
}

func formatFR(t time.Time) string  { return t.Format(dateFormatFR) }
func formatISO(t time.Time) string { return t.Format(dateFormatISO) }

// parseISO reads a date as stored in the database.
func parseISO(s string) (time.Time, error) {
	return time.ParseInLocation(dateFormatISO, strings.TrimSpace(s), time.UTC)
}

// parseDate validates a date typed by the user and reports the same
// "contrôles de vraisemblance" as the original program, with one message per
// kind of mistake.
func parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, errors.New("la date est obligatoire")
	}
	if len([]rune(s)) != 10 {
		return time.Time{}, errors.New("date erronée : elle doit comporter 10 caractères (jj/mm/aaaa)")
	}

	fields := strings.Split(s, "/")
	if len(fields) != 3 {
		return time.Time{}, errors.New("date erronée : format attendu jj/mm/aaaa")
	}

	day, err := strconv.Atoi(fields[0])
	if err != nil {
		return time.Time{}, errors.New("jour erroné")
	}
	month, err := strconv.Atoi(fields[1])
	if err != nil {
		return time.Time{}, errors.New("mois erroné")
	}
	year, err := strconv.Atoi(fields[2])
	if err != nil {
		return time.Time{}, errors.New("année erronée")
	}

	if month < 1 || month > 12 {
		return time.Time{}, errors.New("mois erroné")
	}
	if day < 1 || day > daysInMonth(year, month) {
		return time.Time{}, errors.New("date invraisemblable")
	}

	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), nil
}

// parsePastDate is parseDate plus the rule that hours cannot be recorded in
// the future.
func parsePastDate(s string) (time.Time, error) {
	d, err := parseDate(s)
	if err != nil {
		return time.Time{}, err
	}
	if d.After(today()) {
		return time.Time{}, errors.New("la date saisie est postérieure à la date du jour")
	}
	return d, nil
}

// parseDateRange validates the two bounds of a report.
func parseDateRange(fromText, toText string) (time.Time, time.Time, error) {
	from, err := parseDate(fromText)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("date de départ : %w", err)
	}
	to, err := parseDate(toText)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("date de fin : %w", err)
	}
	if from.After(today()) {
		return time.Time{}, time.Time{}, errors.New("la date de départ est postérieure à la date du jour")
	}
	if from.After(to) {
		return time.Time{}, time.Time{}, errors.New("la date de départ est postérieure à la date de fin")
	}
	return from, to, nil
}

// parseHours reads a whole number of hours. Negative values are allowed on
// purpose: that is how a wrong entry is cancelled.
func parseHours(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("durée erronée : indiquez un nombre d'heures entier")
	}
	h, err := strconv.Atoi(s)
	if err != nil {
		return 0, errors.New("durée erronée : indiquez un nombre d'heures entier")
	}
	return h, nil
}

func daysInMonth(year, month int) int {
	switch month {
	case 1, 3, 5, 7, 8, 10, 12:
		return 31
	case 4, 6, 9, 11:
		return 30
	case 2:
		if isLeapYear(year) {
			return 29
		}
		return 28
	}
	return 0
}

func isLeapYear(y int) bool {
	return (y%4 == 0 && y%100 != 0) || y%400 == 0
}
