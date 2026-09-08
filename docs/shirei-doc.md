# Building Applications with Shirei

This tutorial explains how to build a small utility-style GUI application with
Shirei. It is meant to be readable by humans, while still being practical enough
for AI coding sessions to use as a guide.

Good example programs to study, referenced throughout:

- `examples/dir_weight` — disk usage scanner with background work, tabs, filtering,
  progress, and a large virtual list.
- `examples/see_pprof` — pprof visualizer with a sidebar, dense sortable
  tables, file watching, a custom flame-graph canvas, and snapshot tests.
- `examples/process_monitor` — live process monitor with platform data
  collectors, stable app-owned objects, flat/table mode, tree mode, selection,
  and CPU/RAM history charts.

For sound (mixing, voices, streaming, headless audio verification), see the
companion [audio-tutorial.md](audio-tutorial.md); `examples/piano` is its
reference program.

Specialized feature write-ups live next to this file (for example
[drag-drop.md](drag-drop.md) for item drag-and-drop,
[virtual-list.md](virtual-list.md) for VirtualList and `Measure`,
[layout-tutorial.md](layout-tutorial.md) for a progressive multi-panel shell,
and [android.md](android.md) / [ios.md](ios.md) for device runners). Touch and
multi-touch contacts are covered under Interaction (§6). This file stays a
general overview.

This tutorial was written by Claude Fable 5, and edited by the framework's
original human author and technical director.

An initial version was written with Fugu (SakanaAI).

---

# Part I — The model

## 1. UI as a function of state

Shirei uses a **plain-data, declarative** application model. Your program
describes what the whole UI should look like right now whenever Shirei
evaluates a requested update:

```go
func RootView() {
    Header()
    Toolbar()
    MainPanel()
    DetailPanel()
}
```

You do not create a button object once and attach callbacks to it. Instead, you
write the button where it belongs in the current interface:

```go
if Button(SymRefresh, "Refresh") {
    refreshData()
}
```

If the button was clicked during the update, `Button` returns `true`. If not,
it returns `false`. The UI is just a function of the current application state.

Shirei calls a requested update a **frame**. View code builds a transient
container tree for that update, then Shirei performs layout and rendering in
deferred passes.

A typical Shirei program therefore has two layers:

1. **Application state** — your own structs, slices, maps, selected item,
   filters, sort mode, etc.
2. **View functions** — Shirei layout code that builds the current tree of
   containers whenever an update is requested (§4).

```go
type AppState struct {
    files    []*File
    selected *File
    showGrid bool
    useGrouping bool
}

var appData = &AppState{showGrid: true}
```

This is the central idea: **keep durable data in your own structs; use Shirei
to describe and interact with it during each requested update.**

### Data is just data

Most GUI frameworks have a special kind of variable for anything that
influences the UI — React's "state", observables, reactive properties. You
cannot simply mutate an ordinary variable; you have to shape your data the
way the framework approves of, so it can notice changes and schedule
updates.

Shirei deliberately has none of that. Application state is ordinary Go data
— structs, slices, maps — mutated however you like, because plain data is
much easier to manage. On each requested frame, the whole UI is rebuilt from
the application data, so the next frame reflects whatever the data says. Any
variable can drive the UI.

The trade is explicitness about *when frames run*. Shirei does not render in
a continuous loop: a frame runs when input arrives, when the previous
frame's output changed (so built-in animations keep flowing), or when one is
explicitly requested. An idle app renders nothing and costs nothing. And
because your data is just data, Shirei cannot see you change it — when a
change happens outside the input loop, you must say so:

```go
RequestNextFrame()
```

You need it exactly when change originates outside user input:

- a background goroutine published new data (§11) — without the request, the
  UI would update only when the user next moves the mouse;
- you are driving a continuous effect yourself — a blinking caret, an
  elapsed-time clock — where each frame requests the next.

You do not need it for ordinary interactions: input produces a frame by
itself, and Shirei requests follow-ups for its own animations automatically.
Geometry that depends on last frame's layout settles inside the same
presented frame for the common case (§7) — you do not call
`RequestNextFrame` just because a size query started at zero.

That is the deal: the framework never tells you how to shape your data; in
exchange, you tell it when the data changed behind its back.

## 2. A minimal app

Most Shirei applications start with this shape:

```go
package main

import (
    app "go.hasen.dev/shirei/app"

    . "go.hasen.dev/shirei"
    . "go.hasen.dev/shirei/widgets"
)

func main() {
    app.SetupWindow("My App", 1000, 700)
    app.Run(RootView)
}

func RootView() {
    Container(Attrs(Viewport, Background(220, 10, 97, 1)), func() {
        Header()
        MainContent()
    })
}
```

Window placement or size limits can be configured with the `ext/window` extension:

```go
app.SetupWindow("My App", 1000, 700)
window.Center()          // or: window.Position(120, 80)
window.SetMinSize(400, 300)
app.Run(RootView)
```

Placement and min size are best-effort: macOS, Windows, and X11 honor placement; Wayland leaves
top-level placement to the compositor; mobile is always full-screen. On macOS
the window is centered by default.

The dot imports are the house style; they make UI code much easier to read:

```go
Container(Attrs(Row, Expand, CrossMid, Gap(10), Pad(12)), func() {
    Label("Hello")
    Button(NoIcon, "Run")
})
```

## 3. Verify as you go: headless rendering

Shirei renders with its own software rasterizer, so a frame can be rendered
**without opening a window**. For AI agents especially, it's recommended to
build this into the app from the start — it gives a very fast feedback loop for
everything else in this tutorial:

```go
func main() {
    // `myapp --png out.png` renders one settled frame and exits
    if len(os.Args) >= 3 && os.Args[1] == "--png" {
        if err := RenderToPNG(os.Args[2], 1000, 700, RootView); err != nil {
            fmt.Println("render failed:", err)
        }
        return
    }

    app.SetupWindow("My App", 1000, 700)
    app.Run(RootView)
}
```

`RenderToPNG` (and `RenderToImage`, which returns the `*image.RGBA` instead)
runs several frames until the layout settles, from a neutral input state,
with animations disabled — so the output is deterministic. This gives you:

- **Instant visual checks** during development without clicking around.
- **Snapshot tests**: render into an image and compare against a committed
  golden PNG. A mismatch writes an `.actual.png` you can eyeball; an
  environment flag regenerates goldens. See `examples/see_pprof/main_test.go`
  for a complete worked example (including how to make the app's data
  deterministic for the test).
- **AI-friendly verification**: a coding session can render your app and
  *look at it* without a human driving a window.

For testing interactions (clicks, typing, scrolling) headlessly, drive frames
directly: set `WindowSize`, fill `InputState`/`FrameInput`, and call
`RunFrameFn`. The tests in `widgets/semantic_test.go` are the reference
pattern.

One habit worth forming: when a screen reaches a state you like, snapshot it.
The test then guards every future refactor for free.

---

# Part II — Building blocks

## 4. Containers

For each requested frame, your view functions build a **tree of containers**,
and `Container` is how you add to it. A call like this:

```go
Container(Attrs(Gap(8)), func() {
    Label("First")
    Label("Second")
})
```

does three things:

- it opens a new container as a child of the *current* container;
- `Attrs(...)` builds that container's **attribute set** — its layout and
  appearance (here, an 8px gap between children; more attributes below);
- the builder `func()` runs with the new container as the current one, so the
  `Label` calls inside it become *its* children. When the builder returns,
  "current" reverts to the parent.

Nesting `Container` calls therefore nests containers — that is how the whole tree
takes shape, one builder inside another. At the root, `app.Run(RootView)` calls
your top-level view with the window as the current container, and everything
grows from there.

Note that `Label` is not a special node type. Widgets — `Label`, `Button`,
`TextInput`, and the rest — are ordinary functions that build containers with
`Container`/`Element` themselves, so a `Label` inside a builder adds container
children just as a nested `Container` would. Coming from React, this is the thing
to unlearn: there is no separate "component" or "element" kind, and no virtual
DOM — **there are only containers**, built by plain function calls for the
current frame.

Containers stack their children vertically by default; `Row` makes them
horizontal:

```go
Container(Attrs(Row, Gap(8)), func() {
    Label("Left")
    Label("Right")
})
```

`Element` is a container without a builder (a colored box, a spacer).

```go
Element(Attrs(Expand, FixHeight(1), Background(0, 0, 0, 1))) // 1px separator
```

`ModAttrs(...)` modifies the *current* container's attributes from inside its
builder — useful for conditional styling (hover highlights) and for *clearing*
cascaded values (see cascade timing below). Call it before adding children.

```go
for _, item := range list {
    Container(Attrs(Background(200, 50, 50, 1), Pad(10)), func() {
        // highlight the current row if it's selected
        if selectedItemId == item.Id {
            ModAttrs(Background(200, 50, 70, 1)) // sets a different background color
        }

        // add children *after* ModAttrs
        Label(item.Name, TextColor(0, 0, 100, 0.7))
    })
}
```


### How sizing works (the two-pass mental model)

1. **Bottom-up**: children are laid out, then the parent sizes itself to
   contain them. By default, a container is exactly as big as its content,
   unless MinWidth or MinHeight are set, then a container could be bigger
   than its content.
2. **Top-down**: leftover space is distributed. `Grow(n)` takes remaining
   space along the parent's main axis (proportionally, if several children
   grow); `Expand` stretches across the cross axis; `MinSize`/`MaxSize`
   constrain the result.

The one attribute that changes the rules: **`Extrinsic`** (`ExtrinsicSize`,
also bundled into `Viewport`) makes a container ignore its content entirely
and take only what **parent constraints** give it. That is what makes
budgeted panels and scroll regions possible: without it, content can
**inflate** the measured size of a `Grow`/`Expand` panel instead of
overflowing or clipping inside a fixed budget.

`Extrinsic` is **all-or-nothing across axes**. The container no longer
sizes itself from content on *any* axis. So it needs a real external size
on each axis that matters:

- If a flex parent assigns width and height (e.g. `Grow(1)` in a full-height
  row next to a sidebar), Extrinsic is safe for that **pane**.
- If a container is an ordinary content row in a column (header, find bar,
  status strip) and you set Extrinsic **without** also giving it height
  (`FixHeight`, `MinHeight` with a non-extrinsic height path, or a parent
  height slot), its **height collapses** — there is nothing content can
  contribute and nothing external assigned.

**Pick the tool for the job:**

| Intent | Prefer |
|--------|--------|
| Pane / scroll body: size from outside on both axes | `Viewport`, or `Grow`+`Expand`+`Extrinsic`+`Clip` |
| Content row (toolbar, find bar, title strip): height from content; do not widen the pane | `Expand`+`Clip` (rely on an **extrinsic parent** for width); optional `MinHeight` |
| Cap only one axis | `MaxWidth` / `MaxHeight` / `FixHeight` — not full Extrinsic |
| Clip painting without changing sizing rules | `Clip` alone (does **not** stop content from measuring larger) |

A common full-window layout (and a nested “detail” column that must not
grow from long labels or text fields):

```go
func RootView() {
    Container(Attrs(Viewport, Background(220, 10, 97, 1)), func() {
        Header()                                // content-sized height
        Toolbar()                               // content-sized height
        Container(Attrs(Grow(1), Expand, Clip), func() {
            MainList()                          // takes the remaining space
        })
        StatusBar()                             // content-sized height
    })
}

// Sidebar | detail: detail width from flex, not from the longest line.
Container(Attrs(Row, Grow(1), Expand, Clip), func() {
    Sidebar() // FixWidth(...)
    Container(Attrs(Grow(1), Expand, Extrinsic, Clip), func() {
        DetailHeader()  // Expand, Clip — height from content
        FindBar()       // Expand, Clip, MinHeight(...) — not Extrinsic alone
        Container(Attrs(Viewport), func() {
            // scroll body: Extrinsic + Grow from Viewport
            DiffOrList()
        })
    })
})
```

Inside the extrinsic detail column, tool strips stay **content-height**;
only the fill region uses `Viewport`. Do not put bare `Extrinsic` on every
child “to prevent inflation” — that is how zero-height rows appear.

### Cascade: MaxSize (and friends)

Some attributes **cascade** from parent to child when the child leaves them
unset: cross-axis `MaxSize`, text style (§5), and a few behavior flags
(`NoAnimate`, `ClickThrough`). Cascade is not CSS-style deep inheritance of
every property — it is a one-step copy from the *immediate parent* at container
open. The rules matter for layout, so read them carefully.

#### Cross-axis MaxSize only

A parent only cascades max on its **cross** axis (the axis perpendicular to
how it lays out children):

| Parent layout | Cascades | Does **not** cascade |
|---|---|---|
| Column (default) | `MaxWidth` → children's max width | `MaxHeight` |
| Row | `MaxHeight` → children's max height | `MaxWidth` |

Parent padding is peeled off first, so the child receives the content-box
budget (parent max − pad on that axis).

Main-axis max stays on the parent only. That is deliberate: a wrapping row
with `MaxWidth(400)` should wrap its own children, not force every button
inside to a 400px width.

