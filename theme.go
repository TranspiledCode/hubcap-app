// theme.go — colour palette definitions for hubcap's named themes.
//
// A Palette holds every semantic colour role used across the UI. Seven named
// palettes are provided:
//
//	"default"     — the original dark terminal look (amber + green)
//	"dracula"     — Dracula colour scheme (purple accent, vibrant)
//	"nord"        — Nord colour scheme (cool blue-grey tones)
//	"catppuccin"  — Catppuccin Mocha (warm dark pastels)
//	"transpiled"  — Transpiled brand colours (electric blue + violet + neon green)
//	"cobalt2"     — West Bostis Cobalt 2 (deep blue + mint + yellow + hot pink)
//	"imagescoop"  — ImageScoop brand (periwinkle + purple + lime + hot pink)
//
// Switch themes via the config form (,) or the t key in any view.
package main

import "github.com/charmbracelet/lipgloss"

// Palette groups every semantic colour role used in the UI.
type Palette struct {
	// Footer key-button roles
	Action lipgloss.Color // primary action keys  (green family)
	Meta   lipgloss.Color // structural / meta keys (amber / yellow family)
	Danger lipgloss.Color // destructive keys  (red family)

	// Accent — list cursor bar, active tab, spinner, filter "on" values
	Accent lipgloss.Color

	// Title text — app title, issue/PR title in detail strip
	Title lipgloss.Color

	// Issue/PR numbers in lists and detail strips
	Number lipgloss.Color

	// Text hierarchy
	Text      lipgloss.Color // primary body text
	TextBold  lipgloss.Color // selected-item title, highlighted values
	TextMuted lipgloss.Color // author names, secondary labels
	TextDim   lipgloss.Color // timestamps, inactive meta
	TextFaint lipgloss.Color // separator lines, very dim rules

	// Backgrounds
	BgHeader   lipgloss.Color // top title/version bar
	BgTabs     lipgloss.Color // tab bar row
	BgSelected lipgloss.Color // selected list-item highlight
	BgFooter   lipgloss.Color // footer bar (minimal/default themes)
	BgComfy    lipgloss.Color // footer bar (comfortable theme)

	// State indicators  (open / closed / merged / draft)
	StatusOpen   lipgloss.Color
	StatusClosed lipgloss.Color
	StatusMerged lipgloss.Color
	StatusDraft  lipgloss.Color

	// CI / review check states
	CheckPass    lipgloss.Color
	CheckFail    lipgloss.Color
	CheckPending lipgloss.Color
}

// ── Named colour-theme constants ──────────────────────────────────────────────

const (
	ColorThemeDefault    = "default"
	ColorThemeDracula    = "dracula"
	ColorThemeNord       = "nord"
	ColorThemeCatppuccin = "catppuccin"
	ColorThemeTranspiled = "transpiled"
	ColorThemeCobalt2    = "cobalt2"
	ColorThemeImageScoop = "imagescoop"
)

// colorThemeOrder is the cycle order used by the ThemeCycle key.
var colorThemeOrder = []string{
	ColorThemeDefault,
	ColorThemeDracula,
	ColorThemeNord,
	ColorThemeCatppuccin,
	ColorThemeTranspiled,
	ColorThemeCobalt2,
	ColorThemeImageScoop,
}

// ── Palette definitions ───────────────────────────────────────────────────────

var paletteDefault = Palette{
	Action: lipgloss.Color("83"),
	Meta:   lipgloss.Color("208"),
	Danger: lipgloss.Color("196"),
	Accent: lipgloss.Color("86"),
	Title:  lipgloss.Color("208"),
	Number: lipgloss.Color("69"),

	Text:      lipgloss.Color("252"),
	TextBold:  lipgloss.Color("255"),
	TextMuted: lipgloss.Color("244"),
	TextDim:   lipgloss.Color("240"),
	TextFaint: lipgloss.Color("238"),

	BgHeader:   lipgloss.Color("235"),
	BgTabs:     lipgloss.Color("236"),
	BgSelected: lipgloss.Color("235"),
	BgFooter:   lipgloss.Color("235"),
	BgComfy:    lipgloss.Color("234"),

	StatusOpen:   lipgloss.Color("83"),
	StatusClosed: lipgloss.Color("196"),
	StatusMerged: lipgloss.Color("141"),
	StatusDraft:  lipgloss.Color("214"),

	CheckPass:    lipgloss.Color("83"),
	CheckFail:    lipgloss.Color("196"),
	CheckPending: lipgloss.Color("214"),
}

