package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// This file implements the three maintenance operations of the "Fichier"
// menu. Each one follows the same shape as the actions: a choice, a
// confirmation, then a report.

// pickerStartDir is where the file picker opens: the Downloads folder, since
// that is where Orphée reports land, and the home directory otherwise.
func pickerStartDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return app.paths.Dir
	}
	downloads := filepath.Join(home, "Downloads")
	if st, err := os.Stat(downloads); err == nil && st.IsDir() {
		return downloads
	}
	return home
}

// startInitialisation imports the volunteer list from an Orphée report.
func startInitialisation() {
	if app.store == nil {
		app.NotifyError("Initialisation", "Aucune base n'est ouverte.")
		return
	}

	app.picker.Open("Sélectionner le fichier Orphée des bénévoles", pickerStartDir(), []string{".xlsx"},
		func(path string) {
			report, err := importOrphee(path)
			if err != nil {
				app.NotifyError("Initialisation impossible", err.Error())
				return
			}

			message := fmt.Sprintf(
				"%d bénévoles ont été lus dans « %s », pour la BS « %s ».\n\n"+
					"La liste actuelle (%d bénévoles) sera remplacée. Le journal des heures n'est pas modifié.\n\n"+
					"Vérifiez que vous avez choisi le bon fichier.",
				len(report.Benevoles), filepath.Base(path), report.BS, len(app.data.Benevoles))

			app.confirm.Ask("Initialisation du fichier des bénévoles", message, "Remplacer la liste", true,
				func() { applyInitialisation(report) })
		})
}

func applyInitialisation(report OrpheeReport) {
	if err := app.store.ReplaceBenevoles(report.BS, report.Benevoles); err != nil {
		app.NotifyError("Initialisation impossible", err.Error())
		return
	}
	app.Reload()
	app.GoHome()
	app.Notify("Mise à jour terminée",
		fmt.Sprintf("%d bénévoles enregistrés pour la BS « %s ».", len(report.Benevoles), report.BS))
}

// runBackup copies the database, keeping the previous copy as
// "sauvegarde_old.db" exactly like the earlier versions did.
func runBackup() {
	if app.store == nil {
		app.NotifyError("Sauvegarde", "Aucune base n'est ouverte.")
		return
	}
	if err := backupDatabase(); err != nil {
		app.NotifyError("La sauvegarde n'a pas pu être exécutée", err.Error())
		return
	}
	app.Notify("Sauvegarde réussie", "Fichier "+app.paths.Backup)
}

// backupDatabase rotates the backups and writes a fresh one.
func backupDatabase() error {
	paths := app.paths

	if fileExists(paths.BackupOld) {
		if err := os.Remove(paths.BackupOld); err != nil {
			return err
		}
	}
	if fileExists(paths.Backup) {
		if err := os.Rename(paths.Backup, paths.BackupOld); err != nil {
			return err
		}
	}
	return copyFile(paths.DB, paths.Backup)
}

// startRestore puts back the last backup, after telling the user how old it
// is and what is about to happen.
func startRestore() {
	paths := app.paths

	if !fileExists(paths.Backup) {
		app.NotifyError("Restauration", "Fichier de sauvegarde non trouvé : "+paths.Backup)
		return
	}

	when := "date inconnue"
	if st, err := os.Stat(paths.Backup); err == nil {
		when = formatFR(st.ModTime())
	}

	message := fmt.Sprintf(
		"La sauvegarde « %s » est datée du %s.\n\n"+
			"Elle va remplacer la base actuelle. Par sécurité, la base actuelle sera d'abord copiée dans « %s ».",
		paths.Backup, when, filepath.Base(paths.BackupBeforeRestore))

	app.confirm.Ask("Restauration", message, "Restaurer la sauvegarde", true, applyRestore)
}

func applyRestore() {
	paths := app.paths

	if app.store != nil {
		if fileExists(paths.DB) {
			if err := copyFile(paths.DB, paths.BackupBeforeRestore); err != nil {
				app.NotifyError("Restauration impossible", err.Error())
				return
			}
		}
		if err := app.store.Close(); err != nil {
			app.NotifyError("Restauration impossible", err.Error())
			return
		}
		app.store = nil
	}

	if err := copyFile(paths.Backup, paths.DB); err != nil {
		app.dbErr = err.Error()
		app.NotifyError("Restauration impossible", err.Error())
		return
	}

	store, err := OpenStore(paths.DB)
	if err != nil {
		app.dbErr = err.Error()
		app.NotifyError("Restauration impossible", err.Error())
		return
	}
	app.store = store
	app.Reload()
	app.GoHome()
	app.Notify("Restauration réussie", "La base a été remplacée par « "+paths.Backup+" ».")
}
