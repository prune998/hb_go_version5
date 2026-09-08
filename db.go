package main

import (
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	_ "modernc.org/sqlite"
)

var (
	appDir string
	dbPath string
	db     *sql.DB
	nomBS  string
	bens   []Benevole
)

// Benevole mirrors the BEN table.
type Benevole struct {
	ID      int
	Nom     string
	Prenom  string
	Donneur string
}

// JournalRow mirrors a JOURNAL row as the app displays it.
type JournalRow struct {
	DateISO   string
	DateFR    string
	Nom       string
	Prenom    string
	HP        int64 // permanence
	HD        int64 // extérieur
	HL        int64 // lecture
	HR        int64 // régional
	HN        int64 // national
	Comment   string
}

func initPaths() {
	const appName = "HB_version5"
	if runtime.GOOS == "darwin" {
		home, err := os.UserHomeDir()
		if err != nil {
			panic("cannot resolve home dir: " + err.Error())
		}
		appDir = filepath.Join(home, appName)
		dbPath = filepath.Join(appDir, "HB_version5.db")
		if _, err := os.Stat(appDir); os.IsNotExist(err) {
			if err := os.MkdirAll(appDir, 0o755); err != nil {
				panic("can't create app folder at " + appDir)
			}
		}
	} else {
		wd, err := os.Getwd()
		if err != nil {
			panic("cannot resolve cwd: " + err.Error())
		}
		appDir = wd
		dbPath = filepath.Join(wd, "mabase.db")
	}
}

// openDB opens (creating the file if needed) and runs the migration.
// A fresh/empty file without tables is NOT an error: the caller decides
// what to do with it (noDB flow).
func openDB() error {
	if db != nil {
		db.Close()
		db = nil
	}
	var err error
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	// Same migration as the Python app: add the commentaire column if missing.
	var n int
	if err := db.QueryRow("select count(*) from pragma_table_info('JOURNAL') where name = 'commentaire'").Scan(&n); err == nil && n == 0 {
		var t int
		if err := db.QueryRow("select count(*) from sqlite_master where type='table' and name='JOURNAL'").Scan(&t); err != nil {
			return err
		}
		if t > 0 {
			if _, err := db.Exec("ALTER TABLE JOURNAL ADD commentaire VARCHAR(30)"); err != nil {
				return err
			}
		}
	}
	return nil
}

// createEmptyDB builds a standard-schema database with no journal rows; the
// user then fills it via the normal initialisation flow.
func createEmptyDB() error {
	if err := openDB(); err != nil {
		return err
	}
	stmts := []string{
		"create table if not exists BEN(" +
			"[Id] INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL," +
			"[nom] NVARCHAR(20) NOT NULL,[prenom] NVARCHAR(20) NOT NULL ,[Donneur] NVARCHAR(3)) ",
		"create table if not exists JOURNAL ([Date] DATE  NOT NULL," +
			"[nom] NVARCHAR(20)  NOT NULL,[prenom] NVARCHAR(20)  NOT NULL" +
			", [HP] INTEGER,[HL] INTEGER, [HD] INTEGER,[HR] INTEGER ,[HN] INTEGER, [commentaire] NVARCHAR(30)) ",
		"create table if not exists BS ([nom_bs] NVARCHAR(50)  NOT NULL)",
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	var n int
	if err := db.QueryRow("select count(*) from BS").Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		if _, err := db.Exec("insert into BS (nom_bs) values (?)", ""); err != nil {
			return err
		}
	}
	return loadInitial()
}