#### One step at a time — orientation can flip

Cascade always looks at the *immediate* parent. Nested layouts therefore
change which axis continues:

```go
// Column with MaxWidth: children that leave width unset inherit that max.
Container(Attrs(MaxWidth(300), Pad(12), Gap(8)), func() {
    Label("This paragraph soft-wraps at ~276px") // 300 − 12 − 12 pad
    // (Text reads current.MaxSize[0] for wrap width; see §5.)

    // This row is a *child of the column*, so it inherits MaxWidth.
    // But a row's cross axis is height, not width — so its own children
    // do NOT inherit that width. They only inherit MaxHeight if this row
    // sets one.
    Container(Attrs(Row, Gap(8)), func() {
        Button(NoIcon, "A") // no cascaded MaxWidth from the column above
        Button(NoIcon, "B")
        // If the row itself set MaxHeight(40), A and B would get max height 40
        // (minus the row's vertical pad). They still would not get max width
        // from the outer column.
    })
})
```

Picture the tree:

```text
Column  MaxWidth=300
├── Label          → max width cascaded (column's cross axis)
└── Row            → max width cascaded onto the row itself
    ├── Button A   → row's cross axis is height → no max width from above
    └── Button B   → same
```

So a max width set high in a column tree reaches every nested *column* child
along that path, but the moment you open a row, further descendants stop
receiving width cascade and may start receiving height cascade instead.

#### Opting out of cascade (`UnsetMaxCross`, `YesAnimate`)

Cascade runs when a container opens: if the child's field is still zero (and
not explicitly opted out), the parent's value is copied in.

**`UnsetMaxCross`** and **`YesAnimate` / `NoAnimate` / `AnimateOnly`** set a
flag so open-time cascade does not overwrite them. Both work in `Attrs(...)`:

```go
// Cross-axis max: do not inherit the parent's MaxWidth (column) / MaxHeight (row).
Container(Attrs(UnsetMaxCross), func() {
    // wide content may overflow this box; children do not see the grandparent max
})

// Animation: re-enable easing under a Viewport / NoAnimate parent.
Container(Attrs(Viewport), func() {
    Container(Attrs(YesAnimate, /* ... */), func() { /* eases again */ })
})
```

`ModAttrs(UnsetMaxCross)` after open still works if you prefer to clear in the
builder. Clearing stops further propagation **below that node** — cascade is
only parent → child.

### The attribute cheat sheet

`Attrs(...)` composes attribute functions into an `AttrSet` value; `AttrsWith(base, ...)`
extends an existing one. The ones you will use constantly:

Structure and sizing:

| attr | meaning |
|---|---|
| `Row` | horizontal main axis |
| `Gap(n)` | space between children |
| `Pad(n)` / `Pad2(v,h)` / `Pad4(t,r,b,l)` | padding |
| `Spacing(n)` | shorthand: `Gap(n)` + `Pad(n)` |
| `Grow(n)` | take remaining main-axis space |
| `Expand` | fill the cross axis |
| `FixWidth/FixHeight/FixSize` | exact size |
| `MinWidth/MinHeight/MinSize`, `MaxWidth/MaxHeight` | constraints (cross-axis max cascades; see above) |
| `UnsetMaxCross` | clear cascaded cross-axis max (`Attrs` or `ModAttrs`) |
| `Clip` | clip overflowing content (paint only; does not by itself stop measuring larger) |
| `Wrap` | wrap children into lines |
| `Extrinsic` | size from constraints only — content neither inflates nor defines size (all axes); see sizing above |
| `Viewport` | bundle: `Extrinsic`+`Grow(1)`+`Expand`+`Clip`+`NoAnimate` — fill/scroll panel, not every chrome row |
| `Center` / `CrossMid` | center children (both axes / cross axis) |
| `CrossAlign(...)`/`MainAlign(...)`/`SelfAlign(...)` | cross/main/self alignment |
| `Float(x, y)` | position by hand within the parent (skips flow layout) |
| `InFront` / `Behind` / `Z(n)` | z-ordering |

Appearance:

| attr | meaning |
|---|---|
| `Background(h, s, l, a)` | background color, HSLA |
| `Grad(dh, ds, dl, da)` | vertical gradient as a *delta* from the background color |
| `Corners(r)` / `Corners4(...)` | corner rounding |
| `BorderWidth(w)`, `BorderColor(h, s, l, a)` | border width and color |
| `BoxShadow(blur)` | drop shadow |
| `Trans(a)` | transparency applied to the whole subtree |

Text style (on containers; full story in §5):

| attr | meaning |
|---|---|
| `AmendTextStyle(mods...)` | parent text style + mods → this container's style |
| `SetTextStyle(base, mods...)` | fresh base + mods (does not inherit parent style) |

Behavior:

| attr | meaning |
|---|---|
| `NoAnimate` / `YesAnimate` | opt out of / back into implicit animation (`NoAnimate` cascades; re-enable with `ModAttrs(YesAnimate)`) |
| `ClickThrough` | exempt from hit-testing (tooltips, overlays); cascades |
| `NoClickThrough` | opt back into hit-testing under a `ClickThrough` parent |
| `Focusable` | participates in tab-cycling focus |

From `widgets`, two layout helpers you will use in every toolbar:

```go
Filler(1) // flexible empty space (an Element with Grow)
Spacer(8) // fixed space along the current layout direction
```

```go
Container(Attrs(Row, CrossMid, Gap(10), Pad(12)), func() {
    Label("My App")
    Filler(1)                    // pushes the button to the right edge
    Button(NoIcon, "Refresh")
})
```

## 5. Colors and text styling

Colors are HSLA `Vec4`s: hue 0–360, saturation 0–100, lightness 0–100,
alpha 0–1.

```go
Background(220, 10, 97, 1) // very light cool gray
Background(220, 25, 18, 1) // dark blue-gray
TextColor(0, 0, 100, 1)  // white text
TextColor(0, 0, 15, 1)   // almost black text
```

### Text style lives on containers

Every container carries a fully resolved **text style** (font size, weight,
families, color, and related fields). The frame root starts each frame at
`DefaultTextStyle()`. When a child leaves its style unset, it **inherits the
parent's whole style** (cloned) — wholesale, not field-by-field. Same cascade
timing as MaxSize (§4): parent → child at open; only the immediate parent.

Set style on a container so a whole panel shares defaults:

```go
// Inherit whatever the parent uses, then bump size and color for this subtree.
Container(Attrs(AmendTextStyle(FontSize(14), TextColor(0, 0, 20, 1))), func() {
    Label("Body copy in the panel default")
    Label("Also body — no need to repeat FontSize on every label")

    // Nested amend: starts from *this* panel's style, not the root default.
    Container(Attrs(AmendTextStyle(Fonts(Monospace...))), func() {
        Label("mono body at 14px, same color")
    })
})

// Replace rather than amend: ignore parent style entirely.
Container(Attrs(SetTextStyle(DefaultTextStyle(), FontSize(11), TextColor(0, 0, 45, 1))), func() {
    Label("Caption chrome with an explicit base")
})
```

- **`AmendTextStyle(mods...)`** — copy current parent style, apply mods, store
  the full result on this container (so its children cascade from here).
- **`SetTextStyle(base, mods...)`** — same, but start from an explicit `base`
  (often `DefaultTextStyle()`) instead of the parent. Use when a subtree must
  not pick up a themed ancestor.

There is no separate "unset text style" flag: to break out of a themed parent,
`SetTextStyle` with the base you want.

### Labels and call-local style

`Label` is the everyday API. Optional mods amend the **current container's**
text style for that leaf only — siblings and the container itself are
unchanged:

```go
Label("Heading", FontSize(18), FontWeight(WeightBold), TextColor(220, 40, 25, 1))
Label("caption", FontSize(10), FontStyle(StyleItalic), TextColor(0, 0, 45, 1))
```

Under the hood, `Label(s, mods...)` is `Text(s, TextStyle(mods...))`.
`TextStyle(mods...)` returns current style + mods as a fully resolved value
for `Text`'s second argument; it does not write back onto the container.

### Soft wrap width

Text soft-wraps to the **current container's content-box max width**:
`MaxSize[0]` minus horizontal padding (including a max cascaded from an
ancestor, §4). Zero max means unconstrained (no soft wrap). Put `MaxWidth`
on a panel (or rely on cascade into a column child); do not expect a
separate per-text max-width attribute.

```go
Container(Attrs(MaxWidth(280), Pad(10)), func() {
    // Wraps at 260px content width (280 − 10 − 10).
    Label("Long copy wraps inside this panel without extra width args on Label.")
})
```

### Inline spans

For mixed styles inside one string, use `Text` with `Span` ranges (rune
indices, half-open). Spans resolve against that call's paragraph base:

```go
Text("The word orange is colored.", TextStyle(),
    Span(9, 15, TextColor(30, 90, 50, 1)), // "orange"
)
```

`demos/style-spans` is the worked gallery: panel `AmendTextStyle`, then one
`Text` + `Span`s per card.

`FontSize` sets the size; `Fonts(...)` selects font families (system fonts are
discovered automatically, with per-rune fallback — CJK and bidi text work out
of the box).

Shirei also bundles two icon fonts — **Typicons** and **Microns** — as
`IconGlyph` values in `widgets` (`TypArrowUpThick`, `TypFolderOpen`,
`SymRefresh`, …; see `widgets/typicons.go` and `widgets/microns.go`).
Each value pairs a font family with a codepoint so PUA icons from different
fonts cannot be mixed. Render one with `Icon(TypFolderOpen)`, or pass it to
widgets that take an `IconGlyph`: `Button(SymRefresh, "Refresh")`, with
`NoIcon` meaning no icon. For a custom icon font, register it with
`UseFontBytes` and define your own `IconGlyph{Font: "…", Rune: …}` values.
The bundled fonts register themselves when the `widgets` package is
imported, so they work everywhere — windowed and headless (`--png`,
snapshot tests) alike. To find an icon, run `examples/icons`: a filterable
gallery of the full set, each glyph next to its name.

A typical utility-app palette: dark header, light toolbar, medium-light table
header, alternating light rows, light detail panel.

## 6. Interaction

Shirei input is **data, not events**: each frame the backend publishes the
current input state, and your code queries it inline. `InputState` holds
cumulative state (mouse position, held keys, modifiers); `FrameInput` holds
what happened *this frame* (a click, a key press, typed text, scroll).

### Hover, click, press

```go
Container(Attrs(Row, Pad(8), Background(0, 0, 95, 1)), func() {
    if IsHovered() {
        ModAttrs(Background(0, 0, 90, 1))    // hover highlight
    }
    if PressAction() {
        appData.selected = item      // acts on full press (down then up inside)
    }
    Label(item.Name)
})
```

Three related queries, with different meanings:

- `IsClicked()` — the mouse button went *down* over this container this
  frame. Right for selection and other non-destructive acts.
- `PressAction()` — a full press *gesture* completed: down and up both inside
  this container. Right for buttons and anything destructive (the user can
  still bail by dragging off before releasing).
- `IsDoubleClicked()` — this frame's click is the second (or later) of a
  click streak (`FrameInput.ClickCount`, detected by core from click timing
  and position). Note the first click of the pair fires `IsClicked` on its
  own frame — so the natural pattern "click selects, double-click acts"
  needs no special handling.

`IsHoveredDirectly()` is `IsHovered` minus children — true only when the
cursor is on this container itself, e.g. to treat a click on a panel's empty
background differently from a click on its content.

### Dragging

`PressAction()` marks the container *active* between mouse-down and mouse-up;
`IsActive()` queries it. With `FrameInput.Motion` (per-frame mouse movement),
that is the whole dragging story. A pane splitter, complete:

```go
Container(Attrs(FixHeight(6), Expand, Background(0, 0, 80, 1)), func() {
    if IsHovered() {
        ModAttrs(Background(210, 60, 60, 1))
    }
    PressAction()
    if IsActive() {
        splitRatio += FrameInput.Motion[1] / totalHeight
    }
})
```

For **moving items between drop zones** (cards between columns, balls into
buckets), use the drag-and-drop helpers in `widgets` instead — see
[drag-drop.md](drag-drop.md). That API carries typed payloads;
`PressAction` does not.

### Touch and multi-touch

On mobile (and any backend with a finger), **one finger is also mouse**:
backends fill the usual pointer path so taps, drags, and fling scroll work
with the same `PressAction` / `ScrollOnInput` code as on desktop. In parallel,
every active contact is published as multi-touch **data** —
`InputState.Touches` (up to `MaxTouches`), with began/ended ids on
`FrameInput`, and hit queries `IsTouched` / `IsTouchedDirectly` /
`TouchingIds` / `TouchById` (same timing idea as hover). Prefer
`IsTouched` when several fingers matter; while `InputState.MouseFromTouch` is
set, ignore the synthetic mouse for hold-style logic so a delayed mouse-up
cannot re-press after lift (see `examples/piano`). Shirei does **not** yet
ship built-in multi-finger *gestures* (pinch, two-finger pan, rotate) —
those are ordinary app code over the contact table if you need them.

