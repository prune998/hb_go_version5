package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"runtime"

	. "go.hasen.dev/shirei"
	shapp "go.hasen.dev/shirei/app"
)

// Window size: Python used one 850x450 window plus separate Toplevels up to
// 1400x750 (cons select) / 1200x800 (journal). Shirei renders dialogs as
// modals inside the single window, so it must be large enough for all of them.
const (
	winW = 1400
	winH = 750
)

var quitFn = os.Exit

func quitApp() { quitFn(0) }

func main() {
	// Debug renderer: render one settled frame headlessly to a PNG, the
	// standard shirei way to verify UI without opening a window.
	if len(os.Args) >= 3 && os.Args[1] == "--png" {
		initPaths()
		_ = openDB()
		_ = loadInitial()
		if len(os.Args) >= 4 {
			preparePngScreen(os.Args[3])
		}
		if err := RenderToPNG(os.Args[2], winW, winH, frame); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	initPaths()
	if err := openDB(); err != nil {
		fmt.Fprintln(os.Stderr, "ERREUR: ", err)
		os.Exit(1)
	}
	if err := loadInitial(); err != nil {
		// No usable DB (missing or empty file): let the UI offer to pick a
		// local database or start from an empty one.
		app.screen = scrNoDB
	}

	shapp.SetupWindow("APPLICATION : HEURES BENEVOLAT", winW, winH)
	if icon := bannerPath(); icon != "" {
		if img := logoImage(icon); img != nil {
			shapp.SetupIconImage(img)
		}
	}
	shapp.Run(frame)
}

// preparePngScreen positions the app for a headless render (debug --png only).
func preparePngScreen(name string) {
	if len(bens) == 0 {
		nomBS = "ELIE"
		bens = []Benevole{
			{ID: 1, Nom: "CUIVAS", Prenom: "Jean", Donneur: " "},
			{ID: 2, Nom: "DUPONT", Prenom: "Marie", Donneur: "DDV"},
			{ID: 3, Nom: "FERREIRA", Prenom: "Paulo", Donneur: " "},
			{ID: 4, Nom: "GARCIA", Prenom: "Lucia", Donneur: " "},
			{ID: 5, Nom: "MARTIN", Prenom: "Sophie", Donneur: " "},
			{ID: 6, Nom: "PETIT", Prenom: "André", Donneur: "DDV"},
		}
	}
	switch name {
	case "perm-ask":
		app.screen = scrPermAsk
	case "perm-select":
		app.screen = scrPermSelect
		app.permDate = "03/09/2026"
		app.permDateISO = "2026-09-03"
		app.permDuree = "3"
		app.permDureeInt = 3
		app.permSel = []int{0, 1, 2}
	case "indiv-select":
		app.screen = scrIndivSelect
		app.indivRows = []selEntry{
			{idx: 1, dateISO: "2026-09-03", duree: 2, cat: 2, comm: "Lecture à l'hopital"},
			{idx: 4, dateISO: "2026-09-03", duree: -1, cat: 1, comm: ""},
		}
	case "indiv-fields":
		app.screen = scrIndivFields
		app.indivPending = 2
	case "cons-select":
		app.screen = scrConsSelect
		app.consRows = []selEntry{
			{idx: 0, dateISO: "2026-09-03", hp: 3, hd: 0, hl: 0, hr: 1, hn: 0, comm: "Semaine complète"},
			{idx: 3, dateISO: "2026-09-03", hp: 0, hd: 2, hl: 4, hr: 0, hn: 0, comm: ""},
		}
	case "cons-fields":
		app.screen = scrConsFields
		app.consPending = 5
	case "init-select", "inject-select":
		app.fileCwd, _ = os.UserHomeDir()
		app.screen = scrInitSelect
		if name == "inject-select" {
			app.screen = scrInjectSelect
		}
	case "journal-ask":
		app.screen = scrJournalAsk
	case "journal-view":
		app.screen = scrJournalView
		app.jD1, app.jD2 = "01/09/2026", "03/09/2026"
		app.jRows = []JournalRow{
			{DateISO: "2026-09-01", DateFR: "01/09/2026", Nom: "CUIVAS", Prenom: "Jean", HP: 3, HD: 0, HL: 0, HR: 0, HN: 0, Comment: ""},
			{DateISO: "2026-09-02", DateFR: "02/09/2026", Nom: "DUPONT", Prenom: "Marie", HP: 0, HD: 0, HL: 2, HR: 0, HN: 0, Comment: "Lecture"},
			{DateISO: "2026-09-03", DateFR: "03/09/2026", Nom: "PETIT", Prenom: "André", HP: 3, HD: 1, HL: 0, HR: 2, HN: 1, Comment: ""},
		}
	case "no-db":
		app.screen = scrNoDB
	case "no-db-file":
		app.fileCwd, _ = os.UserHomeDir()
		app.screen = scrNoDBFile
	case "err-msg":
		app.showErr("ERREUR", "LE BÉNÉVOLE DOIT ÊTRE DDV")
	case "info-msg":
		app.showInfo("MESSAGE", "LA BASE A ÉTÉ MISE À JOUR.")
	}
}

// bannerPath locates the bundled Image_logiciel.png (shirei Resources dir).
func bannerPath() string {
	for _, p := range []string{
		shapp.ResourcePath("Image_logiciel.png"),
		"Resources/Image_logiciel.png",
		"Image_logiciel.png",
	} {
		if fileExists(p) {
			return p
		}
	}
	return ""
}

func logoImage(path string) image.Image {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	return img
}

// openHelp mirrors dum2: opens the user manual with the platform opener.
func openHelp() {
	p := shapp.ResourcePath("Mode_d_emploi_benevolat_v5.docx")
	if !fileExists(p) {
		p = "Mode_d_emploi_benevolat_v5.docx"
	}
	if !fileExists(p) {
		app.showErr("ERREUR", "IMPOSSIBLE DE CHARGER LE FICHIER D'AIDE\n '"+p+"'")
		return
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", p)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", p)
	default:
		cmd = exec.Command("xdg-open", p)
	}
	if err := cmd.Start(); err != nil {
		app.showErr("ERREUR", "IMPOSSIBLE D'OUVRIR LE FICHIER D'AIDE\n '"+p+"'")
	}
}
