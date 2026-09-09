package main

import (
	"time"

	. "go.hasen.dev/shirei/widgets"
)

// Screen is the page currently filling the window. Every action of the menu
// bar leads to one of them; each one follows the same three beats:
// parameters, work, validation.
type Screen int

const (
	ScreenHome Screen = iota
	ScreenEntry
	ScreenInject
	ScreenJournal
)

// App is the whole state of the program: the data read from the database,
// the current screen and the state of each workflow. Shirei rebuilds the
// interface from this structure at every frame.
type App struct {
	paths Paths
	store *Store
	data  Reference
	dbErr string

	screen  Screen
	entry   EntryScreen
	inject  InjectScreen
	journal JournalScreen

	picker  FilePicker
	confirm Confirm
	about   bool
}

var app = &App{}

// Setup opens the database and loads the reference data.
func (a *App) Setup() {
	paths, err := NewPaths()
	if err != nil {
		a.dbErr = err.Error()
		return
	}
	a.paths = paths

	store, err := OpenStore(paths.DB)
	if err != nil {
		a.dbErr = err.Error()
		return
	}
	a.store = store
	a.Reload()
}

// Reload re-reads the reference data after the database has changed.
func (a *App) Reload() {
	if a.store == nil {
		return
	}
	data, err := a.store.Load()
	if err != nil {
		a.dbErr = err.Error()
		return
	}
	a.dbErr = ""
	a.data = data
}

// Ready reports whether hours can be recorded, which needs a volunteer list.
func (a *App) Ready() bool { return a.dbErr == "" && a.data.Initialised() }

// BSName is the name of the "BS" shown in the header and written in the
// workbooks.
func (a *App) BSName() string {
	if a.data.BS == "" {
		return "—"
	}
	return a.data.BS
}

// Benevole finds a volunteer by identifier.
func (a *App) Benevole(id int) *Benevole {
	for i := range a.data.Benevoles {
		if a.data.Benevoles[i].ID == id {
			return &a.data.Benevoles[i]
		}
	}
	return nil
}

// GoHome leaves the current workflow.
func (a *App) GoHome() { a.screen = ScreenHome }

// Notify reports a successful operation.
func (a *App) Notify(title, message string) {
	ToastExt(ToastAttrs{
		Icon:       SymPass,
		Title:      title,
		Body:       message,
		Background: ToastBackgroundSuccess,
		Duration:   6 * time.Second,
	})
}

// NotifyInfo reports a neutral piece of information.
func (a *App) NotifyInfo(title, message string) {
	ToastExt(ToastAttrs{
		Icon:       SymInfo,
		Title:      title,
		Body:       message,
		Background: ToastBackgroundInfo,
		Duration:   8 * time.Second,
	})
}

// NotifyError reports a failure; those stay longer on screen.
func (a *App) NotifyError(title, message string) {
	ToastExt(ToastAttrs{
		Icon:       SymWarn,
		Title:      title,
		Body:       message,
		Background: ToastBackgroundDanger,
		Duration:   12 * time.Second,
	})
}

// Confirm is the shared confirmation dialog, used before every operation
// that cannot be undone.
type Confirm struct {
	open     bool
	title    string
	message  string
	okLabel  string
	danger   bool
	onAccept func()
}

// Ask opens the confirmation dialog.
func (c *Confirm) Ask(title, message, okLabel string, danger bool, onAccept func()) {
	*c = Confirm{
		open:     true,
		title:    title,
		message:  message,
		okLabel:  okLabel,
		danger:   danger,
		onAccept: onAccept,
	}
}

func (c *Confirm) close() { *c = Confirm{} }

// FilePicker is the shared "choose a file" dialog.
type FilePicker struct {
	open     bool
	title    string
	cwd      string
	filter   string
	selected int
	exts     []string
	onPick   func(path string)
}

// Open shows the picker; onPick receives the chosen file.
func (p *FilePicker) Open(title, start string, exts []string, onPick func(string)) {
	*p = FilePicker{
		open:     true,
		title:    title,
		cwd:      start,
		selected: -1,
		exts:     exts,
		onPick:   onPick,
	}
}

func (p *FilePicker) close() { *p = FilePicker{} }
