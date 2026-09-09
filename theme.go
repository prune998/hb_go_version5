package main

import (
	"fmt"

	. "go.hasen.dev/shirei"
	. "go.hasen.dev/shirei/widgets"
)

// The palette: a cool, quiet office look. Colours are HSLA.
var (
	colPage       = Vec4{214, 20, 94, 1}
	colSurface    = Vec4{0, 0, 100, 1}
	colHeader     = Vec4{213, 44, 20, 1}
	colHeaderDim  = Vec4{213, 32, 30, 1}
	colAccent     = Vec4{206, 72, 42, 1}
	colAccentSoft = Vec4{206, 60, 94, 1}
	colInk        = Vec4{213, 22, 16, 1}
	colInkSoft    = Vec4{213, 10, 42, 1}
	colOnDark     = Vec4{210, 30, 97, 1}
	colOnDarkSoft = Vec4{210, 20, 78, 1}
	colLine       = Vec4{213, 18, 88, 1}
	colDanger     = Vec4{6, 68, 46, 1}
	colDangerSoft = Vec4{6, 72, 96, 1}
	colOK         = Vec4{150, 42, 34, 1}
	colZebra      = Vec4{214, 22, 98, 1}
)

func init() {
	// Buttons, inputs and check boxes all take their colour from here.
	DefaultAccent = colAccent
	ButtonAccent = Vec4{213, 22, 92, 1}
}

// ----------------------------------------------------------------- layout

// Page is the frame every screen sits in: a scrollable column with generous
// margins, so no screen has to care about the window size.
func Page(body func()) {
	Container(Attrs(Viewport, BackgroundVec(colPage)), func() {
		ScrollOnInput()
		ScrollBars()
		Container(pageAttrs(), func() {
			body()
		})
	})
}

// PageFill is the frame of the screens that fill the window instead of
// scrolling: their tables and lists take the height that is left.
func PageFill(body func()) {
	Container(Attrs(Viewport, BackgroundVec(colPage)), func() {
		Container(AttrsWith(pageAttrs(), Grow(1), Extrinsic), func() {
			body()
		})
	})
}

// pageAttrs is the shared body of Page and PageFill. The width of the
// window is turned into a MaxWidth so it cascades down the columns of the
// screen and long labels soft-wrap instead of overflowing (the settle pass
// answers the query before the frame is presented).
func pageAttrs() AttrSet {
	attrs := Attrs(Expand, Clip, Pad4(20, 26, 26, 26), Gap(16))
	if width := GetResolvedSize()[0]; width > 200 {
		attrs = AttrsWith(attrs, MaxWidth(width))
	}
	return attrs
}

// Card is the white panel used for every block of the interface.
func Card(body func()) {
	Container(Attrs(Expand, Clip, Pad(18), Gap(14), BackgroundVec(colSurface),
		Corners(10), BorderWidth(1), BorderColorVec(colLine)), func() {
		body()
	})
}

// CardHeader is the title line of a card, with an optional hint below.
func CardHeader(icon IconGlyph, title, hint string) {
	Container(Attrs(Row, Expand, CrossMid, Gap(10)), func() {
		if icon.Rune != 0 {
			Icon(icon, FontSize(18), TextColorVec(colAccent))
		}
		Label(title, FontSize(16), FontWeight(WeightBold), TextColorVec(colInk))
	})
	if hint != "" {
		Label(hint, FontSize(12), TextColorVec(colInkSoft))
	}
}

// SectionTitle labels a group inside a card.
func SectionTitle(title string) {
	Label(title, FontSize(11), FontWeight(WeightBold), TextColorVec(colInkSoft))
}

// Divider draws a hairline between two blocks.
func Divider() {
	Element(Attrs(Expand, FixHeight(1), BackgroundVec(colLine)))
}

// StepBar shows where the user stands in a workflow. Every action of the
// menu follows the same steps, so they are always displayed the same way.
func StepBar(steps []string, current int) {
	Container(Attrs(Row, Expand, CrossMid, Gap(8), Wrap), func() {
		for i, step := range steps {
			i, step := i, step
			if i > 0 {
				Icon(SymRight, FontSize(12), TextColorVec(colInkSoft))
			}
			done := i < current
			active := i == current

			bg, fg := colSurface, colInkSoft
			switch {
			case active:
				bg, fg = colAccent, colOnDark
			case done:
				bg, fg = colAccentSoft, colAccent
			}

			Container(Attrs(Row, CrossMid, Gap(8), Pad2(6, 12), Corners(20),
				BackgroundVec(bg), BorderWidth(1), BorderColorVec(colLine)), func() {
				Label(fmt.Sprintf("%d", i+1), FontSize(11), FontWeight(WeightBold), TextColorVec(fg))
				Label(step, FontSize(12), FontWeight(WeightBold), TextColorVec(fg))
			})
		}
	})
}