var paletteDracula = Palette{
	Action: lipgloss.Color("#50fa7b"),
	Meta:   lipgloss.Color("#ffb86c"),
	Danger: lipgloss.Color("#ff5555"),
	Accent: lipgloss.Color("#bd93f9"),
	Title:  lipgloss.Color("#ff79c6"),
	Number: lipgloss.Color("#8be9fd"),

	Text:      lipgloss.Color("#f8f8f2"),
	TextBold:  lipgloss.Color("#ffffff"),
	TextMuted: lipgloss.Color("#6272a4"),
	TextDim:   lipgloss.Color("#44475a"),
	TextFaint: lipgloss.Color("#383a59"),

	BgHeader:   lipgloss.Color("#282a36"),
	BgTabs:     lipgloss.Color("#21222c"),
	BgSelected: lipgloss.Color("#44475a"),
	BgFooter:   lipgloss.Color("#282a36"),
	BgComfy:    lipgloss.Color("#21222c"),

	StatusOpen:   lipgloss.Color("#50fa7b"),
	StatusClosed: lipgloss.Color("#ff5555"),
	StatusMerged: lipgloss.Color("#bd93f9"),
	StatusDraft:  lipgloss.Color("#ffb86c"),

	CheckPass:    lipgloss.Color("#50fa7b"),
	CheckFail:    lipgloss.Color("#ff5555"),
	CheckPending: lipgloss.Color("#f1fa8c"),
}

var paletteNord = Palette{
	Action: lipgloss.Color("#a3be8c"),
	Meta:   lipgloss.Color("#ebcb8b"),
	Danger: lipgloss.Color("#bf616a"),
	Accent: lipgloss.Color("#88c0d0"),
	Title:  lipgloss.Color("#81a1c1"),
	Number: lipgloss.Color("#88c0d0"),

	Text:      lipgloss.Color("#d8dee9"),
	TextBold:  lipgloss.Color("#eceff4"),
	TextMuted: lipgloss.Color("#81a1c1"),
	TextDim:   lipgloss.Color("#616e88"),
	TextFaint: lipgloss.Color("#434c5e"),

	BgHeader:   lipgloss.Color("#2e3440"),
	BgTabs:     lipgloss.Color("#3b4252"),
	BgSelected: lipgloss.Color("#3b4252"),
	BgFooter:   lipgloss.Color("#2e3440"),
	BgComfy:    lipgloss.Color("#242933"),

	StatusOpen:   lipgloss.Color("#a3be8c"),
	StatusClosed: lipgloss.Color("#bf616a"),
	StatusMerged: lipgloss.Color("#b48ead"),
	StatusDraft:  lipgloss.Color("#d08770"),

	CheckPass:    lipgloss.Color("#a3be8c"),
	CheckFail:    lipgloss.Color("#bf616a"),
	CheckPending: lipgloss.Color("#ebcb8b"),
}

var paletteCatppuccin = Palette{
	Action: lipgloss.Color("#a6e3a1"),
	Meta:   lipgloss.Color("#fab387"),
	Danger: lipgloss.Color("#f38ba8"),
	Accent: lipgloss.Color("#cba6f7"),
	Title:  lipgloss.Color("#cba6f7"),
	Number: lipgloss.Color("#89b4fa"),

	Text:      lipgloss.Color("#cdd6f4"),
	TextBold:  lipgloss.Color("#ffffff"),
	TextMuted: lipgloss.Color("#a6adc8"),
	TextDim:   lipgloss.Color("#7f849c"),
	TextFaint: lipgloss.Color("#585b70"),

	BgHeader:   lipgloss.Color("#1e1e2e"),
	BgTabs:     lipgloss.Color("#313244"),
	BgSelected: lipgloss.Color("#313244"),
	BgFooter:   lipgloss.Color("#1e1e2e"),
	BgComfy:    lipgloss.Color("#181825"),

	StatusOpen:   lipgloss.Color("#a6e3a1"),
	StatusClosed: lipgloss.Color("#f38ba8"),
	StatusMerged: lipgloss.Color("#cba6f7"),
	StatusDraft:  lipgloss.Color("#fab387"),

	CheckPass:    lipgloss.Color("#a6e3a1"),
	CheckFail:    lipgloss.Color("#f38ba8"),
	CheckPending: lipgloss.Color("#f9e2af"),
}

