package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	shapp "go.hasen.dev/shirei/app"
)

const appName = "HB_version5"

// Generated workbook names. They are fixed (and overwritten at every
// edition) because the manual tells the user where to find them, and
// because "Journal_benevolat.xlsx" is also the file the injection feature
// consumes on another machine.
const (
	fileJournalDetail = "Journal_benevolat.xlsx"
	fileJournalRecap  = "Journal_recap.xlsx"
	fileJournalIndiv  = "Journal_recapindiv.xlsx"
)

// Paths holds every file location the program uses. Everything lives in one
// folder, laid out exactly like the Python version so an existing
// installation keeps working: a dedicated directory in the user's home on
// macOS, the working directory elsewhere.
type Paths struct {
	Dir                 string
	DB                  string
	Backup              string
	BackupOld           string
	BackupBeforeRestore string
}

// NewPaths resolves the data directory and creates it when needed.
func NewPaths() (Paths, error) {
	dir := dataDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Paths{}, err
	}
	return Paths{
		Dir:                 dir,
		DB:                  filepath.Join(dir, dbFileName()),
		Backup:              filepath.Join(dir, "sauvegarde.db"),
		BackupOld:           filepath.Join(dir, "sauvegarde_old.db"),
		BackupBeforeRestore: filepath.Join(dir, "sauvegarde_avant_restauration.db"),
	}, nil
}

// Export returns the full path of a generated workbook.
func (p Paths) Export(name string) string { return filepath.Join(p.Dir, name) }

func dataDir() string {
	if runtime.GOOS == "darwin" {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, appName)
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}
	return "."
}

func dbFileName() string {
	if runtime.GOOS == "darwin" {
		return "HB_version5.db"
	}
	return "mabase.db"
}

// manualPath locates the user manual shipped with the program. It is looked
// up in the Shirei resources directory first (that is where the packaged
// application keeps it), then next to the executable, then in the working
// directory for `go run`.
func manualPath() string {
	names := []string{"Mode_d_emploi_benevolat_MAC_v5.docx", "Mode_d_emploi_benevolat_v5.docx"}
	if runtime.GOOS != "darwin" {
		names[0], names[1] = names[1], names[0]
	}

	var dirs []string
	if d := shapp.ResourcesDir(); d != "" {
		dirs = append(dirs, d)
	}
	if exe, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Dir(exe))
	}
	if cwd, err := os.Getwd(); err == nil {
		dirs = append(dirs, cwd, filepath.Join(cwd, "Resources"))
	}

	for _, dir := range dirs {
		for _, name := range names {
			if candidate := filepath.Join(dir, name); fileExists(candidate) {
				return candidate
			}
		}
	}
	return ""
}

// bannerPath returns the illustration shown in the home screen, or "" when
// the resource is missing.
func bannerPath() string {
	p := shapp.ResourcePath("Image_logiciel.png")
	if fileExists(p) {
		return p
	}
	return ""
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

// openInDesktop hands a file over to the desktop environment, which is how
// the program shows the generated workbooks and the manual.
func openInDesktop(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", path).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}

// copyFile copies src over dst, truncating an existing destination.
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}