// WorkflowHeader is the top of every action screen: a title, the step bar
// and a "back to the home page" control.
func WorkflowHeader(icon IconGlyph, title, subtitle string, steps []string, current int) {
	Container(Attrs(Expand, Clip, Pad(18), Gap(14), BackgroundVec(colSurface),
		Corners(10), BorderWidth(1), BorderColorVec(colLine)), func() {
		Container(Attrs(Row, Expand, CrossMid, Gap(14)), func() {
			Container(Attrs(Grow(1), Clip, Gap(3)), func() {
				Container(Attrs(Row, CrossMid, Gap(10)), func() {
					Icon(icon, FontSize(20), TextColorVec(colAccent))
					Label(title, FontSize(19), FontWeight(WeightBold), TextColorVec(colInk))
				})
				if subtitle != "" {
					Label(subtitle, FontSize(12), TextColorVec(colInkSoft))
				}
			})
			if GhostButton(SymHome, "Accueil") {
				app.GoHome()
			}
		})
		Divider()
		StepBar(steps, current)
	})
}

// ---------------------------------------------------------------- controls

// PrimaryButton is the button that moves a workflow forward.
func PrimaryButton(icon IconGlyph, label string) bool {
	return ButtonExt(label, ButtonAttrs{Icon: icon, Accent: colAccent, TextSize: 13}, DefaultButtonLook())
}

// PrimaryButtonEnabled is PrimaryButton, greyed out when the step is not
// complete yet.
func PrimaryButtonEnabled(icon IconGlyph, label string, enabled bool) bool {
	return ButtonExt(label, ButtonAttrs{Icon: icon, Accent: colAccent, TextSize: 13, Disabled: !enabled}, DefaultButtonLook()) && enabled
}

// DangerButton is used to clear a selection or to accept a destructive
// operation.
func DangerButton(icon IconGlyph, label string) bool {
	return ButtonExt(label, ButtonAttrs{Icon: icon, Accent: colDanger, TextSize: 13}, DefaultButtonLook())
}

// GhostButton is a discreet secondary action.
func GhostButton(icon IconGlyph, label string) bool {
	return ButtonExt(label, ButtonAttrs{Icon: icon, TextSize: 12}, DefaultCtrlButtonLook())
}

// Field lays out a labelled control, all of them aligned on the same grid.
func Field(label, hint string, body func()) {
	Container(Attrs(Row, Expand, Gap(14), CrossAlign(AlignStart)), func() {
		Container(Attrs(FixWidth(210), Gap(2), Pad4(6, 0, 0, 0)), func() {
			Label(label, FontSize(13), FontWeight(WeightBold), TextColorVec(colInk))
			if hint != "" {
				Label(hint, FontSize(11), TextColorVec(colInkSoft))
			}
		})
		Container(Attrs(Grow(1), Clip, Gap(6)), func() {
			body()
		})
	})
}

// DateField is the "jj/mm/aaaa" input used by every screen.
func DateField(buf *string) {
	attrs := DefaultTextInputAttrs()
	attrs.FontSize = 14
	attrs.MinWidth = 130
	attrs.FixedWidth = true
	attrs.NoAutoFocus = true
	attrs.Placeholder = "jj/mm/aaaa"
	TextInputExt(buf, attrs)
}

// HoursField is the small input holding a whole number of hours.
func HoursField(buf *string) {
	attrs := DefaultTextInputAttrs()
	attrs.FontSize = 14
	attrs.MinWidth = 74
	attrs.FixedWidth = true
	attrs.NoAutoFocus = true
	TextInputExt(buf, attrs)
}

// TextField is a free text input of the given width.
func TextField(buf *string, width float32, placeholder string) {
	attrs := DefaultTextInputAttrs()
	attrs.FontSize = 13
	attrs.MinWidth = width
	attrs.FixedWidth = true
	attrs.NoAutoFocus = true
	attrs.Placeholder = placeholder
	TextInputExt(buf, attrs)
}

