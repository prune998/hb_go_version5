package main

import (
	"fmt"

	. "go.hasen.dev/shirei"
	. "go.hasen.dev/shirei/widgets"
)

// HomeScreenView is the landing page: the state of the database, and one
// card per entry of the "Action" menu.
func HomeScreenView() {
	Page(func() {
		if !app.Ready() {
			NotInitialisedCard()
		} else {
			DashboardCard()
		}
		ActionCards()
		FileActionsCard()
	})
}

// DashboardCard sums up the database in a few figures.
func DashboardCard() {
	Card(func() {
		Container(Attrs(Row, Expand, Gap(20), CrossAlign(AlignStart)), func() {
			Container(Attrs(Grow(1), Expand, Extrinsic, Clip, Gap(14)), func() {
				CardHeader(SymHome, "BS de "+app.BSName(),
					"Saisissez les heures de bénévolat, puis éditez le journal de la période.")

				Container(Attrs(Row, Expand, Gap(12), Wrap), func() {
					StatTile("Bénévoles", fmt.Sprintf("%d", len(app.data.Benevoles)),
						fmt.Sprintf("dont %d DDV", countDDV(app.data.Benevoles)))
					StatTile("Écritures au journal", fmt.Sprintf("%d", app.data.JournalCount), journalPeriod())
					StatTile("Dernière saisie", lastEntryText(), "")
				})
			})
			if banner := bannerPath(); banner != "" {
				Image(banner, Vec2{0, 150})
			}
		})
	})
}

// NotInitialisedCard replaces the dashboard while the volunteer list is
// missing, and repeats the procedure described in the manual.
func NotInitialisedCard() {
	Card(func() {
		CardHeader(SymWarn, "Base non initialisée", "")
		ErrorBanner("Aucun bénévole n'a été trouvé dans la base. La saisie des heures est indisponible.")

		Container(Attrs(Expand, Gap(6)), func() {
			SectionTitle("POUR DÉMARRER")
			Label("1. Dans Orphée, lancez le rapport « Bénévoles_liste pour heures » et enregistrez-le au format Excel.",
				FontSize(12), TextColorVec(colInk))
			Label("2. Choisissez « Fichier ▸ Initialisation du fichier des bénévoles » et désignez ce fichier.",
				FontSize(12), TextColorVec(colInk))
			Label("3. Si vous disposez d'une sauvegarde, vous pouvez aussi la restaurer.",
				FontSize(12), TextColorVec(colInk))
		})

		Container(Attrs(Row, Expand, CrossMid, Gap(10)), func() {
			if PrimaryButton(SymUpload, "Initialisation du fichier des bénévoles") {
				startInitialisation()
			}
			if GhostButton(SymDownload, "Restauration") {
				startRestore()
			}
		})
	})
}

// action describes one entry of the "Action" menu, so the menu and the home
// page can never drift apart.
type action struct {
	icon  IconGlyph
	title string
	hint  string
	open  func()
}

func actions() []action {
	return []action{
		{SymGroup, "Saisie heures permanence",
			"Créditer d'un même nombre d'heures tous les bénévoles présents à une permanence.",
			func() { app.openEntry(KindPermanence) }},
		{SymUser, "Saisie heures individuelles",
			"Saisir des heures hors permanence, une catégorie à la fois.",
			func() { app.openEntry(KindIndividuelle) }},
		{SymGrid, "Saisie globale de consolidation",
			"Reporter en une fois le récapitulatif d'activité d'un bénévole.",
			func() { app.openEntry(KindConsolidation) }},
		{SymExternal, "Consolidation par journal externe",
			"Injecter le journal détaillé Excel d'un autre poste.",
			app.openInject},
		{SymFile, "Edition du journal",
			"Éditer le journal détaillé, le récapitulatif ou le comptage individuel.",
			app.openJournal},
	}
}

// ActionCards is the visual twin of the "Action" menu.
func ActionCards() {
	Card(func() {
		CardHeader(SymList, "Actions", "Les mêmes entrées que le menu « Action ».")
		Container(Attrs(Row, Expand, Gap(12), Wrap, CrossAlign(AlignStart)), func() {
			for _, a := range actions() {
				ActionCard(a)
			}
		})
	})
}

// ActionCard is one clickable action tile.
func ActionCard(a action) {
	ready := app.Ready()

	ContainerWithKey(a.title, Attrs(FixWidth(280), MinHeight(96), Gap(8), Pad(14), Corners(10),
		BackgroundVec(colZebra), BorderWidth(1), BorderColorVec(colLine)), func() {
		if !ready {
			ModAttrs(Trans(0.45))
		} else {
			if IsHovered() {
				ModAttrs(BackgroundVec(colAccentSoft), BorderColorVec(colAccent))
			}
			if PressAction() {
				a.open()
			}
		}

		Container(Attrs(Row, CrossMid, Gap(8)), func() {
			Icon(a.icon, FontSize(16), TextColorVec(colAccent))
			Container(Attrs(MaxWidth(210)), func() {
				Label(a.title, FontSize(13), FontWeight(WeightBold), TextColorVec(colInk))
			})
		})
		Container(Attrs(MaxWidth(250)), func() {
			Label(a.hint, FontSize(11), TextColorVec(colInkSoft))
		})
	})
}

// FileActionsCard groups the maintenance operations of the "Fichier" menu.
func FileActionsCard() {
	Card(func() {
		CardHeader(SymFolder, "Fichier", "Sauvegardez la base à chaque utilisation et conservez les copies en lieu sûr.")
		Container(Attrs(Row, Expand, CrossMid, Gap(10), Wrap), func() {
			if GhostButton(SymUpload, "Initialisation du fichier des bénévoles") {
				startInitialisation()
			}
			if GhostButton(SymDrive, "Sauvegarde") {
				runBackup()
			}
			if GhostButton(SymDownload, "Restauration") {
				startRestore()
			}
			Filler(1)
			if GhostButton(TypDocumentText, "Lisez-moi …") {
				openManual()
			}
		})
	})
}

func countDDV(list []Benevole) int {
	n := 0
	for _, b := range list {
		if b.IsDDV() {
			n++
		}
	}
	return n
}

func journalPeriod() string {
	if app.data.JournalCount == 0 {
		return "journal vide"
	}
	return "du " + formatFR(app.data.FirstDate) + " au " + formatFR(app.data.LastDate)
}

func lastEntryText() string {
	if app.data.JournalCount == 0 {
		return "—"
	}
	return formatFR(app.data.LastDate)
}