### Scrolling

Inside any clipped container, `ScrollOnInput()` applies wheel/trackpad input
to the container's scroll offset, and `ScrollBars()` draws a floating modern
overlay scrollbar (transparent track, thin thumb; override with
`SetDefaultScrollBar` / `ScrollBarExt`):

```go
Container(Attrs(Viewport), func() {
    ScrollOnInput()
    ScrollBars()
    for _, item := range items {
        ItemRow(item)
    }
})
```

(For long lists, prefer the virtualized containers in Part III — they
scroll without building offscreen rows at all.)

### Focus and keyboard

Text inputs handle their own focus (`FocusOnClick`, `AutoFocus`) and support
tab-cycling between `Focusable` containers (`CycleFocusOnTab`). For your own
focusable widgets, the same primitives are available: `HasFocus()`,
`Focus()`, `Blur()`. Keyboard state is queried like everything else:
`FrameInput.Key` (pressed this frame), `FrameInput.Text` (text typed this
frame, IME-aware), `InputState.Modifiers`, `InputState.DownKeys`.

```go
TextInput(&appData.filter)       // edits the string in place
PasswordInput(&appData.secret)
```

That pointer-passing style is the plain-data payoff: no binding and no change
events. If anything else modifies the string, the input shows it on the next
requested update.

### Clipboard

`RequestTextCopy(text)` queues text for the clipboard; the backend writes
it at the end of the frame. So "copy" is just another thing a widget does
inline (see `LogView`'s hover copy button, or `examples/icons`):

```go
if CtrlButton(SymCopy, "Copy name", true) {
    RequestTextCopy(selected.Name)
}
```

`RequestPaste()` is the read side: the backend fetches the clipboard and
delivers it on a later frame as `FrameInput.Text`, exactly as if typed —
which is how `TextInput` gets paste without special cases.

### Popups, menus, tooltips

`MenuButton`/`MenuItem` and `PopupPanel(&open, anchorId, attrs, fn)` build
floating panels anchored to an element; they render at the root, above
everything, automatically (§2). Any value you used as a container key — or the
`ContainerId` handle returned by `Container`/`ContainerWithKey`, or `CurrentId()` —
works as an anchor.

Widgets to reach for: `Button`/`CtrlButton`, `CheckBox`, `OptionButton`,
`ToggleSwitch`, `Slider`, `TextInput`, `DirectoryBrowse`, `Table`,
`VirtualListView`, `LogView`, `LargeText`, `Link`, `DebugPanel`/`DebugVar`,
and drag-and-drop (`DragAndDrop` / `CanDropHere` — [drag-drop.md](drag-drop.md)).

### Widget commands

Input answers "what did the user do"; sometimes the *app* needs to tell a
widget something imperative — "scroll this row into view" — where passing
state every frame is the wrong shape: the request is an event with a
cause, not a fact that stays true. For this there is a small command
queue:

```go
// the list names itself: its first argument is its key — a typed pointer to
// app-owned data, globally unique among live widgets
VirtualListView(pane, count, itemKey, itemHeight, itemView)

// the app posts at the event site:
VirtualListScrollIntoView(pane, row)
```

`VirtualListScrollIntoView` is a one-line wrapper over the plumbing:
`PostCommand(widget, key, name, arg)` stores one slot per (widget kind,
instance key, command name) — a second post before consumption overwrites
the first, latest intent wins, exactly like sampled input — and the
widget consumes on its next render the way it consumes input, by checking
for data (`TakeCommand[T]`). An unconsumed command expires at the start
of the second frame after the post: long enough to survive any build
order (consumer above or below the poster in the tree), short enough that
a request aimed at a hidden view dies quietly instead of firing when the
view returns.

Conventions: widgets ship named wrappers so call sites read as verbs, and
keep the consuming side package-private; the widget-kind string scopes
commands, so two widget types keyed by the same app object don't collide;
and post on *events*, not conditions — a file pane reveals on arrow keys
but never on mouse clicks, because a click on a half-visible row must not
yank the list under the cursor.

## 7. State across frames: identity, hooks, settling, animation

This section is the heart of writing *correct* Shirei code. Everything in it
follows from one fact: the UI is rebuilt from scratch for each requested frame,
so anything that persists — hover, focus, scroll offsets, widget state,
animation — is keyed by **container identity**, maintained in a persistent
identity tree. Full write-up for app authors: [identity.md](identity.md).

### Identity

Anonymous containers (`Container`, `Element`) are matched **positionally** by
(component type, per-type ordinal), where the component type is the builder's
func literal. Fixed structure and loops with stable membership need nothing
special, and a conditionally inserted sibling of a *different* type does not
shift the others' identity. Only a same-type insertion or reorder shifts
positional identity — which is what explicit keys are for.

For dynamic collections — rows that come and go, reorder, or carry state that
must follow the item — pass an explicit **key**:

```go
ContainerWithKey(itemPtr, Attrs(...), func() { ... })       // pointer to an app object
ContainerWithKey(name, Attrs(...), func() { ... })          // any string — dynamic is fine
ContainerWithKey(fmt.Sprintf("%d", pid), Attrs(...), ...)   // also fine
```

A key is any comparable value you own; it's matched by Go value equality
(pointers by pointer, strings by content) and is **scoped to the parent**: the
same key under two different parents is two distinct containers. The one rule:
keys must be **unique among siblings** within a frame — a duplicate is a program
bug, and shirei reports it loudly on stderr rather than behaving erratically.

A key reconciles the container across frames; to *address* the container after
the fact — focus, hover, geometry, popup anchors — you use its **`ContainerId`**,
an opaque handle that `ContainerWithKey` (and `Container`/`Element`/`CurrentId`/
`GetLastId`) return. Hold that handle and pass it to `IdHasFocus`,
`GetScreenRectOf`, and the like.

For lists of real entities, it's best to use the application's intrinsic way of
reliably identifying entities, whether it be pointers or handles.

This is especially important when the list view can re-arrange (sort) items.
Otherwise, the state would be associated with the position of the item in the
list, and resorting items would mix up their associated states.

```go
for _, item := range sortedItems {
    ContainerWithKey(item.Id, ......)
}
```

### Hooks: widget-local state

`Use[T](key)` attaches a piece of state to the *current container*:

```go
var state = Use[MenuState]("menu-state")
```

It returns a pointer; the first use allocates. The state lives exactly as
long as it keeps being used: **a hook not touched for one full frame is
dropped** — a hidden view forgets its widget-local state (scroll position,
sort order) and reinitializes when shown again. This is deliberate: hooks are
UI state, not application state. Anything that should survive belongs in your
own structs (see Part III).

`UseData[T](ptr, key)` is the other flavor: state attached to an *application
object* rather than a container, surviving regardless of rendering, with
manual cleanup (`DeleteHookedData`). Good for caching derived/parsed data
alongside the object it derives from.

Rule of thumb: app data in your structs; widget-local state in `Use`;
object-derived caches in `UseData`.

### Layout queries and multipass settle

Builders run *before* this frame's layout is fully committed, so every
geometry query — `GetResolvedSize`, `GetContentRect`, `GetScreenRect`, and
the `...Of(id)` variants like `GetScreenRectOf(id)` — answers from the
**previous layout pass**. For a brand-new container that has never been
laid out, that answer is zero until a pass has produced real sizes.

That sounds like a problem for split panes and other “size from parent /
sibling geometry” layouts. In practice the runtime handles the common case:

1. Your builder runs and may query geometry that is not known yet.
2. Layout runs and records sizes for this pass.
3. If any geometry query was unanswered, Shirei **re-runs the same frame**
   (a settle pass) without presenting the incomplete intermediate result.
4. The second pass reads the first pass's sizes, so the frame that reaches
   the screen is already settled for direct dependencies.

```go
// Split by ratio of the current container's resolved height.
// On the first pass GetResolvedSize may be zero; the settle pass re-runs
// the builder once sizes exist — no RequestNextFrame required for that.
totalHeight := GetResolvedSize()[1]
topAttrs := Attrs(Expand, Clip)
if totalHeight > 0 {
    topAttrs = Attrs(FixHeight(totalHeight*splitRatio), Expand, Clip)
}
```

A few nuances:

- **One settle pass** covers the usual “query parent, size child” pattern.
  Longer chains of interdependent geometry may still converge across
  successive *presented* frames when content keeps changing.
- **`RequestNextFrame` is not the geometry fix.** Use it for background
  data, animation you drive yourself, and other out-of-band updates — not
  to “wake” a zero size query.
- **`RenderToPNG` / headless snapshots** run until the frame settles, which
  is why goldens are stable without manual multipass in app code.

See `examples/see_pprof` (`MainContent`) for a real split pane that reads
`GetResolvedSize` while building.

### Animation: on by default

When a container's geometry changes between frames (position, size, padding,
corners, border width, transparency), Shirei **animates** it toward the new
values — this is why UI changes feel fluid with zero effort. A container
appearing for the first time does not animate (there is nothing to animate
from).

Sometimes you don't want it:

- **Hand-positioned content** (`Float` canvases, custom scrollbars, cursors):
  interpolation turns precise placement into drift. Use `NoAnimate`.
- `NoAnimate` **cascades** to children; `YesAnimate` opts an inner container
  back in. A scrollbar does exactly this: `NoAnimate` on the track (because it's
  floating) with `YesAnimate` on the thumb (so it glides).
- `Viewport` includes `NoAnimate` already — panels snap, content animates.

---

# Part III — Application patterns

## 8. Application state: prefer your own structs

For application wide state, define a central state struct:

```go
type AppState struct {
    showGrid bool
    // ...
}

var appData = &AppState{}
```

The UI reads and mutates `appData` directly:

```go
func Toolbar() {
    Container(Attrs(Row, CrossMid, Gap(10)), func() {
        ToggleSwitch(&appata.showGrid)
        Label("Grid")
    })
}

func MainContent() {
    Container(......, func() {
        if appState.showGrid {
            // ... show a grid ...
        }
    })
}
```

## 9. Tables and virtual lists

For the common case — sortable columns over a slice of rows — use the
built-in generic `Table`:

```go
columns := []TableColumn[*Item]{
    {
        Label:  "Name",
        Render: func(it *Item) { Label(it.Name) },
        Less:   func(a, b *Item) bool { return a.Name < b.Name },
    },
    {
        Label: "Size", Width: 90, DefaultDesc: true,
        Render: func(it *Item) { Label(formatBytes(it.Size)) },
        Less:   func(a, b *Item) bool { return a.Size < b.Size },
    },
}

Table(nil, 30, columns, appData.items, func(it *Item) any { return it }, 1)
```

You get clickable sortable headers, aligned columns (one flexible column,
the rest fixed-width), and virtualization for free. The `rowKey` function
must return a stable unique identity per row — the row pointer, per §7. The
last argument picks the initial sort column.

For fully custom lists (non-uniform rows, custom cells), drop down to
`VirtualListView`:

```go
itemKey := func(i int) any { return rows[i] }
itemHeight := func(i int, width f32) f32 { return 30 }
itemView := func(i int, width f32) { RowView(rows[i], i) }

// first argument is the list's own key (nil = anonymous; give it a stable
// value to command it — see below)
VirtualListView(nil, len(rows), itemKey, itemHeight, itemView)
```

Virtualization only builds visible rows, so lists scale to hundreds of
thousands of items. It is the right default for process lists, file lists,
logs, and search results. (For huge text blobs specifically, use the
`LargeText` widget.)

A list with an identity can also be *commanded*: give it its key as the first
argument (`VirtualListView(pane, …)`) and `VirtualListScrollIntoView(pane,
rowPtr)` brings a row into view on the next render, minimally — keyboard
selection's best friend (see §6, Widget commands).

Rows in the list do need to have uniform height, so the list view is *not*
restricted to rendering uniform rows. For example, it can be used to render
a document structure where some elements are headers, some are pargraphs, some
are images, etc.

## 10. Selection

Selection is just a pointer in your app state:

```go
type AppState struct {
    selected *Item
}
```

Row:

```go
func ItemRow(item *Item, idx int) {
    bg := f32(100)
    if idx%2 == 1 {
        bg = 97
    }
    if appData.selected == item {
        bg = 87
    }

    Container(Attrs(Row, Expand, FixHeight(30), Background(220, 8, bg, 1)), func() {
        if IsHovered() {
            ModAttrs(Background(220, 12, 92, 1))
        }
        if IsClicked() {
            appData.selected = item
        }
        Label(item.Name)
    })
}
```

Detail panel:

```go
func DetailPanel() {
    item := appData.selected
    if item == nil {
        Label("Select an item")
        return
    }
    Label(item.Name, FontWeight(WeightBold))
    Label(item.Description)
}
```

An interaction convention that has worked well (see `see_pprof`): **single
click selects, double click acts** (opens, drills in, expands). Because the
first click of a double-click pair fires normally, selection should not be a
click-toggle — clear it with an explicit button or a click on empty
background instead.

