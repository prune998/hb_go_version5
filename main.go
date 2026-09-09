package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	shapp "go.hasen.dev/shirei/app"
)

const (
	windowTitle  = "Heures de bénévolat"
	windowWidth  = 1320
	windowHeight = 840
)

// version is stamped at build time by the Makefile
// (-ldflags "-X main.version=…"). It is shown in « À propos … ▸ Infos ».
var version = "dev"

func main() {
	png := flag.String("png", "", "rend un écran dans un fichier PNG puis quitte (mise au point)")
	screen := flag.String("screen", "home",
		"écran à rendre avec -png : "+strings.Join(previewScreens, ", ")+
			" (préfixe « "+previewDBPrefix+" » : utilise la base réelle)")
	listScreens := flag.Bool("screens", false, "liste les écrans acceptés par -screen puis quitte")
	showVersion := flag.Bool("version", false, "affiche la version puis quitte")
	flag.Parse()

	if *showVersion {
		fmt.Println(appName, version)
		return
	}
	if *listScreens {
		for _, name := range previewScreens {
			fmt.Println(name)
		}
		return
	}

	// Headless rendering: the whole interface can be checked without
	// opening a window, which is the recommended way to work on a Shirei
	// application.
	if *png != "" {
		if err := renderScreenshot(*png, *screen); err != nil {
			fmt.Fprintln(os.Stderr, "erreur :", err)
			os.Exit(1)
		}
		return
	}

	app.Setup()

	shapp.SetupWindow(windowTitle, windowWidth, windowHeight)
	if icon := bannerPath(); icon != "" {
		shapp.SetupIcon(icon)
	}
	shapp.Run(RootView)
}