// paletteCobalt2 is based on the West Bostis Cobalt 2 editor theme.
// Backgrounds are the deep navy blues from the theme; accents use the
// iconic mint (#2AFFDF), yellow (#FFC600), hot pink (#FF0088), and
// neon green (#3AD900).
var paletteCobalt2 = Palette{
	Action: lipgloss.Color("#3AD900"), // neon green — "do it" actions
	Meta:   lipgloss.Color("#FF9D00"), // orange accent — structural / nav
	Danger: lipgloss.Color("#FF0088"), // hot pink — destructive

	Accent: lipgloss.Color("#2AFFDF"), // mint — cursor bar, spinner
	Title:  lipgloss.Color("#FFC600"), // yellow accent — titles, app border
	Number: lipgloss.Color("#9EFFFF"), // light cyan — issue/PR numbers

	Text:      lipgloss.Color("#cce7f0"), // derived soft blue-white — body text
	TextBold:  lipgloss.Color("#9EFFFF"), // light cyan — selected / highlighted
	TextMuted: lipgloss.Color("#5a9fb8"), // medium steel blue — authors, secondary
	TextDim:   lipgloss.Color("#3a6d82"), // dim blue — timestamps
	TextFaint: lipgloss.Color("#234E6D"), // Highlight Background 2 — separators

	BgHeader:   lipgloss.Color("#122738"), // Darker Blue
	BgTabs:     lipgloss.Color("#15232D"), // Dark Background
	BgSelected: lipgloss.Color("#1F4662"), // Highlight Background
	BgFooter:   lipgloss.Color("#122738"),
	BgComfy:    lipgloss.Color("#0D3A58"), // Off Blue

	StatusOpen:   lipgloss.Color("#3AD900"), // green — open
	StatusClosed: lipgloss.Color("#FF628C"), // blush pink — closed
	StatusMerged: lipgloss.Color("#2AFFDF"), // mint — merged
	StatusDraft:  lipgloss.Color("#FF9D00"), // orange — draft

	CheckPass:    lipgloss.Color("#3AD900"),
	CheckFail:    lipgloss.Color("#FF0088"),
	CheckPending: lipgloss.Color("#FFC600"), // yellow — pending
}

// paletteTranspiled uses Transpiled's actual brand colours extracted from
// transpiled.com: electric blue (#0098E4), royal blue (#123EDB), vivid
// violet (#D05FEC), and neon green (#3CEE39) on a near-black background.
var paletteTranspiled = Palette{
	Action: lipgloss.Color("#3CEE39"), // neon green — "do it" actions
	Meta:   lipgloss.Color("#D05FEC"), // vivid violet — structural / nav
	Danger: lipgloss.Color("#F34D2C"), // orange-red — destructive

	Accent: lipgloss.Color("#0098E4"), // electric blue — cursor bar, spinner
	Title:  lipgloss.Color("#D05FEC"), // violet — titles, app border
	Number: lipgloss.Color("#7faaf0"), // cornflower — issue/PR numbers

	Text:      lipgloss.Color("#B7B8BA"), // mid grey — body text
	TextBold:  lipgloss.Color("#FAF9F8"), // off-white — selected / highlighted
	TextMuted: lipgloss.Color("#6E7275"), // dim grey — authors, secondary
	TextDim:   lipgloss.Color("#404347"), // darker grey — timestamps
	TextFaint: lipgloss.Color("#2a2c2e"), // near-invisible — separators

	BgHeader:   lipgloss.Color("#141515"), // brand near-black
	BgTabs:     lipgloss.Color("#1a1b1d"), // slightly lighter
	BgSelected: lipgloss.Color("#1e2030"), // subtle blue-black tint
	BgFooter:   lipgloss.Color("#141515"),
	BgComfy:    lipgloss.Color("#0f1010"),

	StatusOpen:   lipgloss.Color("#3CEE39"), // neon green — open
	StatusClosed: lipgloss.Color("#F34D2C"), // orange-red — closed
	StatusMerged: lipgloss.Color("#D05FEC"), // violet — merged (brand)
	StatusDraft:  lipgloss.Color("#ED951A"), // orange — draft

	CheckPass:    lipgloss.Color("#3CEE39"),
	CheckFail:    lipgloss.Color("#F34D2C"),
	CheckPending: lipgloss.Color("#E7C61E"), // yellow — pending
}

