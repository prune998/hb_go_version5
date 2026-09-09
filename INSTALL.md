# Installation de « Heures de bénévolat »

Le programme est un fichier unique : ni Python, ni Excel, ni base de données à
installer. Une base créée par une version précédente (Python) est reprise
telle quelle.

## 1. Télécharger

Les paquets sont publiés sur la page **Releases** du dépôt. Choisissez
l'archive qui correspond à votre machine :

| Votre machine | Archive à télécharger |
|---|---|
| Mac Apple Silicon (M1, M2, M3, M4) | `HB_version5-<version>-macos-apple-silicon.zip` |
| Mac Intel | `HB_version5-<version>-macos-intel.zip` |
| Mac — si vous ne savez pas | `HB_version5-<version>-macos-universal.zip` |
| PC Windows habituel (Intel ou AMD) | `HB_version5-<version>-windows-amd64.zip` |
| PC Windows ARM (Snapdragon, Surface ARM) | `HB_version5-<version>-windows-arm64.zip` |
| PC Linux 64 bits | `HB_version5-<version>-linux-amd64.tar.gz` |
| Linux ARM 64 bits (Raspberry Pi 64 bits…) | `HB_version5-<version>-linux-arm64.tar.gz` |

> **Quel Mac ai-je ?** Menu  ▸ « À propos de ce Mac » : « Puce Apple M… »
> pour Apple Silicon, « Processeur Intel… » pour Intel.

Chaque archive contient un fichier `INSTALLATION.txt` qui reprend les étapes
ci-dessous.

## 2. Installer

### macOS

1. Double-cliquez sur le `.zip` (il se décompresse tout seul).
2. Glissez **`HB_version5.app`** dans votre dossier **Applications**.
3. **Premier lancement uniquement** : clic droit (ou Ctrl + clic) sur l'icône
   ▸ **Ouvrir** ▸ **Ouvrir**. macOS demande cette confirmation parce que
   l'application n'est pas signée par un certificat Apple payant. Ensuite, un
   double-clic suffit.

   Si macOS annonce que l'application « est endommagée », ouvrez le Terminal :

   ```sh
   xattr -dr com.apple.quarantine /Applications/HB_version5.app
   ```

4. Pour l'avoir dans le Dock : glissez l'icône depuis Applications.

La base et les fichiers Excel sont créés dans **`~/HB_version5`**.

### Windows

1. Clic droit sur le `.zip` ▸ **« Extraire tout … »**, par exemple dans
   `Documents\HB_version5`.
   Gardez `hb_go_version5.exe` et le dossier `Resources` **côte à côte**.
2. Double-cliquez sur **`hb_go_version5.exe`**.
3. **Premier lancement uniquement** : si Windows affiche « Windows a protégé
   votre ordinateur », cliquez sur **« Informations complémentaires »** puis
   **« Exécuter quand même »**.
4. Pour un accès rapide : clic droit sur le `.exe` ▸ « Envoyer vers ▸ Bureau
   (créer un raccourci) », ou « Épingler à la barre des tâches ».

La base (`mabase.db`) et les fichiers Excel sont créés **dans le dossier où
vous avez décompressé l'archive**.

### Linux

```sh
tar xzf HB_version5-*-linux-amd64.tar.gz
cd HB_version5
./install.sh            # installe dans ~/.local (menu des applications inclus)
```

Autre emplacement : `PREFIX=/opt ./install.sh`.
Sans rien installer : `./hb_go_version5` (en gardant `Resources` à côté).

Prérequis : un bureau Wayland ou X11 ; aucune bibliothèque supplémentaire.
La base (`mabase.db`) et les fichiers Excel sont créés dans le répertoire
courant.

## 3. Première utilisation

1. Dans **Orphée**, widget « Rapports », lancez
   « Bénévoles_liste pour heures … » et enregistrez le résultat au format
   **Excel** (`.xlsx`).
2. Dans le programme : menu **« Fichier ▸ Initialisation du fichier des
   bénévoles »**, désignez ce fichier, puis confirmez.
3. La saisie des heures est disponible. Le mode d'emploi complet s'ouvre par
   **« À propos … ▸ Lisez-moi … »**.

À refaire à chaque changement dans la liste Orphée (arrivée, départ,
changement de statut DDV) : le journal des heures n'est jamais modifié par
cette opération.

## 4. Sauvegarder

Faites **« Fichier ▸ Sauvegarde »** à chaque utilisation, puis copiez les
fichiers `sauvegarde.db` / `sauvegarde_old.db` en lieu sûr (clé USB, espace en
ligne). En cas d'accident, **« Fichier ▸ Restauration »** remet la dernière
sauvegarde en place (la base remplacée est conservée sous
`sauvegarde_avant_restauration.db`).

## 5. Mettre à jour

Téléchargez la nouvelle archive et remplacez l'application :

- **macOS** : glissez la nouvelle `HB_version5.app` dans Applications en
  acceptant le remplacement ;
- **Windows** : décompressez la nouvelle archive **par-dessus** l'ancienne (ou
  à côté, puis recopiez votre base) ;
- **Linux** : relancez `./install.sh` depuis la nouvelle archive.

La base n'est pas touchée par une mise à jour. Faites tout de même une
sauvegarde avant.

## 6. Désinstaller

- **macOS** : mettez `HB_version5.app` à la corbeille. Vos données restent
  dans `~/HB_version5`.
- **Windows** : supprimez le dossier décompressé (il contient aussi vos
  données : copiez-les d'abord si besoin).
- **Linux** : `./install.sh --uninstall` (les données du répertoire de travail
  ne sont pas touchées).

## 7. Compiler soi-même

Il suffit de [Go](https://go.dev/dl/) (version indiquée dans `go.mod`) :

```sh
git clone https://github.com/prune998/hb_go_version5.git
cd hb_go_version5
make run                 # lance l'application
make install             # macOS : ~/Applications · Linux : ~/.local
make dist                # toutes les archives dans dist/
```

macOS passe par Cocoa (cgo) : les paquets macOS se construisent sur un Mac
(Xcode Command Line Tools). Windows et Linux se compilent sans cgo, donc
depuis n'importe quelle machine — `make dist-windows dist-linux` fonctionne
aussi bien sur un Mac que sur un PC Linux. `make help` liste toutes les
cibles.
