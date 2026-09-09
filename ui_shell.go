package main

import (
	"fmt"
	"os"

	. "go.hasen.dev/shirei"
	. "go.hasen.dev/shirei/widgets"
)

// RootView draws the whole window: a title band, the menu bar, the current
// screen, a status bar, and the dialogs on top.
func RootView() {
	Container(Attrs(Viewport, BackgroundVec(colPage),
		AmendTextStyle(FontSize(13), TextColorVec(colInk))), func() {
		TitleBand()
		MenuBar()

		Container(Attrs(Grow(1), Expand, Extrinsic, Clip), func() {
			switch app.screen {
			case ScreenEntry:
				EntryScreenView()
			case ScreenInject:
				InjectScreenView()
			case ScreenJournal:
				JournalScreenView()
			default:
				HomeScreenView()
			}
		})

		StatusBar()

		FilePickerDialog()
		ConfirmDialog()
		AboutDialog()
	})
}

// TitleBand is the dark strip naming the application and the BS.
func TitleBand() {
	Container(Attrs(Row, Expand, CrossMid, Gap(14), Pad2(12, 22), BackgroundVec(colHeader)), func() {
		Icon(TypBook, FontSize(24), TextColorVec(colOnDark))
		Container(Attrs(Gap(1)), func() {
			Label("HEURES DE BÉNÉVOLAT", FontSize(17), FontWeight(WeightBold), TextColorVec(colOnDark))
			Label("Association des Donneurs de Voix — version 5", FontSize(11), TextColorVec(colOnDarkSoft))
		})
		Filler(1)
		Container(Attrs(CrossAlign(AlignEnd), Gap(2)), func() {
			Label("BS de "+app.BSName(), FontSize(14), FontWeight(WeightBold), TextColorVec(colOnDark))
			Label("Nous sommes le "+formatFR(today()), FontSize(11), TextColorVec(colOnDarkSoft))
		})
	})
}

// MenuBar keeps the three menus of the original program, with the same
// entries in the same order.
func MenuBar() {
	Container(Attrs(Row, Expand, CrossMid, Gap(6), Pad2(6, 16), BackgroundVec(colHeaderDim)), func() {
		MenuButton(SymFolder, "Fichier", fileMenu)
		MenuButton(SymList, "Action", actionMenu)
		MenuButton(SymInfo, "À propos …", aboutMenu)

		Filler(1)

		if app.Ready() {
			Label(fmt.Sprintf("%d bénévoles · %d écritures au journal",
				len(app.data.Benevoles), app.data.JournalCount),
				FontSize(11), TextColorVec(colOnDarkSoft))
		} else {
			Label("Base non initialisée", FontSize(11), FontWeight(WeightBold), TextColorVec(Vec4{40, 80, 80, 1}))
		}
	})
}

func fileMenu() {
	if MenuItem(SymUpload, "Initialisation du fichier des bénévoles") {
		startInitialisation()
	}
	MenuSeparator()
	if MenuItemExt("Sauvegarde", ButtonAttrs{Icon: SymDrive, Disabled: app.store == nil}) {
		runBackup()
	}
	MenuSeparator()
	if MenuItem(SymDownload, "Restauration") {
		startRestore()
	}
	MenuSeparator()
	if MenuItem(SymPower, "Quitter") {
		quit()
	}
}

func actionMenu() {
	ready := app.Ready()

	if MenuItemExt("Saisie heures permanence", ButtonAttrs{Icon: SymGroup, Disabled: !ready}) {
		app.openEntry(KindPermanence)
	}
	MenuSeparator()
	if MenuItemExt("Saisie heures individuelles", ButtonAttrs{Icon: SymUser, Disabled: !ready}) {
		app.openEntry(KindIndividuelle)
	}
	MenuSeparator()
	if MenuItemExt("Saisie globale de consolidation", ButtonAttrs{Icon: SymGrid, Disabled: !ready}) {
		app.openEntry(KindConsolidation)
	}
	MenuSeparator()
	if MenuItemExt("Consolidation par journal externe", ButtonAttrs{Icon: SymExternal, Disabled: !ready}) {
		app.openInject()
	}
	MenuSeparator()
	if MenuItemExt("Edition du journal", ButtonAttrs{Icon: SymFile, Disabled: !ready}) {
		app.openJournal()
	}

	if !ready {
		MenuSeparator()
		Container(Attrs(Pad2(4, 10), MaxWidth(280)), func() {
			Label("Commencez par « Fichier ▸ Initialisation du fichier des bénévoles ».",
				FontSize(11), TextColorVec(colInkSoft))
		})
	}
}

func aboutMenu() {
	if MenuItem(SymInfo, "Infos") {
		app.about = true
	}
	MenuSeparator()
	if MenuItem(TypDocumentText, "Lisez-moi …") {
		openManual()
	}
}

// StatusBar shows where the data lives; the manual asks the user to back up
// that folder regularly.
func StatusBar() {
	Container(Attrs(Row, Expand, CrossMid, Gap(10), Pad2(6, 16), BackgroundVec(colHeaderDim)), func() {
		Icon(SymDrive, FontSize(11), TextColorVec(colOnDarkSoft))
		if app.store != nil {
			Label(app.store.Path(), FontSize(11), TextColorVec(colOnDarkSoft))
		} else {
			Label("aucune base ouverte", FontSize(11), TextColorVec(colOnDarkSoft))
		}
		Filler(1)
		if app.dbErr != "" {
			Label(app.dbErr, FontSize(11), FontWeight(WeightBold), TextColorVec(Vec4{40, 85, 78, 1}))
		}
	})
}

