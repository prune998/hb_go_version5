package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// The schema is the one created by the Python versions of the program, kept
// as is so an existing database opens unchanged:
//
//	BEN     ( Id, nom, prenom, Donneur )        the volunteers
//	BS      ( nom_bs )                          the name of the "BS"
//	JOURNAL ( Date, nom, prenom, HP, HL, HD, HR, HN, commentaire )
const (
	sqlCreateBen = `create table if not exists BEN (
		[Id] INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
		[nom] NVARCHAR(20) NOT NULL,
		[prenom] NVARCHAR(20) NOT NULL,
		[Donneur] NVARCHAR(3))`

	sqlCreateBS = `create table if not exists BS ([nom_bs] NVARCHAR(50) NOT NULL)`

	sqlCreateJournal = `create table if not exists JOURNAL (
		[Date] DATE NOT NULL,
		[nom] NVARCHAR(20) NOT NULL,
		[prenom] NVARCHAR(20) NOT NULL,
		[HP] INTEGER, [HL] INTEGER, [HD] INTEGER, [HR] INTEGER, [HN] INTEGER,
		[commentaire] NVARCHAR(30))`
)

// Store is the whole database access layer of the program.
type Store struct {
	db   *sql.DB
	path string
}

// OpenStore opens (creating it when needed) the database and brings its
// schema up to date.
func OpenStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("ouverture de la base : %w", err)
	}
	// The program is single user and every write is a short transaction:
	// one connection keeps SQLite out of "database is locked" territory.
	db.SetMaxOpenConns(1)

	store := &Store{db: db, path: path}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// Path is the database file the store is bound to.
func (s *Store) Path() string { return s.path }

// migrate creates the tables of a fresh database and adds the comment column
// to journals written by version 4 and older.
func (s *Store) migrate() error {
	for _, stmt := range []string{sqlCreateBen, sqlCreateBS, sqlCreateJournal} {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("création de la base : %w", err)
		}
	}

	has, err := s.hasColumn("JOURNAL", "commentaire")
	if err != nil {
		return err
	}
	if !has {
		if _, err := s.db.Exec(`ALTER TABLE JOURNAL ADD commentaire VARCHAR(30)`); err != nil {
			return fmt.Errorf("mise à jour de la base : %w", err)
		}
	}
	return nil
}