## 11. Background work and live data

If a goroutine updates UI state, publish the result under Shirei's frame lock
and then request a new frame:

```go
go func() {
    for {
        data, err := collectData() // expensive work outside the lock

        WithFrameLock(func() {
            appData.data = data
            appData.err = err
        })
        RequestNextFrame()

        time.Sleep(time.Second)
    }
}()
```

Rules:

- Do expensive work outside `WithFrameLock`.
- Mutate shared UI state inside `WithFrameLock`.
- Call `RequestNextFrame()` after publishing.
- App code should not call backend-specific wake functions — backends poll
  Shirei's requested-frame state, keeping the dependency direction clean.

For live monitors, avoid burst sampling: keep the previous sample and compute
deltas from the next one, one collection per refresh interval. (Burst
sampling suits one-shot terminal tools; a continuously-running GUI should
sample smoothly.)

Another use of the same machinery: watching external state. `see_pprof`
watches its directory with fsnotify; the watcher goroutine updates the file
list under `WithFrameLock` and requests a frame — new files appear in the
sidebar the moment they land on disk.

## 12. Stable stores for live entities

The example program `process_monitor` queries the operating system for the list
of running processes and their states at a fixed interval.

When live data arrives as a fresh list each sample, we do not render that list
directly, because we need stable selection, history, expansion state, and
recently-departed items. So we create a stable store:

```go
type ProcessKey struct {
    PID       int
    StartTime time.Time
}

type Process struct {
    ProcInfo
    Key       ProcessKey
    LastSeen  time.Time
    StoppedAt time.Time
    History   []ProcessPoint
    Collapsed bool
}

type ProcessStore struct {
    ByKey map[ProcessKey]*Process
}
```

Each sample:

1. Compute a stable key for each incoming entity.
2. Look up the existing object; update in place, or allocate.
3. Append a history point.
4. Mark missing entities as stopped.
5. Prune old stopped entries unless selected.

This gives us stable pointers for row ids and selection (§7, §10), and a
natural home for history charts and tree expansion state.

---

# Part IV — Case study: patterns from process_monitor

These sections are more specific — they document patterns from building a
live system monitor. Skim them if your app is not monitor-shaped; the
techniques (derived display orders, time-bucketed charts, platform layering)
transfer to other domains.

## 13. Tree views

A tree view is a derived display order over stable objects:

```go
byPID := map[int]*Process{}
children := map[int][]*Process{}
roots := []*Process{}

for _, p := range procs {
    byPID[p.PID] = p
}
for _, p := range procs {
    parent := byPID[p.PPID]
    if parent == nil || parent == p {
        roots = append(roots, p)
    } else {
        children[parent.PID] = append(children[parent.PID], p)
    }
}
```

Flatten depth-first into the visible row list:

```go
func walk(p *Process, depth int) {
    p.TreeDepth = depth
    p.TreeChildCount = len(children[p.PID])
    rows = append(rows, p)

    if p.Collapsed {
        return
    }
    for _, child := range children[p.PID] {
        walk(child, depth+1)
    }
}
```

Render indentation inside the name column so numeric columns stay aligned:

```go
Container(Attrs(Row, Grow(1), CrossMid, Clip), func() {
    Element(Attrs(FixWidth(f32(depth) * 14)))

    if p.TreeChildCount > 0 {
        if PressAction() {
            p.Collapsed = !p.Collapsed
        }
        if p.Collapsed {
            Label("▸")
        } else {
            Label("▾")
        }
    }

    Label(p.Name)
})
```

When filtering a tree: show matches plus their ancestors, and ignore
collapsed state while a filter is active so matches are always visible.

## 14. Simple charts

Charts are just layouts and elements. A bottom-aligned bar chart is a row of
fixed-height columns, each containing a flexible spacer above the bar:

```go
Container(Attrs(Row, FixHeight(chartHeight), Gap(1)), func() {
    for _, b := range buckets {
        Container(Attrs(Grow(1), FixHeight(chartHeight), NoAnimate), func() {
            Filler(1)
            Element(Attrs(FixHeight(barHeight), Expand, Background(hue, sat, light, 1)))
        })
    }
})
```

(Note the `NoAnimate` — per §7, hand-shaped data visualizations should not
interpolate.)

## 15. Platform layers

Keep platform-specific collection separate from UI code, behind one
build-selected function:

```go
func Collect() (RawSnapshot, error)
```

```text
collect_darwin.go      //go:build darwin
collect_linux.go       //go:build linux
collect_windows.go     //go:build windows
collect_unsupported.go //go:build !darwin && !linux && !windows
```

The rest of the app does not know about `libproc`, `/proc`, or Win32 APIs:

```text
platform Collect()
    -> common Sampler computes rates/deltas
        -> stable store updates app-owned objects
            -> UI renders tables, tree views, and charts
```

This separation is what let `process_monitor` support macOS, Linux, and
Windows without touching the GUI.

---

# Part V — Reference

## 16. Common mistakes

### Dynamic lists without keys

Symptoms: duplicate-key reports on stderr; or row state (scroll, hover,
focus, animation) jumping between items when a list sorts, filters, or
reorders.

Fix: for collections of the *same kind of thing* that can change membership
or order, give each item a **key** with `ContainerWithKey` — usually a
pointer or stable id from your app data, never the loop index `i`. Keys must
be unique among siblings under one parent. Anonymous containers are matched
positionally by (component type, per-type ordinal): fixed chrome and
mixed-type siblings are fine without keys; same-type insert/reorder without
keys is what scrambles state. Full rules: [identity.md](identity.md).

### Animation artifacts on hand-positioned content

Symptoms: custom-drawn content (charts, canvases, cursors) drifts, smears,
or lags one step behind interactions.

Fix: `NoAnimate` on hand-positioned containers (§7). `Viewport` already
includes it.

### Extrinsic on a content row collapses height

Symptoms: a toolbar, find bar, or title strip has almost no vertical size
(or disappears), while siblings look fine.

Fix: `Extrinsic` ignores content on **all** axes. A row that only needs a
width budget should not use bare Extrinsic unless the parent also assigns
height (or you set `FixHeight` / a real height constraint). Prefer
content-height chrome (`Expand`, `Clip`, optional `MinHeight`) inside an
**extrinsic pane**; use `Viewport` only for the fill/scroll region. See
§4 (sizing).

### Content inflates a flex pane (scrollbar shoved off)

Symptoms: a long title, path, or text field makes the whole main column
wider than the remaining space; scrollbars or neighbors shift.

Fix: the pane that owns that budget needs `Extrinsic` (often with `Clip`),
not only `Grow`/`Expand`/`Clip`. `Clip` alone does not stop content-driven
measurement. See §4 (sizing).

### Too much work under `WithFrameLock`

Symptoms: UI stutters while background work runs.

Fix: compute outside the lock; publish only final results inside it.

### Forgetting `RequestNextFrame()`

Symptoms: background data changes, but the UI updates only after mouse
movement or another input event.

Fix: after publishing background data, call `RequestNextFrame()`.

### Rendering fresh structs directly

Symptoms: selection/history/expansion state is impossible to maintain.

Fix: a stable store; update objects in place (§12).

### Sample-count charts

Symptoms: chart density changes when the sampling rate changes.

Fix: bucket timestamped samples into fixed time windows (§14).

## 17. A good workflow for building a Shirei app

1. Define your durable app model.
2. If the data source is uncertain, build a terminal prototype first.
3. Create a simple Shirei shell: header, toolbar, main panel, detail panel.
4. Add the `--png` flag on day one (§3); render as you go.
5. Render static or fake data first if needed.
6. Add live background data with `WithFrameLock` and `RequestNextFrame`.
7. Use stable app-owned objects for rows; pointer ids.
8. Add selection (click selects; double-click acts).
9. Add filtering and sorting.
10. Add derived views such as tree mode.
11. Add charts or visual summaries.
12. Snapshot-test the screens you care about; polish layout and styling last.

The most successful Shirei apps share this shape:

```text
platform/data acquisition
    -> common processing
        -> stable app-owned model
            -> derived visible rows/charts
                -> Shirei view functions
                    -> deferred layout and rendering
```



# App resources

Shirei apps keep non-code assets (icons, images, data files) in a **`Resources/`**
directory next to `package main`. The same paths work under `go run` and in a
desktop release package: call sites use `app.ResourcePath` and never hardcode
`.app/Contents/Resources` or similar layout details.

Desktop packaging via [`shirei_bundle`](../cmd/shirei_bundle/README.md) copies
that directory into the platform resource root. The runtime API is the source of
truth; the bundler follows the same convention.

Mobile resource packaging is not part of this path yet.

## Layout

```
myapp/
  main.go
  Resources/
    icon.png
    assets/
      …
```

Name the directory **`Resources`** (capital R). Nested folders inside it are
fine; paths passed to `ResourcePath` are relative to that root.

## API (`go.hasen.dev/shirei/app`)

```go
import app "go.hasen.dev/shirei/app"

app.SetupIcon(app.ResourcePath("icon.png"))
Image(app.ResourcePath("assets/photo.png"), Vec2{0, 120})

root := app.ResourcesDir() // absolute directory, or "" if none found
```

| Function | Role |
|----------|------|
| **`ResourcePath(name)`** | Join `name` under the resolved resources root |
| **`ResourcesDir()`** | Absolute path to that root, or `""` if unresolved |
| **`SetResourcesDir(dir)`** | Pin the root for this process (tests / odd layouts); empty clears the pin |

Environment override: **`SHIREI_RESOURCES`** — absolute or relative path to the
root. Useful in tests. Not a bundler setting.

## Resolution order

1. `SetResourcesDir` pin
2. `SHIREI_RESOURCES`
3. macOS `.app`: `Contents/Resources` next to the executable
4. `<exeDir>/Resources` when that directory exists (Linux/Windows packages, loose binaries)
5. Dev probe for `go run ./pkg` from a module or monorepo root:
   - Prefer `<cwd>/<main-package-base>/Resources` when that directory exists
     (main package path from build info, e.g. `gardener` → `./gardener/Resources`)
   - Else, if the cwd has exactly one immediate child with `Resources/`, use it
   - Else walk from the working directory and executable directory (and parents)
     looking for a `Resources` directory

   Package-local lookup runs before the parent walk so a shared monorepo-level
   `Resources/` (fonts, test data, …) does not shadow `<package>/Resources`.

Call sites always go through `ResourcePath` / `ResourcesDir`. Do not open
`Contents/Resources` or `<exeDir>/Resources` by hand.

## Packaging (`shirei_bundle`)

When you add an app in the bundler UI, the icon field defaults to
`Resources/icon.png` if that file exists (otherwise `icon.png` beside the
package). Prefer keeping the dock/launcher icon there so runtime
`SetupIcon(app.ResourcePath("icon.png"))` and packaging share one file.

When `<package>/Resources` exists, desktop builds copy its **contents** into the
platform resource root:

| Platform | Destination |
|----------|-------------|
| macOS | `App.app/Contents/Resources/` |
| Linux | `<exeDir>/Resources/` inside the tarball |
| Windows | `<exeDir>/Resources/` inside the zip |

No bundler config field: presence of the directory is enough. Details and CLI/GUI
workflow: [shirei_bundle README](../cmd/shirei_bundle/README.md).

## Tips

- Prefer `ResourcePath` at every load site (`Image`, `SetupIcon`, `os.ReadFile`, …).
- Large or generated blobs belong in `Resources/` too — they ship beside the
  binary without forcing a Go rebuild of unrelated code when only assets change.
- If `ResourcesDir()` is empty, `ResourcePath` returns the cleaned `name` alone;
  opens will fail until a root is found or pinned.

  # Building a chat-shell layout in Shirei

This tutorial builds a multi-panel **chat shell** one step at a time. Early
steps use loud colors so every container “box” is obvious; later steps fill
them with content and polish to a **light** chrome.

It is for humans learning layout and for AI agents that need runnable samples.

Inspired by multi-column chat apps; this is **not** a product clone.

## How this tutorial is written