// ---------------------------------------------------------------- dialogs

// FilePickerDialog is the shared file chooser: the same dialog serves the
// Orphée import and the injection of an external journal.
func FilePickerDialog() {
	picker := &app.picker
	if !picker.open {
		return
	}

	Modal(620, picker.close, func() {
		Label(picker.title, FontSize(15), FontWeight(WeightBold), TextColorVec(colInk))
		Label("Seuls les fichiers Excel (.xlsx) sont proposés.", FontSize(11), TextColorVec(colInkSoft))

		attrs := FileBrowserAttrs{
			Title: picker.title,
			Width: 580,
			Files: true,
			Exts:  picker.exts,
		}

		var chosen string
		if FileBrowserPanel(&picker.cwd, &picker.filter, &picker.selected, &chosen, attrs) {
			onPick := picker.onPick
			picker.close()
			if onPick != nil && chosen != "" {
				onPick(chosen)
			}
			// The dialog runs in the popup pass, after the screen was built:
			// whatever it opened is only visible on the next frame.
			RequestNextFrame()
			return
		}

		Container(Attrs(Row, Expand, CrossMid, Gap(10)), func() {
			Filler(1)
			if GhostButton(SymCancel, "Annuler") {
				picker.close()
			}
		})
	})
}

// ConfirmDialog asks before an operation that rewrites data.
func ConfirmDialog() {
	confirm := &app.confirm
	if !confirm.open {
		return
	}

	Modal(520, confirm.close, func() {
		Container(Attrs(Row, CrossMid, Gap(10)), func() {
			icon, tint := SymInfo, colAccent
			if confirm.danger {
				icon, tint = SymWarn, colDanger
			}
			Icon(icon, FontSize(20), TextColorVec(tint))
			Label(confirm.title, FontSize(16), FontWeight(WeightBold), TextColorVec(colInk))
		})
		Container(Attrs(Expand, MaxWidth(470)), func() {
			Label(confirm.message, FontSize(13), TextColorVec(colInk))
		})
		Container(Attrs(Row, Expand, CrossMid, Gap(10), Pad4(6, 0, 0, 0)), func() {
			Filler(1)
			if GhostButton(SymCancel, "Annuler") {
				confirm.close()
				return
			}
			accept := confirm.onAccept
			label := confirm.okLabel
			var clicked bool
			if confirm.danger {
				clicked = DangerButton(SymPass, label)
			} else {
				clicked = PrimaryButton(SymPass, label)
			}
			if clicked {
				confirm.close()
				if accept != nil {
					accept()
				}
				RequestNextFrame()
			}
		})
	})
}

// AboutDialog is the "Infos" entry of the "À propos" menu.
func AboutDialog() {
	if !app.about {
		return
	}
	close := func() { app.about = false }

	Modal(520, close, func() {
		Container(Attrs(Row, CrossMid, Gap(12)), func() {
			Icon(TypBook, FontSize(24), TextColorVec(colAccent))
			Container(Attrs(Gap(2)), func() {
				Label("HB_version5", FontSize(17), FontWeight(WeightBold), TextColorVec(colInk))
				Label("Saisie des heures de bénévolat — Elie Couzinié", FontSize(12), TextColorVec(colInkSoft))
			})
		})
		Divider()
		Container(Attrs(Expand, Gap(6)), func() {
			aboutLine("BS", app.BSName())
			aboutLine("Bénévoles", fmt.Sprintf("%d", len(app.data.Benevoles)))
			aboutLine("Écritures au journal", fmt.Sprintf("%d", app.data.JournalCount))
			aboutLine("Dossier de travail", app.paths.Dir)
			aboutLine("Version", version)
			aboutLine("Interface", "Go + Shirei")
		})
		Container(Attrs(Row, Expand, CrossMid, Gap(10), Pad4(6, 0, 0, 0)), func() {
			if GhostButton(TypDocumentText, "Lisez-moi …") {
				openManual()
			}
			Filler(1)
			if PrimaryButton(SymPass, "Fermer") {
				close()
			}
		})
	})
}

func aboutLine(label, value string) {
	Container(Attrs(Row, Expand, CrossAlign(AlignStart), Gap(10)), func() {
		Container(Attrs(FixWidth(160)), func() {
			Label(label, FontSize(12), FontWeight(WeightBold), TextColorVec(colInkSoft))
		})
		Container(Attrs(Grow(1), Clip, MaxWidth(textWrapWidth(180))), func() {
			Label(value, FontSize(12), TextColorVec(colInk))
		})
	})
}

func openManual() {
	path := manualPath()
	if path == "" {
		app.NotifyError("Lisez-moi", "Le mode d'emploi n'a pas été trouvé à côté du programme.")
		return
	}
	if err := openInDesktop(path); err != nil {
		app.NotifyError("Lisez-moi", "Impossible d'ouvrir « "+path+" ».")
	}
}

// quit is a variable so the tests never stop the test binary.
var quit = func() { os.Exit(0) }