func (s *Store) hasColumn(table, column string) (bool, error) {
	rows, err := s.db.Query(fmt.Sprintf("pragma table_info(%s)", table))
	if err != nil {
		return false, fmt.Errorf("lecture du schéma : %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid                 int
			name, ctype         string
			notNull, primaryKey int
			defaultValue        sql.NullString
		)
		if err := rows.Scan(&cid, &name, &ctype, &notNull, &defaultValue, &primaryKey); err != nil {
			return false, fmt.Errorf("lecture du schéma : %w", err)
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

// Reference is everything the interface needs to know about the database.
type Reference struct {
	BS           string
	Benevoles    []Benevole
	JournalCount int
	FirstDate    time.Time
	LastDate     time.Time
}

// Initialised reports whether the volunteer list has been imported yet.
func (r Reference) Initialised() bool { return len(r.Benevoles) > 0 }

// Load reads the reference data: the name of the BS, the volunteers and a
// few counters shown in the interface.
func (s *Store) Load() (Reference, error) {
	var ref Reference

	if err := s.db.QueryRow("select nom_bs from BS limit 1").Scan(&ref.BS); err != nil && err != sql.ErrNoRows {
		return ref, fmt.Errorf("lecture du nom de la BS : %w", err)
	}

	bens, err := s.Benevoles()
	if err != nil {
		return ref, err
	}
	ref.Benevoles = bens

	// The dates are read back through strftime: the driver turns a column
	// declared DATE into a time.Time, while an expression stays plain text,
	// which is the format the table has always stored.
	var first, last sql.NullString
	err = s.db.QueryRow(`select count(*), strftime('%Y-%m-%d', min(Date)), strftime('%Y-%m-%d', max(Date))
		from JOURNAL`).Scan(&ref.JournalCount, &first, &last)
	if err != nil {
		return ref, fmt.Errorf("lecture du journal : %w", err)
	}
	if first.Valid {
		ref.FirstDate, _ = parseISO(first.String)
	}
	if last.Valid {
		ref.LastDate, _ = parseISO(last.String)
	}
	return ref, nil
}

// Benevoles lists the volunteers in the order the interface displays them.
func (s *Store) Benevoles() ([]Benevole, error) {
	rows, err := s.db.Query("select Id, nom, prenom, coalesce(Donneur, '') from BEN order by nom, prenom")
	if err != nil {
		return nil, fmt.Errorf("lecture des bénévoles : %w", err)
	}
	defer rows.Close()

	var list []Benevole
	for rows.Next() {
		var b Benevole
		if err := rows.Scan(&b.ID, &b.Nom, &b.Prenom, &b.Donneur); err != nil {
			return nil, fmt.Errorf("lecture des bénévoles : %w", err)
		}
		list = append(list, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("lecture des bénévoles : %w", err)
	}
	return list, nil
}

// AddEntries appends lines to the journal in a single transaction: either
// the whole entry screen is recorded, or nothing is.
func (s *Store) AddEntries(entries []Entry) error {
	if len(entries) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("mise à jour de la base : %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`insert into JOURNAL (Date, nom, prenom, HP, HD, HL, HR, HN, commentaire)
		values (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("mise à jour de la base : %w", err)
	}
	defer stmt.Close()

	for _, e := range entries {
		_, err := stmt.Exec(formatISO(e.Date), e.Nom, e.Prenom, e.HP, e.HD, e.HL, e.HR, e.HN, truncateComment(e.Commentaire))
		if err != nil {
			return fmt.Errorf("mise à jour de la base : %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("mise à jour de la base : %w", err)
	}
	return nil
}

// EntriesBetween reads the journal for a period, in chronological order.
func (s *Store) EntriesBetween(from, to time.Time) ([]Entry, error) {
	rows, err := s.db.Query(`select strftime('%Y-%m-%d', Date), nom, prenom,
			coalesce(HP, 0), coalesce(HD, 0), coalesce(HL, 0), coalesce(HR, 0), coalesce(HN, 0),
			coalesce(commentaire, '')
		from JOURNAL where Date between ? and ? order by Date, nom, prenom`,
		formatISO(from), formatISO(to))
	if err != nil {
		return nil, fmt.Errorf("lecture du journal : %w", err)
	}
	defer rows.Close()

	var list []Entry
	for rows.Next() {
		var (
			e       Entry
			rawDate string
		)
		if err := rows.Scan(&rawDate, &e.Nom, &e.Prenom, &e.HP, &e.HD, &e.HL, &e.HR, &e.HN, &e.Commentaire); err != nil {
			return nil, fmt.Errorf("lecture du journal : %w", err)
		}
		e.Date, _ = parseISO(rawDate)
		list = append(list, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("lecture du journal : %w", err)
	}
	return list, nil
}

// ReplaceBenevoles rewrites the volunteer list and the name of the BS from a
// freshly exported Orphée report. The journal is never touched.
func (s *Store) ReplaceBenevoles(bs string, list []Benevole) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("initialisation de la base : %w", err)
	}
	defer tx.Rollback()

	for _, stmt := range []string{
		"drop table if exists BEN",
		sqlCreateBen,
		"delete from BS",
	} {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("initialisation de la base : %w", err)
		}
	}

	insert, err := tx.Prepare("insert into BEN (Id, nom, prenom, Donneur) values (?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("initialisation de la base : %w", err)
	}
	defer insert.Close()

	for i, b := range list {
		if _, err := insert.Exec(i+1, b.Nom, b.Prenom, b.Donneur); err != nil {
			return fmt.Errorf("initialisation de la base : %w", err)
		}
	}

	if _, err := tx.Exec("insert into BS (nom_bs) values (?)", bs); err != nil {
		return fmt.Errorf("initialisation de la base : %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("initialisation de la base : %w", err)
	}
	return nil
}