- **Step 01** shows a complete small program (starting point).
- **Later steps** show only **what changed**, as one or more ` ```diff ` chunks,
  with a short note on each chunk.
- **Full source** for every step lives under
  [`demos/layout-shell/`](../demos/layout-shell/) — follow the link if you want
  the whole file.
- Diffs deliberately **omit** noise: file-header comments, `SetupWindow` title
  strings, and dummy chat copy you can ignore. Focus on `frame` (and data only
  when it matters).

**Engine root:** `frameFn` already runs inside a window-sized, clipped root.
Do **not** wrap the whole UI in `Viewport` as an “app root.”

**Compose appears early:** step **05** subdivides main into header / messages /
compose. Steps **09–11** teach why the messages pane needs **`Extrinsic` /
`Viewport`**. Compose becomes a real `TextInput` when messages are filled.

| Steps | Focus |
|-------|--------|
| 01–04 | Outer shell |
| **05** | Main: header · messages · **compose** |
| 06–08 | Labels, servers, channels |
| **09–11** | Messages + Extrinsic / Viewport |
| 12–13 | Members + light polish |
| 14 | VirtualList at scale |

**Final result (step 14):**

![Step 14](layout-tutorial/images/step14.png)

When the shell is solid, a separate tutorial builds a **custom compose bar**
(padded field + circular send, typical of modern chat-style UIs) on top of
step 14 — process APIs, not more Extrinsic/Viewport:
[custom-widgets-tutorial.md](custom-widgets-tutorial.md).

## Prerequisites

Skim [tutorial.md](tutorial.md) for containers, `Attrs`, and `Label`.

## How to run a step

```bash
cd shirei
go run ./demos/layout-shell/step05
go run ./demos/layout-shell/step09   # intentional bug
./demos/layout-shell/gen-pngs.sh
```

Screenshots are **1100×720**.

---

## Wireframe

```text
┌──────────────────────── top bar ─────────────────────────┐
├────┬────────────┬─────────────────────────┬──────────────┤
│ S  │  channels  │  header                 │   members    │
│ e  │            ├─────────────────────────┤              │
│ r  │            │  messages (scroll)      │              │
│ v  │            ├─────────────────────────┤              │
│ e  │            │  compose (TextInput)    │              │
└────┴────────────┴─────────────────────────┴──────────────┘
```

| Tool | Role |
|------|------|
| Engine root | Window-sized + clipped |
| `Grow` / `Row` / column | Flex shell |
| **`Extrinsic`** / **`Viewport`** | Budgeted scroll panes |
| `TextInput` | Real compose field |
| `VirtualListView` | Large lists (step 14) |

---

## Step 01 — The engine root

Style the engine root with `ModAttrs` — no outer `Viewport` window.

![Step 01](layout-tutorial/images/step01.png)

**Full source:** [`demos/layout-shell/step01/main.go`](../demos/layout-shell/step01/main.go)

```bash
go run ./demos/layout-shell/step01
```

```go
package main

import (
	"fmt"
	"os"

	"go.hasen.dev/shirei/app"

	. "go.hasen.dev/shirei"
)

const winW, winH = 1100, 720

