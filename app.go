package main

// screen enumerates the dialogs; the frame renders the current one as a Modal.
type screen int

const (
	scrMain screen = iota
	scrPermAsk // date + durée standard
	scrPermSelect
	scrIndivSelect
	scrIndivFields
	scrConsSelect
	scrConsFields
	scrJournalAsk
	scrJournalView
	scrInitSelect // choix du fichier orphelinat
	scrInjectSelect
	scrNoDB      // pas de base : choisir un fichier .db ou partir d'une base vide
	scrNoDBFile  // sélecteur de fichier pour scrNoDB
)

type msgKind int

const (
	msgNone msgKind = iota
	msgInfo
	msgErr
)

// selEntry is one row of the "selected" list, carrying the per-member data
// the insertion flows need (Python kept the same data in liste_permanence).
type selEntry struct {
	idx     int
	dateISO string
	duree   int // perm + indiv
	cat     int // indiv: 1=Extérieur 2=Lecture DDV 3=Régional 4=National
	hp, hd, hl, hr, hn int // cons
	comm    string
	red     bool
}

// App holds every piece of UI state; the frame loop mutates it directly
// (shirei frames run on the UI loop, like tkinter callbacks).
type App struct {
	screen screen

	// message overlay
	msg      msgKind
	msgTitle string
	msgText  string
	msgQuit  bool // OK on this message exits the app (initialisation & fatal errors)

	// permanence
	permDate     string
	permDateISO  string
	permDuree    string
	permDureeInt int
	permSel      []int

	// individuel
	indivPending int // member index waiting for the fields dialog
	indivDate    string
	indivDuree   string
	indivCat     int
	indivCom     string
	indivRows    []selEntry

	// consolidation
	consPending int
	consDate    string
	consHP      string
	consHD      string
	consHL      string
	consHR      string
	consHN      string
	consCom     string
	consRows    []selEntry

	// journal
	jMode     int
	jD1       string
	jD2       string
	jRows     []JournalRow
	jSum      [6]int // HP, HD, HL, HR, HN, TG
	jMembers  map[string][6]int
	jExcelMsg string
	jErr      string

	// file selection (init + injection) — FileBrowserPanel bindings
	fileCwd    string
	fileFilter string
	fileSelIdx int
	fileSel    string
}

var app = &App{
	permDate:     todayFR(),
	indivDate:    todayFR(),
	consDate:     todayFR(),
	jD1:          todayFR(),
	jD2:          todayFR(),
	permDuree:    "3",
	permDureeInt: 3,
	indivDuree:   "3",
	indivCat:     1,
	consHP: "0", consHD: "0", consHL: "0", consHR: "0", consHN: "0",
	jMode:      1,
	fileSelIdx: -1,
}

// showInfo / showErr set the message overlay (the Go twin of messagebox).
func (a *App) showInfo(title, text string) {
	a.msg, a.msgTitle, a.msgText = msgInfo, title, text
}

func (a *App) showErr(title, text string) {
	a.msg, a.msgTitle, a.msgText = msgErr, title, text
}

func (a *App) closeMsg() {
	a.msg = msgNone
	a.msgTitle, a.msgText = "", ""
	if a.msgQuit {
		a.msgQuit = false
		quitApp()
	}
}

// dismissTo is the modal close handler for a screen: nil while a message
// overlay is open, so Escape / scrim click only closes the topmost dialog.
func (a *App) dismissTo(scr screen) func() {
	if a.msg != msgNone {
		return nil
	}
	return func() { a.screen = scr }
}