// paletteImageScoop is based on colours extracted from imagescoop.app.
// The brand signature is a periwinkle→purple gradient (#667eea / #764ba2)
// with lime green (#a3e635), hot pink (#ec4899), and orange (#f97316)
// as accent colours, on Tailwind slate dark backgrounds.
var paletteImageScoop = Palette{
	Action: lipgloss.Color("#a3e635"), // lime green — "do it" actions
	Meta:   lipgloss.Color("#667eea"), // periwinkle — structural / nav
	Danger: lipgloss.Color("#ec4899"), // hot pink — destructive

	Accent: lipgloss.Color("#667eea"), // periwinkle — cursor bar, spinner
	Title:  lipgloss.Color("#764ba2"), // purple — titles, app border
	Number: lipgloss.Color("#7BC4D4"), // light teal — issue/PR numbers

	Text:      lipgloss.Color("#E5E7EB"), // gray-200 — body text
	TextBold:  lipgloss.Color("#FAFAFA"), // near-white — selected / highlighted
	TextMuted: lipgloss.Color("#9CA3AF"), // gray-400 — authors, secondary
	TextDim:   lipgloss.Color("#6B7280"), // gray-500 — timestamps
	TextFaint: lipgloss.Color("#4B5563"), // gray-600 — separators

	BgHeader:   lipgloss.Color("#0F172A"), // slate-900 — deepest
	BgTabs:     lipgloss.Color("#1F2937"), // gray-800
	BgSelected: lipgloss.Color("#283040"), // dark blue-grey selection
	BgFooter:   lipgloss.Color("#0F172A"),
	BgComfy:    lipgloss.Color("#111827"), // gray-900

	StatusOpen:   lipgloss.Color("#a3e635"), // lime — open
	StatusClosed: lipgloss.Color("#ec4899"), // hot pink — closed
	StatusMerged: lipgloss.Color("#667eea"), // periwinkle — merged
	StatusDraft:  lipgloss.Color("#f97316"), // orange — draft

	CheckPass:    lipgloss.Color("#a3e635"),
	CheckFail:    lipgloss.Color("#ec4899"),
	CheckPending: lipgloss.Color("#f97316"), // orange — pending
}

// resolvePalette maps a colour-theme name to its Palette, falling back to
// paletteDefault for unknown / empty values.
func resolvePalette(s string) Palette {
	switch s {
	case ColorThemeDracula:
		return paletteDracula
	case ColorThemeNord:
		return paletteNord
	case ColorThemeCatppuccin:
		return paletteCatppuccin
	case ColorThemeTranspiled:
		return paletteTranspiled
	case ColorThemeCobalt2:
		return paletteCobalt2
	case ColorThemeImageScoop:
		return paletteImageScoop
	default:
		return paletteDefault
	}
}

// nextColorTheme returns the theme name that follows current in the cycle order.
func nextColorTheme(current string) string {
	for i, t := range colorThemeOrder {
		if t == current {
			return colorThemeOrder[(i+1)%len(colorThemeOrder)]
		}
	}
	return ColorThemeDefault
}