func main() {
	if len(os.Args) >= 3 && os.Args[1] == "--png" {
		if err := RenderToPNG(os.Args[2], winW, winH, frame); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	app.SetupWindow("Layout shell — step 01", winW, winH)
	app.Run(frame)
}

func frame() {
	// frameFn already runs inside an engine-created root: Min/Max/resolved size
	// = WindowSize, Clip = true. You do not add a Viewport "window root".
	// Style the current container (the root) before adding children:
	ModAttrs(Background(220, 20, 88, 1))
}
```

**What to notice:** Agents often wrap everything in `Viewport`. The engine
already created a full-window root. `Viewport` is for **nested** fill-and-scroll
panes (we introduce it properly after a real bug in step 09).

The `main` + `--png` boilerplate is the same in every step — later diffs skip it.

---

## Step 02 — Top bar + body

Split the root column: fixed-height top, growing body.

![Step 02](layout-tutorial/images/step02.png)

**Full source:** [`step02/main.go`](../demos/layout-shell/step02/main.go)

Replace the single `ModAttrs` paint with two children of the root:

```diff
 func frame() {
-	ModAttrs(Background(220, 20, 88, 1))
+	// Top bar: fixed height, does not grow.
+	Container(Attrs(Expand, FixHeight(48), Background(280, 45, 42, 1)), func() {})
+
+	// Body: Grow(1) takes all remaining height on the main axis.
+	Container(Attrs(Grow(1), Expand, Background(210, 25, 72, 1)), func() {})
 }
```

**What to notice:** The root is a **column**. `Grow(1)` claims leftover
**height**. The top bar stays 48px.

---

## Step 03 — Server rail + rest

Turn the body into a **row**: narrow rail + growing rest.

![Step 03](layout-tutorial/images/step03.png)

**Full source:** [`step03/main.go`](../demos/layout-shell/step03/main.go)

```diff
 func frame() {
 	Container(Attrs(Expand, FixHeight(48), Background(280, 45, 42, 1)), func() {})
 
-	Container(Attrs(Grow(1), Expand, Background(210, 25, 72, 1)), func() {})
+	// Body is a row: fixed-width rail + growing rest.
+	Container(Attrs(Row, Grow(1), Expand), func() {
+		Container(Attrs(FixWidth(72), Expand, Background(260, 40, 32, 1)), func() {})
+		Container(Attrs(Grow(1), Expand, Background(210, 25, 72, 1)), func() {})
+	})
 }
```

**What to notice:** In a `Row`, `Grow(1)` claims leftover **width**. `FixWidth(72)`
keeps the rail narrow.

---

## Step 04 — Channels | main | members

Split the “rest” into three columns.

![Step 04](layout-tutorial/images/step04.png)

**Full source:** [`step04/main.go`](../demos/layout-shell/step04/main.go)

```diff
 	Container(Attrs(Row, Grow(1), Expand), func() {
 		Container(Attrs(FixWidth(72), Expand, Background(260, 40, 32, 1)), func() {})
-		Container(Attrs(Grow(1), Expand, Background(210, 25, 72, 1)), func() {})
+		// Rest of the body: three columns
+		Container(Attrs(Row, Grow(1), Expand), func() {
+			Container(Attrs(FixWidth(240), Expand, Background(220, 30, 52, 1)), func() {})
+			Container(Attrs(Grow(1), Expand, Background(200, 20, 78, 1)), func() {})
+			Container(Attrs(FixWidth(220), Expand, Background(150, 35, 48, 1)), func() {})
+		})
 	})
```

**What to notice:** Classic app chrome — fixed side panels, growing center.
You can nest rows freely.

---

## Step 05 — Main: header | messages | compose

Subdivide the **center** column so compose is a permanent layout slot (not a
late surprise).

![Step 05](layout-tutorial/images/step05.png)

**Full source:** [`step05/main.go`](../demos/layout-shell/step05/main.go)

### Chunk 1 — Optional labels on the outer panes

(Also adds `Center` so the labels read cleanly.)

```diff
-		Container(Attrs(FixWidth(72), Expand, Background(260, 40, 32, 1)), func() {})
+		Container(Attrs(FixWidth(72), Expand, Background(260, 40, 32, 1), Center, Pad(4)), func() {
+			Label("servers", FontSize(12), FontWeight(WeightSemibold), TextColor(0, 0, 100, 1))
+		})
```

Same idea for channels and members (labels only).

### Chunk 2 — The important change: three rows inside main

```diff
-			Container(Attrs(Grow(1), Expand, Background(200, 20, 78, 1)), func() {})
+			// Main column: three rows — header, growing messages, fixed compose.
+			Container(Attrs(Grow(1), Expand, Background(200, 20, 78, 1)), func() {
+				Container(Attrs(Expand, FixHeight(48), Background(200, 25, 70, 1), Center), func() {
+					Label("header", FontSize(14), FontWeight(WeightSemibold), TextColor(0, 0, 100, 1))
+				})
+				Container(Attrs(Grow(1), Expand, Background(200, 15, 82, 1), Center), func() {
+					Label("messages", FontSize(14), FontWeight(WeightSemibold), TextColor(0, 0, 100, 1))
+				})
+				Container(Attrs(Expand, FixHeight(56), Background(200, 30, 65, 1), Center), func() {
+					Label("compose", FontSize(14), FontWeight(WeightSemibold), TextColor(0, 0, 100, 1))
+				})
+			})
```

**What to notice:** Chat main is its own column recipe: fixed header · growing
middle · fixed compose. That structure stays for the rest of the tutorial.

---

## Step 06 — Name every region

Mostly the same layout as step 05 (labels already present). Top bar gets a
label too. No structural change.

![Step 06](layout-tutorial/images/step06.png)

**Full source:** [`step06/main.go`](../demos/layout-shell/step06/main.go)

```diff
-	Container(Attrs(Expand, FixHeight(48), Background(280, 45, 42, 1), Center), func() {
-		// (empty or unlabeled in earlier variants)
-	})
+	Container(Attrs(Expand, FixHeight(48), Background(280, 45, 42, 1), Center), func() {
+		Label("top bar", FontSize(16), FontWeight(WeightSemibold), TextColor(0, 0, 100, 1))
+	})
```

**What to notice:** If your step 05 already labeled everything, this step is a
checkpoint — the screenshot is the map of the shell.

---

## Step 07 — Server icons

Fill the rail with dummy servers; main slots stay placeholders.

![Step 07](layout-tutorial/images/step07.png)

**Full source:** [`step07/main.go`](../demos/layout-shell/step07/main.go)

### Chunk 1 — Data

```diff
+var servers = []struct {
+	letter string
+	hue    float32
+}{
+	{"A", 10}, {"B", 40}, {"C", 120}, {"D", 200}, {"E", 280}, {"F", 320},
+}
```

### Chunk 2 — Rail becomes a column of tiles

```diff
-		Container(Attrs(FixWidth(72), Expand, Background(260, 40, 32, 1), Center, Pad(4)), func() {
-			Label("servers", ...)
-		})
+		Container(Attrs(FixWidth(72), Expand, Background(260, 40, 32, 1), Pad(8), Gap(8)), func() {
+			for _, s := range servers {
+				Container(Attrs(FixSize(48, 48), Corners(16), Background(s.hue, 55, 50, 1), Center), func() {
+					Label(s.letter, FontSize(18), FontWeight(WeightBold), TextColor(0, 0, 100, 1))
+				})
+			}
+		})
```

**What to notice:** Filling a pane does not change the outer shell. Header /
messages / compose placeholders remain.

---

## Step 08 — Channel list

Fill channels with a short scrollable list.

![Step 08](layout-tutorial/images/step08.png)

**Full source:** [`step08/main.go`](../demos/layout-shell/step08/main.go)

### Chunk 1 — Data

```diff
+var channels = []string{
+	"general", "random", "help", /* ... */,
+}
```

### Chunk 2 — Channels column: header + scroll body

```diff
-			Container(Attrs(FixWidth(240), Expand, Background(...), Center), func() {
-				Label("channels", ...)
-			})
+			Container(Attrs(FixWidth(240), Expand, Background(220, 30, 52, 1)), func() {
+				Container(Attrs(Expand, FixHeight(44), Pad2(0, 12), CrossMid), func() {
+					Label("Channels", FontSize(14), FontWeight(WeightBold), TextColor(0, 0, 100, 1))
+				})
+				// Grow+Clip+Scroll — fine for a *short* list.
+				Container(Attrs(Grow(1), Expand, Clip, Pad2(4, 8), Gap(2)), func() {
+					ScrollOnInput()
+					for _, name := range channels {
+						Container(Attrs(Expand, Pad2(6, 8), Corners(4), Background(0, 0, 0, 0.12)), func() {
+							Label("# "+name, FontSize(14), TextColor(0, 0, 98, 1))
+						})
+					}
+				})
+			})
```

**What to notice:** Twelve channels fit. The same `Grow`+`Clip` recipe will
fail when the middle list is taller than its budget — next step.

---

## Step 09 — Compose bug (intentional)

Fill **messages** with a long list using only `Grow`+`Expand`+`Clip`. Watch the
compose strip.

![Step 09](layout-tutorial/images/step09.png)

**Full source:** [`step09/main.go`](../demos/layout-shell/step09/main.go)

### Chunk 1 — Widgets import + draft buffer + message data

```diff
 import (
 	...
 	. "go.hasen.dev/shirei"
+	. "go.hasen.dev/shirei/widgets"
 )
+
+var messages = []msg{ /* ~15 lines of dummy chat */ }
+var draft string
```

### Chunk 2 — Messages pane: the **wrong** recipe

```diff
-				Container(Attrs(Grow(1), Expand, Background(200, 15, 82, 1), Center), func() {
-					Label("messages", ...)
-				})
-				Container(Attrs(Expand, FixHeight(56), Background(200, 30, 65, 1), Center), func() {
-					Label("compose", ...)
-				})
+				// WRONG on purpose: Grow+Expand+Clip without Extrinsic.
+				Container(Attrs(Grow(1), Expand, Clip, Pad(12), Gap(10)), func() {
+					ScrollOnInput()
+					for _, m := range messages {
+						// author / time / body labels...
+					}
+				})
+				Container(Attrs(Expand, Pad(10), Gap(8), Row, CrossMid, Background(0, 0, 0, 0.12)), func() {
+					a := DefaultTextInputAttrs()
+					a.NoAutoFocus = true
+					TextInputExt(&draft, a)
+					Button(NoIcon, "Send")
+				})
```

**What to notice:** Compose is **half-cut or gone**. Without **`Extrinsic`**,
the middle still measures from its children, so the whole main column grows
past the window. `Grow` alone does not mean “take remaining budget only.”

---

## Step 10 — Fix with Extrinsic

Same layout; pin the message region’s height to the parent budget.

![Step 10](layout-tutorial/images/step10.png)

**Full source:** [`step10/main.go`](../demos/layout-shell/step10/main.go)

One decisive change in the messages container:

```diff
-				Container(Attrs(Grow(1), Expand, Clip, Pad(12), Gap(10)), func() {
+				// FIX: Extrinsic — middle height from parent, not message count.
+				Container(Attrs(Grow(1), Expand, Clip, Extrinsic, Pad(12), Gap(10)), func() {
 					ScrollOnInput()
 					for _, m := range messages {
```

(Optional: apply `Extrinsic` on the channel list the same way for consistency.)

**What to notice:** Compose is fully visible. Scroll still works inside the
budgeted middle. **`Extrinsic`** = size from constraints, not from content.

---

## Step 11 — Viewport helper

Package the step-10 recipe as `Viewport`.

![Step 11](layout-tutorial/images/step11.png)

**Full source:** [`step11/main.go`](../demos/layout-shell/step11/main.go)

```diff
-				Container(Attrs(Grow(1), Expand, Clip, Extrinsic, Pad(12), Gap(10)), func() {
+				// Viewport = Clip + Extrinsic + Expand + Grow(1) + NoAnimate
+				Container(Attrs(Viewport, Pad(12), Gap(10)), func() {
```

Same substitution for any other scroll pane you already fixed with Extrinsic
(e.g. channels).

**What to notice:** `Viewport` is for **nested** fill-and-scroll regions
(message lists, sidebars) — not an outer app root. That is why it exists.

---

## Step 12 — Members list

Fill the right rail with the same Viewport pattern.

![Step 12](layout-tutorial/images/step12.png)

**Full source:** [`step12/main.go`](../demos/layout-shell/step12/main.go)

### Chunk 1 — Data

```diff
+var members = []struct {
+	name string
+	hue  float32
+}{
+	{"alex", 10}, {"blair", 40}, /* ... */,
+}
```

### Chunk 2 — Members column

```diff
-			Container(Attrs(FixWidth(220), Expand, Background(...), Center), func() {
-				Label("members", ...)
-			})
+			Container(Attrs(FixWidth(220), Expand, Background(150, 35, 48, 1)), func() {
+				Container(Attrs(Expand, FixHeight(44), Pad2(0, 12), CrossMid), func() {
+					Label("Members — online", ...)
+				})
+				Container(Attrs(Viewport, Pad2(6, 10), Gap(6)), func() {
+					ScrollOnInput()
+					for _, m := range members {
+						// avatar circle + name
+					}
+				})
+			})
```

**What to notice:** Same recipe as channels/messages. Short lists can stay a
plain loop inside `Viewport`.

---

## Step 13 — Polish (light)

Same structure; light chrome so standard widgets read cleanly.

![Step 13](layout-tutorial/images/step13.png)

**Full source:** [`step13/main.go`](../demos/layout-shell/step13/main.go)

### Chunk 1 — Palette + root background

```diff
 func frame() {
+	const (
+		bgMain, bgSide, bgRail float32 = 97, 94, 92
+		textPrim, textMuted    float32 = 18, 45
+		borderA                float32 = 0.08
+	)
+	ModAttrs(Background(220, 6, bgMain, 1))
```

### Chunk 2 — Restyle each pane

Swap loud debug `Background(...)` for light greys, dark text, hairline
separators (`Element` 1px). Selection tint on `# general` stays subtle.

Structure (rows/columns/`Viewport`/compose) is **unchanged** — only colors and
labels like `"Layout shell"`.

**What to notice:** Compose remains a real `TextInput` that **grows** in the
row by default (`FixedWidth` would pin a chip-sized field).

---

## Step 14 — VirtualList for scale

Same light shell; messages and members use `VirtualListView` so hundreds of
rows stay cheap.

![Step 14](layout-tutorial/images/step14.png)

**Full source:** [`step14/main.go`](../demos/layout-shell/step14/main.go)

### Chunk 1 — Scale the data + list keys

```diff
-var messages = []msg{ /* 15 hand-written rows */ }
-var members = []struct{ ... }{ /* 8 people */ }
+var (
+	messages   []msg
+	members    []member
+	msgList    = new(int) // VirtualList identity keys
+	memberList = new(int)
+)
+
+func init() {
+	// e.g. 800 messages, 250 members
+}
```

### Chunk 2 — Messages body: loop → VirtualList

```diff
-				Container(Attrs(Viewport, Pad(14), Gap(12)), func() {
-					ScrollOnInput()
-					for _, m := range messages {
-						// row UI...
-					}
-				})
+				Container(Attrs(Grow(1), Expand), func() {
+					VirtualListView(msgList, len(messages),
+						func(i int) any { return messages[i].id },
+						nil, // auto-height via Measure
+						func(i int, width f32) {
+							// same row UI as before, for messages[i]
+						},
+					)
+				})
```

### Chunk 3 — Members: same swap

```diff
-				Container(Attrs(Viewport, ...), func() {
-					ScrollOnInput()
-					for _, m := range members { ... }
-				})
+				Container(Attrs(Grow(1), Expand), func() {
+					VirtualListView(memberList, len(members),
+						func(i int) any { return members[i].id },
+						nil,
+						func(i int, width f32) { /* row UI */ },
+					)
+				})
```

Channels can stay a small `Viewport`+loop.

**What to notice:** Layout shell is unchanged. VirtualList only builds visible
rows; `ItemHeight` nil measures each row with `Measure`. Details:
[virtual-list.md](virtual-list.md).

---

## Recap

1. Outer shell (01–04), then **lock compose into the main column (05)**.
2. Fill content gradually (06–08).
3. **09 → 10 → 11:** broken compose → **`Extrinsic`** → **`Viewport`**.
4. Members, light polish, VirtualList at scale (14).

## Next: custom widgets

Layout ends at step 14. A **separate** tutorial teaches Shirei’s process vs
presentation model (custom button → custom field → compose bar), then product
polish (dark shell + optional scrollbar tint):

**[Custom widgets: process, paint, and a chat compose bar](custom-widgets-tutorial.md)**

Samples:
[`step15a`](../demos/layout-shell/step15a/) (custom send only),
[`step15`](../demos/layout-shell/step15/) (full compose),
[`step16`](../demos/layout-shell/step16/) (dark shell).

## Common mistakes

- Outer `Viewport` around the whole UI.
- Message list with only `Grow`+`Clip` (step 09).
- Forgetting `Grow` on the center column of a row shell.
- Plain loops for huge histories (step 14).

## Source map

| | |
|--|--|
| Steps | [`demos/layout-shell/stepNN/`](../demos/layout-shell/) (01–14) |
| Images | [`layout-tutorial/images/`](layout-tutorial/images/) |
| Regen | `demos/layout-shell/gen-pngs.sh` |
| Custom compose follow-up | [custom-widgets-tutorial.md](custom-widgets-tutorial.md) · [`step15/`](../demos/layout-shell/step15/) |



# Container Identity

Shirei rebuilds the current UI description for each requested frame. That
raises a practical question: when a row scrolls, stays hovered, or keeps a text
field’s caret, how does Shirei know it is looking at the *same* container as the
previous frame?

This document aims to answer these questions, and explain when you can ignore it,
and when you must give items an explicit key.

---

## 1. The short version

Most of the time you write `Container(...)` and nothing special is needed.
Shirei matches containers by **where they sit** in the tree: “the first
button under this panel,” “the third row in this loop,” and so on.

That works for fixed layouts and for lists that don’t change order or
membership.

When items can appear, disappear, reorder, or move between parents — and
you care that *each item* keeps its own scroll position, focus, or widget
state — give that item an explicit key with `ContainerWithKey`.

```go
// Good for a list that can sort, filter, or reorder:
for _, item := range items {
    ContainerWithKey(item.Id, Attrs(...), func() {
        // ...
    })
}
```

Use something stable that names the *data* (an id field, a pointer to your
struct) — not the loop index `i`, which changes meaning when the list
shuffles.

---

## 2. What goes wrong if identity is wrong

If shirei thinks two different items are “the same” container (or loses
track of one), you may see:

- hover or focus jumping to another row after a sort
- a scrolled panel jumping back to the top
- a text field forgetting its caret or selection
- a brief animation glitch as if the row were brand new

Those symptoms usually mean a dynamic list needed `ContainerWithKey`, or
used a key that wasn’t unique / wasn’t stable.

---

## 3. Explicit ids (`ContainerWithKey`)

```go
ContainerWithKey(item.Id, attrs, func() { ... })
```

The first argument is a **key**: any comparable Go value you own — a
string, an int, a pointer to your object. Fresh strings each frame are
fine (`fmt.Sprintf("row-%d", id)`).

**Rules that matter in practice:**

- **Unique among siblings.** Under one parent, don’t use the
  same key twice in the same frame. (The same key under *different*
  parents is fine — two panels can each have a `"header"`.)
- **Stable for the item.** Prefer `item.Id` or `&item` over the position
  in the slice.
- **Follows the item.** If a card moves from one column to another
  (drag-and-drop), keep the same key so its identity travels with it.

You only need this for containers whose *per-item* UI state should stick
to the data. Static chrome (title bars, fixed toolbars) can stay plain
`Container`.

---

## 4. Things that appear and disappear

A common worry: “If I show a banner above my list, do all the rows get new
ids and lose their state?”

**Usually no.** Optional UI that is a *different* piece of code than the
list rows — a banner function, a divider, an empty-state message — does
not scramble the rows below it. Turning that banner on or off is safe
without keys on the rows.

**Where you do need care:** a list of similar rows that can grow, shrink,
or reorder. Then positional matching (“third row”) is the wrong idea —
the third *visible* row isn’t always the same *item*. Key those rows.

| What you do | What happens to the other items |
|-------------|----------------------------------|
| Show/hide a banner or toolbar above a list | List rows keep their identity |
| Insert/remove/reorder similar list rows without keys | Later rows can “inherit” the wrong state |
| Filter a keyed list (some items not drawn this frame) | Other keyed items are unaffected |
| Bring a keyed item back | It is recognized as the same item again |

**Practical rule:** if the children are “many of the same kind of thing”
and the set can change, use `ContainerWithKey`. If you’re toggling
unrelated chrome around a stable list, you’re fine.

---

## 5. Two different “ids” (don’t mix them up)

| What | What it’s for |
|------|----------------|
| **Key** — the value you pass to `ContainerWithKey` | Tells shirei “this is item X” across frames |
| **`ContainerId`** — returned by `Container` / `ContainerWithKey`, or from `CurrentId()` | A handle you pass to helpers like “is this focused?” or “where is this on screen?” |

You pick the key from your app data. Shirei gives you the `ContainerId`
when you need to ask questions about that container later.

---

## 6. Checklist

- Fixed layout, nothing reordering → plain `Container` is enough.
- List that sorts, filters, or virtualizes → `ContainerWithKey` per item.
- Key by the item’s real identity, not by `i`.
- Don’t reuse the same key twice under one parent in one frame.
- Optional banners/dividers above a list usually don’t require re-keying
  the list.
- If hover/focus/scroll “follows the wrong row” after a data change, add
  or fix keys first.



  # Drag and Drop in Shirei

How to move items between drop zones with the `widgets` drag-and-drop API.
Companion to [tutorial.md](tutorial.md) (the general overview); this
document covers only item DnD.

Reference demos:

- `demos/balls-buckets` — balls into lettered buckets (and back out)
- `demos/kanban` — cards between lane columns

---

## 1. Two different “drags”

The main tutorial’s **Dragging** section (`PressAction` / `IsActive` +
`FrameInput.Motion`) is for *pointer capture*: splitters, panning a custom
canvas, scrubbing a slider thumb. That path does not transfer application
data between widgets.

**Item drag-and-drop** is a separate mechanism in `shirei/widgets`:

| Concern | Tool |
|---------|------|
| Which widget is being dragged | `CurrentId()` (opaque `ContainerId`) |
| What data the drag carries | **payload** argument to `DragAndDrop` |
| What drop zone was hit | **target** argument to `CanDropHere` |

Payloads and targets are ordinary Go values you pass in each frame — typically
small typed ids (`BallPayload("red")`, `BucketTarget("A")`) that your drop
handler uses to update app state.
---

## 2. The pieces

```go
import . "go.hasen.dev/shirei/widgets"

// On the draggable item (inside its container):
if DragAndDrop(payload) {
    // mouse released over a zone that accepted this payload type
    target := GetDropTarget[TargetType]()
    // mutate your app state
}

// On each drop zone:
if CanDropHere[PayloadType](target) {
    // true while this zone is the active drop target — use for highlight
}

// Optional feedback on the item itself:
if IsDragging() { /* dim / restyle the source */ }

// Optional floating ghost (usually at the end of the frame’s UI):
if item, ok := GetDraggingItem[PayloadType](); ok {
    rect := GetDraggingItemRect()
    // Float is parent-relative; rect is surface-absolute (MousePoint space):
    //   origin := Vec2Sub(rect.Origin, GetRenderData().ResolvedOrigin)
    //   FloatVec(origin), FixSizeVec(rect.Size), ClickThrough, …
}
```

- `DragAndDrop(payload)` — call every frame on the item. Arms on mouse-down
  and only begins a real drag after a small movement threshold (so clicks
  and double-clicks are not swallowed). Returns `true` once on successful
  drop. Double-click presses (`ClickCount >= 2`) do not arm a drag.
- `CanDropHere[Accept](target)` — call every frame on the zone. Registers
  `target` while hovered **only if** the active drag’s payload is type
  `Accept`. Returns whether this zone is currently the drop target.
- `GetDropTarget[T]()` — read the zone’s `target` in the drop handler.
- `GetDraggingItem[T]()` / `GetDraggingItemRect()` — for ghosts and status.

Define distinct named types for payload and target so the type parameter on
`CanDropHere` filters correctly (e.g. only balls drop on buckets):

```go
type BallPayload string
type BucketTarget string
```

---

## 3. Minimal pattern (from balls-buckets)

App state stays yours — e.g. each ball has a `Bucket` field (`""` =
unassigned). The view only renders and wires DnD:

```go
// Drop zone
target := BucketTarget("A")
ContainerWithKey(target, bucketAttrs, func() {
    if CanDropHere[BallPayload](target) {
        ModAttrs(/* highlight */)
    }
    // … children, including balls currently in this bucket …
})

// Draggable item
payload := BallPayload(ball.Id)
ContainerWithKey(payload, ballAttrs, func() {
    if IsDragging() {
        ModAttrs(/* faded source */)
    }
    if DragAndDrop(payload) {
        ball.Bucket = string(GetDropTarget[BucketTarget]())
    }
    Label(ball.Name)
})
```

On drop, mutate durable state (`ball.Bucket = …`). The next frame redraws
the ball under the new parent. No widget object graph to update.

A tray that “unassigns” is just another zone whose target is
`BucketTarget("")` (or any sentinel your state understands).

---

## 4. Ghost preview

While a drag is active, draw a floating copy that follows the pointer.
Do this **after** the main layout so it stacks above content; mark it
`ClickThrough` so it does not steal hover from drop zones:

```go
if payload, ok := GetDraggingItem[BallPayload](); ok {
    rect := GetDraggingItemRect()
    // rect.Origin is surface-absolute; Float is parent-relative.
    origin := Vec2Sub(rect.Origin, GetRenderData().ResolvedOrigin)
    ContainerWithKey("dnd-ghost", ballAttrsFor(payload), func() {
        ModAttrs(NoAnimate, FloatVec(origin), FixSizeVec(rect.Size),
            ClickThrough, Trans(0.55))
        Label(nameFor(payload))
    })
}
```

`GetDraggingItemRect()` tracks the origin captured at mouse-down plus
accumulated `FrameInput.Motion` (surface space, same as `MousePoint`).

---

## 5. Rules of thumb

**Hover includes ancestors.** `CanDropHere` uses `IsHovered()`, which is
true for a zone when the pointer is over it *or* a descendant (e.g. a card
inside a lane). Put `CanDropHere` on the zone container that should accept
the drop, not only on empty padding.

**Ordered lists: prefer one zone + geometry.** Gaps between items are not
part of either child, so per-item `CanDropHere` misses “drop between.”
Use a single stable target on the list/lane, compute the insertion index
from pointer Y vs item midpoints (previous-frame rects via
`GetRenderDataOf`), and draw markers in layout gaps. See `demos/kanban`.

**Targets should be comparable.** Clearing the active zone compares
`DropTarget == target` with Go interface equality. Typed strings/ints work;
avoid non-comparable payloads as targets.

**One drag at a time.** State is process-global in the widgets package —
fine for a single window’s UI.

**Stable item identity.** When the same logical item can appear under
different parents across frames (a ball moving from tray to bucket), give
its container an explicit key via `ContainerWithKey` so hover/drag identity
follows the item. That key is separate from the DnD payload; demos often
pass the same value to both for convenience.

**Not for OS file drops.** This API is in-app item transfer. Platform file
open/drag from Finder/Explorer is a different path (if/when exposed by the
backend).
---

## 6. API checklist

| Function | Where | Role |
|----------|-------|------|
| `DragAndDrop(payload)` | item | start/continue drag; `true` on drop |
| `CanDropHere[Accept](target)` | zone | accept + highlight |
| `IsDropTarget(target)` | anywhere | active target? (markers in gaps) |
| `IsDragging()` | item | source is the active drag |
| `GetDropTarget[T]()` | on drop | zone payload |
| `GetDraggingItem[T]()` | anywhere | active item payload |
| `GetDraggingItemRect()` | ghost | floating rect |

Full examples: `go run ./demos/balls-buckets` and `go run ./demos/kanban`.


# Custom widgets: process, paint, and a chat compose bar

This is a **follow-up** to the [layout tutorial](layout-tutorial.md). Layout
taught multi-panel structure (shell, Extrinsic, Viewport, VirtualList). This
tutorial teaches a different skill: **how Shirei expects you to customize
interactive controls**.

We build on layout **step 14** in two product steps:

1. **Philosophy** — process vs presentation  
2. **A flat button** — easiest way into the model  
3. **A text field** — process plus plain text/caret draw  
4. **Compose** — put the button next to the field in one chrome box  
5. **Dark shell** — recolor the finished chat and optionally retint the
   package default scrollbar for dark panels  

Runnable samples (each is a full window you can open while reading):

| Step | Code | What you learn |
|------|------|----------------|
| 14 | [layout step 14](../demos/layout-shell/step14/main.go) | Starting shell (default compose) |
| 15a | [`step15a/main.go`](../demos/layout-shell/step15a/main.go) | Custom send circle, **default** text field |
| 15 | [`step15/main.go`](../demos/layout-shell/step15/main.go) | Full custom compose (pill + field + send) |
| 16 | [`step16/main.go`](../demos/layout-shell/step16/main.go) | Dark theme + light scrollbar tint |

```bash
cd shirei
go run ./demos/layout-shell/step14    # before: default field + default Send
go run ./demos/layout-shell/step15a   # custom circle, default field
go run ./demos/layout-shell/step15    # full custom compose
go run ./demos/layout-shell/step16    # dark shell
```

Screenshots of the shell appear **with each section** below (and again in
the recap). The progression is easier to see in order than only at the top.

---

## Prerequisites

Finish [layout-tutorial.md](layout-tutorial.md) through **step 14**, or at
least run and skim
[`step14/main.go`](../demos/layout-shell/step14/main.go). You should know
containers, `Attrs`, rows/columns, and that compose is already a layout slot.

Optional deeper references (not required to follow along):

- Gallery demos: [custom-buttons](../demos/custom-buttons/),
  [custom-textinputs](../demos/custom-textinputs/)

---

## 1. Philosophy: process vs presentation

Default widgets (`Button`, `TextInput`, …) are **convenience packages**: they
wire up interaction **and** draw a default look. That is fine until the
default look is wrong for your app.

A common industry move is to invent “themes,” skin interfaces, or
`Draw(state)` callbacks the framework calls. Shirei takes a direct,
compositional path:

> **You own the container.**  
> **We provide functions that process input for the current container.**  
> **You paint whatever you want from the snapshot they return.**

There is no plug-in skin object. There is no inverted “framework calls your
draw.” You build the tree; when you need “is this box hovered / clicked /
being edited?”, you call a **process** helper on that box.

```text
  ┌─────────────────────────────────────────┐
  │  your Container (you set size, pad, bg) │
  │                                         │
  │   st := Process…Events(...)             │  ← interaction (and for text, edits)
  │   // use st.Hovered, st.Clicked, …      │
  │   Label / Icon / Draw…                  │  ← presentation (yours, or plain helpers)
  └─────────────────────────────────────────┘
```

### Why this shape?

- **State depends on the box.** Hover, press, and focus are about *this*
  container’s id and geometry. Handing state into a nested “view callback”
  the shell owns is circular — the shell would need the view to exist before
  it could compute the state the view needs.
- **Data-centric.** Process returns a plain snapshot (`Hovered`, `Clicked`,
  caret position, …). You branch and paint; you do not implement an interface.
- **Defaults are thin.** `Button` is process + a default face. `TextInput` is
  process + plain text/caret draw + a little default chrome. You can drop to the
  same building blocks anytime.

### The one rule that bites

`ModAttrs` must run **before** any child is added (labels, icons, draw
helpers). Process helpers are written so they **do not create children**, so
this stays legal:

```go
st := ProcessButtonEvents(false)
if st.Hovered {
    ModAttrs(Background(...)) // OK — no children yet
}
Icon(...) // children only after attrs are settled
```

---

## 2. Warm-up: a flat custom button

Buttons are the easiest place to learn the model. Interaction is just
pointer press/release; you draw everything yourself.

### What the library provides

```go
st := ProcessButtonEvents(disabled bool) ButtonState
// st.Hovered, st.Active, st.Clicked, st.Disabled, st.Local
```

That is the whole interaction contract for a clickable box. Default
`Button` / `CtrlButton` call the same idea under the hood and then paint an
accent face. You can skip the default face entirely.

### A circular “send” control

In the chat shell we will want a circle with an arrow, not a labeled
rectangle. That is still just process + paint:

```go
func sendCircle(disabled bool) bool {
    const size float32 = 36
    var clicked bool
    accent := Vec4{220, 55, 52, 1}
    if disabled {
        accent = Vec4{0, 0, 78, 1}
    }

    Container(Attrs(FixSize(size, size), Corners(size/2),
        BackgroundVec(accent), Center), func() {
        st := ProcessButtonEvents(disabled)
        clicked = st.Clicked
        if st.Hovered && !disabled {
            ModAttrs(Background(220, 55, 48, 1))
        }
        if st.Active && !disabled {
            ModAttrs(Background(220, 55, 42, 1))
        }
        Icon(TypArrowUp, FontSize(18), TextColor(0, 0, 100, 1))
    })
    return clicked
}
```

**What to notice:**

- The **circle is your container** — size, corners, fill are presentation.
- **Process** only answers “what is the pointer doing to *this* box?”
- Disable by passing `true` when there is nothing to send; process will not
  report a click.

Optional gallery of other faces (Material flat, XP Luna, Win98 bevel):
[`demos/custom-buttons/`](../demos/custom-buttons/).

### On the chat shell (still a default field)

Drop that circle next to the default `TextInputExt` from layout step 14 — you
do **not** need a custom field yet. Intermediate sample:

[`demos/layout-shell/step15a/main.go`](../demos/layout-shell/step15a/main.go)

![Step 15a — custom send, default field](layout-tutorial/images/step15a.png)

Compare to default send on the same shell:

![Step 14 — default field and Send button](layout-tutorial/images/step14.png)

**What to notice:** Only the control you care about changed. The list, rails,
and text field are still default. That is the process/paint model working at
call-site scale.

---

## 3. Text fields: process *and* plain paint

Text is harder than buttons: keys, selection, scroll, IME, caret blink. We
still separate **process** from **chrome**, but we also ship **plain paint**
helpers so you do not reimplement a text editor.

### Two layers

| Function | Responsibility |
|----------|----------------|
| `ProcessTextInput(buf, cfg)` | Focus, hooks, edit model, clipboard, IME rules. **No children.** Returns a snapshot. |
| `DrawTextInputPlain(st, cfg)` | Scrollable text, selection, composition marks, blinking caret. |

You own the **field box** (border, background, padding of the control as a
product). We own **editing** and a **default way to draw the text and caret**
— but you choose **where** to call the draw helpers (inside your box, after
your chrome attrs).

```go
Container(Attrs(Focusable, Clip, PadVec(cfg.Padding), /* your chrome */), func() {
    st := ProcessTextInput(buf, cfg)
    if st.HasFocus {
        ModAttrs(/* focus chrome — still before children */)
    }
    DrawTextInputPlain(st, cfg) // text + cursor, wherever you placed this call
})
```

### Why paint helpers for text but not for buttons?

A button face is a few rectangles and a label — easy to reinvent. A correct
field is not. Requiring every app to redraw caret affinity, composition
underlines, and scroll-to-caret would fight the “customize chrome” goal. So:

- **Buttons:** process only; presentation is entirely yours.  
- **Text:** process + optional plain draw; presentation of the **chrome** is
  yours; presentation of **glyphs and caret** can be the library’s.

You can still style text/caret colors via `TextInputConfig` when needed. You
are free to call `DrawTextInputContent` / `DrawTextInputCaret` separately if
you want them in different places (advanced).

### Contract worth remembering

- Call `ProcessTextInput` **inside** the focusable field container.  
- That container’s **padding** is the text geometry padding (caret and
  hit-testing).  
- Process creates **no children** (so focus `ModAttrs` stays legal).  

Gallery of field skins only: [`demos/custom-textinputs/`](../demos/custom-textinputs/).

You will see a borderless field **in context** in the next section (inside
the compose pill). Until then, the gallery demos are the best place to try
field chrome alone without the whole chat shell.

---

## 4. Put them together: the compose bar

Layout step 14 ends with a practical but plain strip:

```go
Container(Attrs(Expand, Pad(10), Gap(8), Row, CrossMid, Background(220, 6, 95, 1)), func() {
    a := DefaultTextInputAttrs()
    a.NoAutoFocus = true
    TextInputExt(&draft, a)
    if Button(NoIcon, "Send") && draft != "" {
        draft = ""
    }
})
```

A default field next to a default button works. Modern chat-style UIs usually
want **one chrome surface** that houses both: generous outer padding, a
rounded “pill,” a quiet multi-line field, and a compact send control.

```text
[  pad  ]
[  rounded pill:  [ multi-line field .............. ]  (↑)  ]
[  pad  ]
```

That is just **§2 + §3 in one row** — not a new API.

### Step A — Helper on the shell

Keep the layout shell from step 14; only replace the compose strip:

```diff
-				Container(Attrs(Expand, Pad(10), Gap(8), Row, CrossMid, Background(...)), func() {
-					TextInputExt(&draft, ...)
-					if Button(NoIcon, "Send") && draft != "" { draft = "" }
-				})
+				chatCompose(&draft, &messages, textPrim)
```

### Step B — Outer pad + pill (chrome only)

```go
func chatCompose(draft *string, messages *[]msg, textPrim float32) {
    Container(Attrs(Expand, Pad(12), Background(...)), func() {
        Container(Attrs(Expand, Row, CrossMid, Gap(8),
            Pad2(6, 8), Corners(12),
            Background(0, 0, 100, 1),
            BorderWidth(1), BorderColor(0, 0, 0, 0.08),
        ), func() {
            // field (step C) + send circle (step D)
        })
    })
}
```

The **pill** is product chrome. The field inside will stay visually quiet.

### Step C — Field inside the pill

```go
cfg := TextInputConfig{
    FontSize: DefaultTextSize,
    Padding:  N4(10),
    Wrap: true, MaxLines: 0, Rows: 2,
    NoAutoFocus: true,
    TextColor:   Vec4{0, 0, textPrim, 1},
}
// boxH from rows + padding...

Container(Attrs(
    Focusable, Clip, Grow(1),
    PadVec(cfg.Padding),
    MinSize(80, boxH), MaxSizeVec(Vec2{0, boxH}),
    Background(0, 0, 100, 0), // transparent — pill is the chrome
), func() {
    st := ProcessTextInput(draft, cfg)
    if st.HasFocus {
        ModAttrs(Background(220, 10, 98, 1))
    }
    DrawTextInputPlain(st, cfg)
})
```

**What to notice:** Default `TextInputExt` would draw its own border and
underline. Here the pill already frames the control, so the field is
transparent and only process + plain text/caret remain.

### Step D — Send circle in the same pill

Reuse the §2 idea next to the field (`Grow(1)` on the field leaves a fixed
circle on the right):

```go
canSend := *draft != ""
Container(Attrs(FixSize(36, 36), Corners(18),
    BackgroundVec(accentOrGray), Center), func() {
    bst := ProcessButtonEvents(!canSend)
    // hover / press ModAttrs, then Icon...
    if bst.Clicked && canSend {
        text := *draft
        *draft = ""
        *messages = append(*messages, msg{
            id: len(*messages) + 1, author: "you",
            body: text, time: time.Now().Format("15:04"),
        })
    }
})
```

Messages already use `VirtualListView` with stable ids from layout step 14 —
append is enough; no layout rewrite.

### Full source

[`demos/layout-shell/step15/main.go`](../demos/layout-shell/step15/main.go)

```bash
cd shirei
go run ./demos/layout-shell/step15
```

![Step 15 — full custom compose](layout-tutorial/images/step15.png)

**What to notice:** One pill holds both process helpers. The shell above the
strip is still the light layout from step 14; only compose product chrome
changed relative to 15a (default field → borderless field + shared pill).

---

## 5. Dark shell (and retinting the default scrollbar)

Step 15 left you with a working chat **product control** (compose) on a
**light** shell. The package default scrollbar is already a **modern
overlay** (transparent track, thin rounded neutral-gray thumb, darker on
hover/drag). On a dark product UI that mid-gray pill can look too quiet — and
`VirtualList` draws a bar for you, so you cannot “just forget” it.

This section does two finishing moves on the same shell:

1. Switch the palette to a dark chat look.  
2. Optionally retint the **app-wide** scrollbar once so every list picks up a
   light thumb that reads on dark panels.

Full sample:
[`demos/layout-shell/step16/main.go`](../demos/layout-shell/step16/main.go)

```bash
cd shirei
go run ./demos/layout-shell/step16
```

![Step 16 — dark shell](layout-tutorial/images/step16.png)

### Package default is already modern

You do **not** need to implement an overlay bar to get the modern look.
`ScrollBars()`, `VirtualListView`, and menus use `DefaultScrollBarStyle`
unless you override it:

| Idle | Mid-gray translucent pill |
|------|---------------------------|
| Hover | Slightly darker / more opaque |
| Drag | Darker still (still neutral — no accent) |
| Track | Transparent (content shows through) |

`SetDefaultScrollBar(nil)` restores that package style.

Themed skins (classic white-track, Win98, cool blue, …) live in
[`demos/custom-scrollbars/`](../demos/custom-scrollbars/).

### Why a global scrollbar setting?

Buttons and text fields are things **you call yourself**:

```go
if Button(NoIcon, "Send") { ... }
TextInputExt(&draft, attrs)
```

When you want a different look, you either pass attrs at that call site or
build a custom control with `Process…` (as in §§2–4). That is natural —
the call site *is* the product decision.

Scrollbars are different. Most of the time **you never call them**. Other
widgets do:

- `VirtualListView` paints a bar for long lists  
- Menus and similar chrome may call `ScrollBars()`  
- Your own `ScrollOnInput` panes call `ScrollBars()` when you remember  

So retinting bars for a dark shell is **app chrome**, not a parameter on
every list:

```go
// once, typically in main before app.Run:
SetDefaultScrollBar(darkShellScrollBar)

// every default path uses it:
ScrollBars()                    // your panes
// VirtualListView → ScrollBars() internally
```

**What this is not:** a full theming system for every widget. Buttons still
use package defaults unless you process/paint them (or pass accents). Scrollbars
get a global hook because they are **nested chrome** shared by many widgets.

### Light thumb for dark panels

You do **not** reimplement drag math. You call `ScrollBarExt` with chrome
options and an optional thumb painter — same geometry as the package
default, different face:

```go
func darkShellScrollBar() ContainerId {
    return ScrollBarExt(ScrollBarAttrs{
        TrackBG:        Vec4{}, // transparent track
        ThumbMinHeight: 24,
        Thumb: func(size Vec2) {
            r := size[0] / 2
            if r < 1 {
                r = 1
            }
            Element(Attrs(
                FixSizeVec(size),
                Corners(r),
                Background(0, 0, 100, 0.28), // light pill on dark panels
            ))
        },
    })
}
```

`ScrollBarExt` always draws with **no layout animation** (track and thumb snap
with the scroll offset).

| You control | How |
|-------------|-----|
| Track width | `TrackWidth` (zero: `SCROLLBAR_WIDTH` hit target) |
| Track fill | `TrackBG` (zero = transparent) |
| Inner pad | `TrackPad` |
| Shortest thumb | `ThumbMinHeight` |
| Thumb face paint | `Thumb` callback — **size only**; no interaction |

| The framework keeps | |
|---------------------|--|
| When the bar is needed | Content taller than the viewport |
| Thumb length / position from scroll | Geometry from layout |
| Click track to jump | Inside `ScrollBarExt` |
| Drag thumb | Inside `ScrollBarExt` |
| Wheel / `ScrollOnInput` | Separate; still your pane |

### Registering the default

```go
func main() {
    SetDefaultScrollBar(darkShellScrollBar)
    app.SetupWindow("…", winW, winH)
    app.Run(frame)
}
```

Do this **once** at startup, not every frame. After that, step 15’s
`VirtualListView` for messages and members automatically shows the light
thumb — no new VL parameters.

If one pane must differ, call `ScrollBarExt(…)` (or another `ScrollBarFn`)
**at that site** instead of `ScrollBars()`. The global default is only what
`ScrollBars()` uses.

### Dark palette (same structure, new constants)

Layout step 13 already taught “swap loud debug colors for a calm palette.”
Dark is the same idea with different HSLA numbers:

```go
const (
    bgApp, bgSide, bgMain float32 = 16, 18, 20
    textPrim, textMuted   float32 = 92, 62
    borderA               float32 = 0.22 // light lines need more alpha on dark
)
ModAttrs(Background(220, 10, bgApp, 1))
// … Background(220, 10, bgSide, 1) on rails, TextColor(0, 0, textPrim, 1), …
```

Compose (§4) keeps the same process/paint structure; only fills and borders
move to darker values so the pill reads as raised chrome on the strip.

### Section titles: center in the header bar

In a **column** parent, `CrossMid` only centers **horizontally**. A label in
a fixed-height header still sits on the **top** of that bar unless you also
center on the main (vertical) axis.

```go
// Stuck to the top edge of the 44px bar:
Container(Attrs(Expand, FixHeight(44), Pad2(0, 12), CrossMid), ...)

// Vertically and horizontally centered:
Container(Attrs(Expand, FixHeight(44), Pad2(0, 12), Center), ...)
// Center == MainAlign(AlignMiddle) + CrossAlign(AlignMiddle)
```

Step 16 uses a small `sectionTitle` helper so “Channels”, “# general”, and
“Online” share that centering.

### What changed vs step 15

| | Step 15 | Step 16 |
|--|---------|---------|
| Palette | Light greys | Dark surfaces, light text |
| Scrollbar | Package modern (gray) | `SetDefaultScrollBar` light thumb for dark |
| Compose | Custom process/paint | Same structure, dark colors |
| Headers | `CrossMid` only | `Center` (vertical + horizontal) |
| Shell / VL / compose layout | Unchanged | Unchanged |

You did not re-learn Extrinsic or VirtualList. You learned **where product
chrome lives**: call-site for controls you invent; package default for bars
nested widgets draw for you — override only when the product palette needs it.

---

## Recap

| Stage | You learn |
|-------|-----------|
| Philosophy | You own containers; process returns data; you present |
| Flat button | `ProcessButtonEvents` + your paint |
| Text field | `ProcessTextInput` + `DrawTextInputPlain` (chrome yours; glyphs/caret optional helpers) |
| Compose | Same two pieces in one chrome box on the chat shell |
| Dark shell | Palette polish; optional light scrollbar tint via `SetDefaultScrollBar` |

| Default convenience | Building blocks |
|-------------------|-----------------|
| `Button` | `ProcessButtonEvents` + paint |
| `TextInputExt` | field container + `ProcessTextInput` + `DrawTextInputPlain` (+ optional chrome) |
| `ScrollBars()` (modern overlay) | `ScrollBarExt` + optional `SetDefaultScrollBar` |

Layout (Extrinsic / Viewport / VirtualList) did not need to change for
compose. For scrollbars on a dark shell, you also did not change VirtualList —
you changed what `ScrollBars()` means for the process.

### Final results (same images as above)

Light shell, full custom compose (package modern scrollbar):

![Step 15](layout-tutorial/images/step15.png)

Dark shell, light scrollbar tint:

![Step 16](layout-tutorial/images/step16.png)

```bash
cd shirei
go run ./demos/layout-shell/step15
go run ./demos/layout-shell/step16
```

---

## Common mistakes

- `ModAttrs` **after** `Icon` / `DrawTextInputPlain` (panic).  
- Default field chrome **and** pill chrome (double borders).  
- Calling `Button(NoIcon, "Send")` and considering the control “custom.”  
- Changing Extrinsic/Viewport while redesigning compose — usually unnecessary.  
- Calling `ProcessTextInput` outside the focusable field container (hooks and
  focus attach to the wrong node).  
- Expecting `CrossMid` alone to vertically center a header label in a column.  
- Reimplementing thumb drag — use `ScrollBarExt` / the package default instead.  
- Building a “modern” bar from scratch when `DefaultScrollBarStyle` already is one.  
- Passing scrollbar attrs into every `VirtualListView` when a single
  `SetDefaultScrollBar` would do for a product-wide tint.

---

## Related

| | |
|--|--|
| Layout shell (01–14) | [layout-tutorial.md](layout-tutorial.md) |
| Custom send only (step 15a) | [`demos/layout-shell/step15a/`](../demos/layout-shell/step15a/) |
| Compose sample (step 15) | [`demos/layout-shell/step15/`](../demos/layout-shell/step15/) |
| Dark shell (step 16) | [`demos/layout-shell/step16/`](../demos/layout-shell/step16/) |
| Scrollbar skins gallery | [`demos/custom-scrollbars/`](../demos/custom-scrollbars/) |
| Button skins gallery | [`demos/custom-buttons/`](../demos/custom-buttons/) |
| Field skins gallery | [`demos/custom-textinputs/`](../demos/custom-textinputs/) |