// adoptDB replaces the live database with a copy of src and reloads it.
func adoptDB(src string) error {
	if db != nil {
		db.Close()
		db = nil
	}
	if s, err := filepath.Abs(src); err == nil && s == filepath.Clean(dbPath) {
		// The picked file is the live one: just reopen it.
		if err := openDB(); err != nil {
			return err
		}
		return loadInitial()
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	out, err := os.Create(dbPath)
	if err != nil {
		in.Close()
		return err
	}
	_, err = io.Copy(out, in)
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	in.Close()
	if err != nil {
		return err
	}
	if err := openDB(); err != nil {
		return err
	}
	return loadInitial()
}

// loadInitial loads BEN (ordered) and the BS name, like the Python startup.
func loadInitial() error {
	bens = bens[:0]
	rows, err := db.Query("select nom, prenom, Donneur from BEN order by nom, prenom")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var b Benevole
		var donneur sql.NullString
		if err := rows.Scan(&b.Nom, &b.Prenom, &donneur); err != nil {
			return err
		}
		b.Donneur = donneur.String
		bens = append(bens, b)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	var n int
	if err := db.QueryRow("select count(*) from JOURNAL").Scan(&n); err != nil {
		return err
	}
	_ = n
	row := db.QueryRow("select nom_bs from BS")
	if err := row.Scan(&nomBS); err != nil {
		return err
	}
	return nil
}

// queryJournal runs the exact j() query: Date between the two ISO limits.
func queryJournal(fromISO, toISO string) ([]JournalRow, error) {
	rows, err := db.Query(
		"select Date, Strftime('%d/%m/%Y', Date), nom, prenom, HP, HD, HL, HR, HN, commentaire "+
			"from JOURNAL where Date between (?) and (?) order by Date, nom, prenom", fromISO, toISO)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []JournalRow
	for rows.Next() {
		var (
			r     JournalRow
			hp, hd, hl, hr, hn sql.NullInt64
			comm sql.NullString
		)
		if err := rows.Scan(&r.DateISO, &r.DateFR, &r.Nom, &r.Prenom, &hp, &hd, &hl, &hr, &hn, &comm); err != nil {
			return nil, err
		}
		r.HP, r.HD, r.HL, r.HR, r.HN = hp.Int64, hd.Int64, hl.Int64, hr.Int64, hn.Int64
		r.Comment = comm.String
		out = append(out, r)
	}
	return out, rows.Err()
}

// insertPermanence mirrors maj_journal: one JOURNAL row per selected member.
func insertPermanence(dateISO string, duree int, idxs []int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, i := range idxs {
		if i < 0 || i >= len(bens) {
			continue
		}
		b := bens[i]
		if _, err := tx.Exec("insert into JOURNAL (nom,prenom,Date,HP) values (?,?,?,?)",
			b.Nom, b.Prenom, dateISO, duree); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// indivInsert is one selected row of the individuel flow.
type indivInsert struct {
	idx     int
	dateISO string
	heure   int
	cat     int
	comm    string
}

// insertIndiv mirrors maj_journal_i: the category picks the hour column.
func insertIndiv(entries []indivInsert) error {
	colOf := func(cat int) string {
		if c := map[int]string{1: "HD", 2: "HL", 3: "HR", 4: "HN"}[cat]; c != "" {
			return c
		}
		return "HD"
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, e := range entries {
		if e.idx < 0 || e.idx >= len(bens) {
			continue
		}
		b := bens[e.idx]
		if _, err := tx.Exec(
			"insert into JOURNAL (nom,prenom,Date,"+colOf(e.cat)+",commentaire) values (?,?,?,?,?)",
			b.Nom, b.Prenom, e.dateISO, e.heure, e.comm); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// consInsert is one selected row of the consolidation flow.
type consInsert struct {
	idx     int
	dateISO string
	hp, hd, hl, hr, hn int
	comm    string
}

// insertConsList mirrors maj_journal_i_consolidation.
func insertConsList(entries []consInsert) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, e := range entries {
		if e.idx < 0 || e.idx >= len(bens) {
			continue
		}
		b := bens[e.idx]
		if _, err := tx.Exec(
			"insert into JOURNAL (nom,prenom,Date,HP,HD,HL,HR,HN,commentaire) values (?,?,?,?,?,?,?,?,?)",
			b.Nom, b.Prenom, e.dateISO, e.hp, e.hd, e.hl, e.hr, e.hn, e.comm); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// rebuildBEN mirrors initialise(): recreate BEN + BS from the Excel rows.
func rebuildBEN(bsName string, rows []Benevole) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmts := []string{
		"drop table if exists BEN",
		"create table BEN(" +
			"[Id] INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL," +
			"[nom] NVARCHAR(20) NOT NULL,[prenom] NVARCHAR(20) NOT NULL ,[Donneur] NVARCHAR(3)) ",
		"create table if not exists JOURNAL ([Date] DATE  NOT NULL," +
			"[nom] NVARCHAR(20)  NOT NULL,[prenom] NVARCHAR(20)  NOT NULL" +
			", [HP] INTEGER,[HL] INTEGER, [HD] INTEGER,[HR] INTEGER ,[HN] INTEGER, [commentaire] NVARCHAR(30)) ",
		"create table if not exists BS ([nom_bs] NVARCHAR(50)  NOT NULL)",
		"delete from BS",
	}
	for _, s := range stmts {
		if _, err := tx.Exec(s); err != nil {
			return err
		}
	}
	for i, b := range rows {
		if _, err := tx.Exec("insert into BEN (Id,nom,prenom,Donneur) values (?,?,?,?)",
			i+1, b.Nom, b.Prenom, b.Donneur); err != nil {
			return err
		}
	}
	if _, err := tx.Exec("insert into BS (nom_bs) values (?)", bsName); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return loadInitial()
}

// insertInjection mirrors injection(): bulk rows from an external journal.
func insertInjection(rows []JournalRow) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, r := range rows {
		if _, err := tx.Exec(
			"insert into JOURNAL (Date,nom,prenom,HP,HD,HL,HR,HN,commentaire) values (?,?,?,?,?,?,?,?,?)",
			r.DateISO, r.Nom, r.Prenom, r.HP, r.HD, r.HL, r.HR, r.HN, r.Comment); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// sauvegarde mirrors sauvegarde(): rotate the backup then copy the DB.
func sauvegarde() (string, error) {
	oldBkp := filepath.Join(appDir, "sauvegarde_old.db")
	newBkp := filepath.Join(appDir, "sauvegarde.db")
	if _, err := os.Stat(oldBkp); err == nil {
		os.Remove(oldBkp)
	}
	if _, err := os.Stat(newBkp); err == nil {
		os.Rename(newBkp, oldBkp)
	}
	src, err := os.Open(dbPath)
	if err != nil {
		return "", err
	}
	defer src.Close()
	dst, err := os.Create(newBkp)
	if err != nil {
		return "", err
	}
	defer dst.Close()
	if _, err := dst.ReadFrom(src); err != nil {
		return "", err
	}
	return newBkp, nil
}

// restauration mirrors restauration(): report info about the backup file.
// It does NOT copy yet (Python only copies when the live DB is missing).
func restauration() (dateFR string, liveExists bool, err error) {
	newBkp := filepath.Join(appDir, "sauvegarde.db")
	st, err := os.Stat(newBkp)
	if err != nil {
		return "", false, err
	}
	dateFR = time2fr(st.ModTime())
	liveExists = fileExists(dbPath)
	return dateFR, liveExists, nil
}

func time2fr(t time.Time) string { return t.Format("02/01/2006") }

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
