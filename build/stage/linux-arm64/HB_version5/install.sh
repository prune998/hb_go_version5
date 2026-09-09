#!/bin/sh
# Installe « Heures de bénévolat » pour l'utilisateur courant.
#
#   ./install.sh              installe dans ~/.local
#   PREFIX=/opt ./install.sh  installe ailleurs (droits d'écriture requis)
#
# Désinstallation : ./install.sh --uninstall

set -eu

BIN=hb_go_version5
PREFIX=${PREFIX:-$HOME/.local}
LIBDIR=$PREFIX/lib/$BIN
BINDIR=$PREFIX/bin
DESKTOPDIR=$PREFIX/share/applications
SOURCE=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

if [ "${1:-}" = "--uninstall" ]; then
	rm -rf "$LIBDIR"
	rm -f "$BINDIR/$BIN" "$DESKTOPDIR/$BIN.desktop"
	echo "Désinstallation terminée."
	exit 0
fi

echo "Installation dans $PREFIX …"
mkdir -p "$LIBDIR" "$BINDIR" "$DESKTOPDIR"

# Le programme cherche ses ressources dans le dossier « Resources » placé à
# côté de l'exécutable : les deux restent donc ensemble dans $LIBDIR.
install -m 0755 "$SOURCE/$BIN" "$LIBDIR/$BIN"
rm -rf "$LIBDIR/Resources"
cp -R "$SOURCE/Resources" "$LIBDIR/Resources"

ln -sf "$LIBDIR/$BIN" "$BINDIR/$BIN"

sed -e "s|@EXEC@|$LIBDIR/$BIN|" \
    -e "s|@ICON@|$LIBDIR/Resources/Image_logiciel.png|" \
    "$SOURCE/$BIN.desktop" > "$DESKTOPDIR/$BIN.desktop"

if command -v update-desktop-database >/dev/null 2>&1; then
	update-desktop-database "$DESKTOPDIR" >/dev/null 2>&1 || true
fi

echo "Terminé."
echo "  Lancement          : $BIN   (ou depuis le menu des applications)"
echo "  Dossier de travail : le répertoire courant (base et fichiers Excel)"
case ":$PATH:" in
	*":$BINDIR:"*) ;;
	*) echo "  Note : ajoutez $BINDIR à votre PATH pour lancer « $BIN » directement." ;;
esac
