## Installation

Téléchargez l'archive correspondant à votre machine, puis suivez le fichier
`INSTALLATION.txt` qu'elle contient. Le dépôt en contient une version détaillée
dans `INSTALL.md`.

| Votre machine | Archive |
|---|---|
| Mac Apple Silicon (M1 … M4) | `HB_version5-@VERSION@-macos-apple-silicon.zip` |
| Mac Intel | `HB_version5-@VERSION@-macos-intel.zip` |
| Mac, si vous hésitez | `HB_version5-@VERSION@-macos-universal.zip` (fonctionne sur les deux) |
| PC Windows habituel | `HB_version5-@VERSION@-windows-amd64.zip` |
| PC Windows ARM (Snapdragon, Surface ARM) | `HB_version5-@VERSION@-windows-arm64.zip` |
| Linux PC 64 bits | `HB_version5-@VERSION@-linux-amd64.tar.gz` |
| Linux ARM 64 bits (Raspberry Pi…) | `HB_version5-@VERSION@-linux-arm64.tar.gz` |

En résumé :

- **macOS** — décompressez, glissez `HB_version5.app` dans Applications, puis
  **clic droit ▸ Ouvrir** au premier lancement (l'application n'est pas signée
  par un certificat Apple payant).
- **Windows** — décompressez le dossier entier (l'exécutable et `Resources`
  restent ensemble) et lancez `hb_go_version5.exe` ; à la première ouverture,
  SmartScreen demande « Informations complémentaires ▸ Exécuter quand même ».
- **Linux** — `tar xzf …` puis `cd HB_version5 && ./install.sh`.

Aucune installation de Python, d'Excel ou de base de données n'est nécessaire.
Une base créée par une version précédente est reprise telle quelle.

`SHA256SUMS.txt` permet de vérifier les téléchargements
(`shasum -a 256 -c SHA256SUMS.txt`).
