# HB_version5 — Heures de bénévolat

Application de saisie des heures de bénévolat d'une BS (Bibliothèque Sonore) de
l'Association des Donneurs de Voix. Réécriture en Go, avec l'interface
[Shirei](https://go.hasen.dev/shirei), du programme Python/Tkinter `HB_version5.py`
d'Elie Couzinié. Les données, le schéma de la base et les fichiers Excel produits
sont ceux des versions précédentes : une installation existante s'ouvre sans
conversion.

## Utilisation

Les menus reprennent ceux du programme d'origine :

| Menu | Entrées |
|---|---|
| **Fichier** | Initialisation du fichier des bénévoles · Sauvegarde · Restauration · Quitter |
| **Action** | Saisie heures permanence · Saisie heures individuelles · Saisie globale de consolidation · Consolidation par journal externe · Edition du journal |
| **À propos …** | Infos · Lisez-moi … |

Chaque entrée du menu « Action » suit le même déroulé :

1. **Paramètres** — date, durée, catégorie ou encadrement de dates (contrôles de
   vraisemblance) ;
2. **Saisie / Consultation** — les bénévoles à gauche, les lignes préparées à
   droite (ou le résultat de l'édition) ;
3. **Enregistrement** — une seule transaction, puis un message de confirmation.

Les durées négatives servent à rectifier une erreur : elles s'affichent en rouge
et diminuent d'une unité le comptage des présences, comme le décrit le mode
d'emploi (menu « À propos … ▸ Lisez-moi … »).

## Dossier de travail

macOS : `~/HB_version5` — ailleurs : le répertoire courant.

| Fichier | Rôle |
|---|---|
| `HB_version5.db` (`mabase.db` hors macOS) | la base SQLite (`BEN`, `BS`, `JOURNAL`) |
| `sauvegarde.db`, `sauvegarde_old.db` | les deux dernières sauvegardes |
| `sauvegarde_avant_restauration.db` | copie de sécurité prise avant une restauration |
| `Journal_benevolat.xlsx` | journal détaillé (c'est le fichier que l'on injecte sur un autre poste) |
| `Journal_recap.xlsx`, `Journal_recapindiv.xlsx` | récapitulatif et comptage individuel |

## Installation

Les paquets prêts à l'emploi sont publiés sur la page **Releases** :

| Machine | Archive |
|---|---|
| Mac Apple Silicon / Intel / les deux | `…-macos-apple-silicon.zip` · `…-macos-intel.zip` · `…-macos-universal.zip` |
| Windows x86-64 / ARM64 | `…-windows-amd64.zip` · `…-windows-arm64.zip` |
| Linux x86-64 / ARM64 | `…-linux-amd64.tar.gz` · `…-linux-arm64.tar.gz` |

Décompressez, lancez : ni Python, ni Excel, ni base de données à installer.
Chaque archive contient un `INSTALLATION.txt`, et le pas-à-pas complet
(premier lancement sous macOS et Windows, mise à jour, sauvegarde,
désinstallation) est dans **[INSTALL.md](INSTALL.md)**.

## Développement

```sh
make            # liste toutes les cibles
make run        # lance l'application
make check      # gofmt + go vet + tests
make screens    # rend tous les écrans en PNG dans build/screens/
make install    # installe sur cette machine (macOS : ~/Applications, Linux : ~/.local)
```

Sans `make`, les commandes `go` habituelles fonctionnent :

```sh
go run .                              # lance l'application
go test ./...                         # dates, agrégats, base, Excel
go run . -png ecran.png -screen home  # rend un écran sans ouvrir de fenêtre
```

## Paquets de distribution

```sh
make dist              # tout ce que cette machine peut construire
make dist-macos        # Intel, Apple Silicon et universel (.app + .zip)
make dist-windows      # amd64 et arm64 (.zip, application graphique)
make dist-linux        # amd64 et arm64 (.tar.gz + install.sh + .desktop)
make checksums         # dist/SHA256SUMS.txt
```

Les archives sortent dans `dist/`, avec un numéro de version pris de
`git describe` (ou `make dist VERSION=1.0.0`) qui est aussi compilé dans le
binaire (`hb_go_version5 -version`, et « À propos … ▸ Infos »).

macOS passe par Cocoa, donc par **cgo** : ses paquets se construisent sur un
Mac (Xcode Command Line Tools) ; le bundle `.app` est assemblé par le
Makefile — `Info.plist`, icône `.icns` dérivée de l'illustration, ressources,
signature ad hoc — puis compressé avec `ditto`. Windows et Linux se compilent
**sans cgo** (backends Win32 / X11 / Wayland en Go pur) : leurs quatre
binaires peuvent être produits depuis n'importe quelle machine.

Les gabarits de paquet sont dans `packaging/` (`Info.plist.in`,
`hb_go_version5.desktop.in`, `install-linux.sh`, les `INSTALLATION-*.txt`
embarqués dans les archives et les notes de release).

## Intégration continue

| Workflow | Déclencheur | Rôle |
|---|---|---|
| `.github/workflows/ci.yml` | chaque push et chaque PR | `make check`, compilation sur Linux/macOS/Windows, et construction des sept paquets (artéfacts conservés 7 jours) |
| `.github/workflows/release.yml` | tag `v*` ou lancement manuel | construit les sept paquets, calcule `SHA256SUMS.txt` et publie la release GitHub |

Publier une version :

```sh
git tag v1.0.0 && git push origin v1.0.0
```

`-screen` accepte : `home`, `empty`, `perm-params`, `perm-saisie`,
`indiv-saisie`, `indiv-form`, `cons-form`, `inject-fichier`, `inject-controle`,
`journal-params`, `journal-detail`, `journal-recap`, `journal-indiv`, `about`,
`confirm`. Le préfixe `db-` (`-screen db-journal`) utilise la base réelle au lieu
des données d'exemple.

### Organisation du code

| Fichier | Contenu |
|---|---|
| `main.go`, `app.go` | point d'entrée, état de l'application, écrans, dialogues partagés |
| `model.go`, `dates.go`, `reports.go` | bénévoles, écritures, contrôles de saisie, agrégats |
| `db.go`, `paths.go`, `xlsx.go` | base SQLite, emplacements des fichiers, imports/exports Excel |
| `theme.go` | palette et composants d'interface partagés |
| `ui_shell.go`, `ui_home.go`, `ui_entry.go`, `ui_journal.go`, `ui_inject.go`, `ui_file.go` | les écrans |
| `preview.go` | rendu hors fenêtre (`-png`), données d'exemple |
| `Makefile`, `packaging/` | compilation, paquets macOS/Windows/Linux, installation locale |

Les ressources (`Resources/`) sont l'illustration et les deux modes d'emploi ;
elles sont recopiées dans le paquet par `shirei_bundle`.