// SearchField is the volunteer filter shown above every list.
func SearchField(buf *string) {
	Container(Attrs(Row, Expand, CrossMid, Gap(6)), func() {
		Icon(SymSearch, FontSize(13), TextColorVec(colInkSoft))
		attrs := DefaultTextInputAttrs()
		attrs.FontSize = 13
		attrs.NoAutoFocus = true
		attrs.Placeholder = "Rechercher un bénévole"
		TextInputExt(buf, attrs)
	})
}

// ----------------------------------------------------------------- pieces

// Banner is the inline message block: validation errors and warnings appear
// where the user is working instead of in a dialog to acknowledge.
func Banner(icon IconGlyph, message string, bg, fg Vec4) {
	if message == "" {
		return
	}
	Container(Attrs(Row, Expand, Clip, CrossAlign(AlignStart), Gap(10), Pad2(10, 12),
		BackgroundVec(bg), Corners(8)), func() {
		wrap := textWrapWidth(60)
		Icon(icon, FontSize(14), TextColorVec(fg))
		Container(Attrs(Grow(1), Clip, MaxWidth(wrap)), func() {
			Label(message, FontSize(12), TextColorVec(fg))
		})
	})
}

// ErrorBanner reports a rejected entry.
func ErrorBanner(message string) { Banner(SymWarn, message, colDangerSoft, colDanger) }

// InfoBanner explains what the current step expects.
func InfoBanner(message string) { Banner(SymInfo, message, colAccentSoft, colAccent) }

// textWrapWidth is the width left for a text cell in the current row, used
// as a soft-wrap limit: the cell then keeps its content height (unlike an
// Extrinsic cell, which would collapse a row that has no fixed height).
// Zero means "no limit", which is what a first, not yet measured frame gets.
func textWrapWidth(reserved float32) float32 {
	width := GetAvailableSize()[0] - reserved
	if width < 120 {
		return 0
	}
	return width
}

// Tag is a small coloured label, e.g. the "DDV" marker of a volunteer.
func Tag(text string, bg, fg Vec4) {
	Container(Attrs(Pad2(1, 6), Corners(4), BackgroundVec(bg)), func() {
		Label(text, FontSize(10), FontWeight(WeightBold), TextColorVec(fg))
	})
}

// StatTile is one figure of the dashboard or of a summary.
func StatTile(label, value, hint string) {
	Container(Attrs(Grow(1), MinWidth(150), Gap(4), Pad(14), Corners(8),
		BackgroundVec(colZebra), BorderWidth(1), BorderColorVec(colLine)), func() {
		Label(label, FontSize(11), FontWeight(WeightBold), TextColorVec(colInkSoft))
		Label(value, FontSize(24), FontWeight(WeightBold), TextColorVec(colInk))
		if hint != "" {
			Label(hint, FontSize(11), TextColorVec(colInkSoft))
		}
	})
}

// HoursLabel prints a number of hours: zero stays discreet, a negative
// value (a correction) is red, as in the printed journal.
func HoursLabel(hours int) {
	color := colInk
	switch {
	case hours < 0:
		color = colDanger
	case hours == 0:
		color = Vec4{213, 8, 72, 1}
	}
	Label(fmt.Sprintf("%d", hours), FontSize(12), TextColorVec(color), Fonts(Monospace...))
}

// EmptyState fills a list or a table that has nothing to show yet.
func EmptyState(icon IconGlyph, message string) {
	Container(Attrs(Expand, Grow(1), Center, Gap(8), Pad(24)), func() {
		Icon(icon, FontSize(26), TextColorVec(Vec4{213, 10, 78, 1}))
		Label(message, FontSize(12), TextColorVec(colInkSoft))
	})
}

// Pane is a titled sub-panel of a screen; the two panes of the entry
// screens (volunteers, pending lines) are built with it.
func Pane(title string, trailing func(), body func()) {
	Container(Attrs(Grow(1), Expand, Extrinsic, Clip, Gap(0), BackgroundVec(colSurface),
		Corners(10), BorderWidth(1), BorderColorVec(colLine)), func() {
		Container(Attrs(Row, Expand, CrossMid, Gap(10), Pad2(10, 14), BackgroundVec(colZebra)), func() {
			Container(Attrs(Grow(1), Clip), func() {
				Label(title, FontSize(13), FontWeight(WeightBold), TextColorVec(colInk))
			})
			if trailing != nil {
				trailing()
			}
		})
		Divider()
		Container(Attrs(Grow(1), Expand, Extrinsic, Clip), func() {
			body()
		})
	})
}
