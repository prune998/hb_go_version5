package main

import (
	"sync"
	"testing"

	shirei "go.hasen.dev/shirei"
)

var fontOnce sync.Once

type tIn struct {
	mouse  shirei.Vec2
	action shirei.MouseAction
}

// runTFrame runs one frame of the real app frame function with simulated
// input, mirroring how the cocoa backend feeds shirei (one frame per mouse
// action; hover uses previous-frame geometry).
func runTFrame(in tIn) {
	shirei.GetHost().WindowSize = shirei.Vec2{1400, 750}
	shirei.GetInputState().MousePoint = in.mouse
	shirei.GetFrameInput().Mouse = in.action
	shirei.GetFrameInput().Scroll = shirei.Vec2{}
	shirei.GetFrameInput().Motion = shirei.Vec2{}
	shirei.GetFrameInput().Key = 0
	shirei.GetFrameInput().Text = ""
	shirei.RunFrameFn(frame)
}

func rectCenter(id shirei.ContainerId) shirei.Vec2 {
	r := shirei.GetScreenRectOf(id)
	return shirei.Vec2{r.Origin[0] + r.Size[0] / 2, r.Origin[1] + r.Size[1] / 2}
}

func freshAppState(t *testing.T) {
	t.Helper()
	fontOnce.Do(shirei.InitFontSubsystem)
	*app = App{}
	app.screen = scrMain
	nomBS = "ELIE"
	bens = []Benevole{
		{ID: 1, Nom: "CUIVAS", Prenom: "Jean", Donneur: " "},
		{ID: 2, Nom: "DUPONT", Prenom: "Marie", Donneur: "DDV"},
	}
}

func TestActionMenuSaisiePermanenceOpensPermAsk(t *testing.T) {
	freshAppState(t)

	off := shirei.Vec2{-100, -100}
	for i := 0; i < 3; i++ {
		runTFrame(tIn{mouse: off})
	}

	// 1. Open the Action menu: press down, release on the trigger button.
	btn := rectCenter(actionBtnId)
	if btn[0] <= 0 || btn[1] <= 0 {
		t.Fatalf("Action button has no resolved rect (id nil?): %v", btn)
	}
	runTFrame(tIn{mouse: btn, action: shirei.MouseClick})
	runTFrame(tIn{mouse: btn, action: shirei.MouseRelease})
	runTFrame(tIn{mouse: btn}) // settle: popup geometry committed for next frame

	// 2. Move onto the first item, then click it.
	it := rectCenter(actionItem1Id)
	t.Logf("button center %v, first item center %v", btn, it)
	if it[0] <= 0 || it[1] <= 0 {
		t.Fatalf("first menu item has no resolved rect — menu did not open")
	}
	runTFrame(tIn{mouse: it})
	runTFrame(tIn{mouse: it, action: shirei.MouseClick})
	runTFrame(tIn{mouse: it, action: shirei.MouseRelease})
	runTFrame(tIn{mouse: it})

	if app.screen != scrPermAsk {
		t.Fatalf("after clicking 'Saisie heures permanence', screen = %v, want %v (scrPermAsk)", app.screen, scrPermAsk)
	}
}
