// theme.go — colour palette definitions for hubcap's named themes.
//
// A Palette holds every semantic colour role used across the UI. Five named
// palettes are provided:
//
//	"default"    — the original dark terminal look (amber + green)
//	"transpiled" — Transpiled brand colours (electric blue + violet + neon green)
//	"cobalt2"    — West Bostis Cobalt 2 (deep blue + mint + yellow + hot pink)
//	"imagescoop" — ImageScoop brand (periwinkle + purple + lime + hot pink)
//	"parchment"  — light warm paper with jewel-tone accents (emerald + amethyst + sapphire + ruby)
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
	BgBody     lipgloss.Color // main content area (list rows, empty space)
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

	// Semantic label category colours — used for text in list rows (LabelXxx)
	// and as pill chip backgrounds in detail strips, with LabelXxxFg as pill text.
	LabelDanger  lipgloss.Color // bug, critical, high-priority
	LabelWarn    lipgloss.Color // medium-priority, question
	LabelSuccess lipgloss.Color // low-priority, passing
	LabelFeature lipgloss.Color // enhancement, feature
	LabelDocs    lipgloss.Color // documentation
	LabelSubtle  lipgloss.Color // effort, size (dim)
	LabelDefault lipgloss.Color // uncategorised / fallback

	LabelDangerFg  lipgloss.Color // text on top of LabelDanger pill
	LabelWarnFg    lipgloss.Color // text on top of LabelWarn pill
	LabelSuccessFg lipgloss.Color // text on top of LabelSuccess pill
	LabelFeatureFg lipgloss.Color // text on top of LabelFeature pill
	LabelDocsFg    lipgloss.Color // text on top of LabelDocs pill
	LabelSubtleFg  lipgloss.Color // text on top of LabelSubtle pill
	LabelDefaultFg lipgloss.Color // text on top of LabelDefault pill
}

// ── Named colour-theme constants ──────────────────────────────────────────────

const (
	ColorThemeDefault    = "default"
	ColorThemeTranspiled = "transpiled"
	ColorThemeCobalt2    = "cobalt2"
	ColorThemeImageScoop = "imagescoop"
	ColorThemeParchment  = "parchment"
)

// colorThemeOrder is the cycle order used by the ThemeCycle key.
var colorThemeOrder = []string{
	ColorThemeDefault,
	ColorThemeTranspiled,
	ColorThemeCobalt2,
	ColorThemeImageScoop,
	ColorThemeParchment,
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

	BgBody:     lipgloss.Color(""),
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

	LabelDanger:  lipgloss.Color("1"),
	LabelWarn:    lipgloss.Color("3"),
	LabelSuccess: lipgloss.Color("2"),
	LabelFeature: lipgloss.Color("6"),
	LabelDocs:    lipgloss.Color("5"),
	LabelSubtle:  lipgloss.Color("8"),
	LabelDefault: lipgloss.Color("208"),

	LabelDangerFg:  lipgloss.Color("15"),
	LabelWarnFg:    lipgloss.Color("0"),
	LabelSuccessFg: lipgloss.Color("0"),
	LabelFeatureFg: lipgloss.Color("0"),
	LabelDocsFg:    lipgloss.Color("15"),
	LabelSubtleFg:  lipgloss.Color("15"),
	LabelDefaultFg: lipgloss.Color("0"),
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

	BgBody:     lipgloss.Color(""),
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

	LabelDanger:  lipgloss.Color("#FF628C"),
	LabelWarn:    lipgloss.Color("#FFC600"),
	LabelSuccess: lipgloss.Color("#3AD900"),
	LabelFeature: lipgloss.Color("#2AFFDF"),
	LabelDocs:    lipgloss.Color("#C792EA"),
	LabelSubtle:  lipgloss.Color("#546E7A"),
	LabelDefault: lipgloss.Color("#FF9D00"),

	LabelDangerFg:  lipgloss.Color("0"),
	LabelWarnFg:    lipgloss.Color("0"),
	LabelSuccessFg: lipgloss.Color("0"),
	LabelFeatureFg: lipgloss.Color("0"),
	LabelDocsFg:    lipgloss.Color("0"),
	LabelSubtleFg:  lipgloss.Color("15"),
	LabelDefaultFg: lipgloss.Color("0"),
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

	BgBody:     lipgloss.Color(""),
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

	LabelDanger:  lipgloss.Color("#FF0088"),  // hot pink — Transpiled Danger
	LabelWarn:    lipgloss.Color("#FFC600"),  // yellow — Transpiled CheckPending
	LabelSuccess: lipgloss.Color("#3AD900"),  // neon green — Transpiled Action
	LabelFeature: lipgloss.Color("#2AFFDF"),  // mint — Transpiled Accent
	LabelDocs:    lipgloss.Color("#C792EA"),  // soft violet — distinct docs colour
	LabelSubtle:  lipgloss.Color("#546E7A"),  // blue-grey — dim effort/size
	LabelDefault: lipgloss.Color("#FF9D00"),  // orange — Transpiled Meta

	LabelDangerFg:  lipgloss.Color("0"),
	LabelWarnFg:    lipgloss.Color("0"),
	LabelSuccessFg: lipgloss.Color("0"),
	LabelFeatureFg: lipgloss.Color("0"),
	LabelDocsFg:    lipgloss.Color("0"),
	LabelSubtleFg:  lipgloss.Color("15"),
	LabelDefaultFg: lipgloss.Color("0"),
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

	BgBody:     lipgloss.Color(""),
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

	LabelDanger:  lipgloss.Color("#ec4899"),
	LabelWarn:    lipgloss.Color("#f97316"),
	LabelSuccess: lipgloss.Color("#a3e635"),
	LabelFeature: lipgloss.Color("#06b6d4"),
	LabelDocs:    lipgloss.Color("#a855f7"),
	LabelSubtle:  lipgloss.Color("#6B7280"),
	LabelDefault: lipgloss.Color("#667eea"),

	LabelDangerFg:  lipgloss.Color("15"),
	LabelWarnFg:    lipgloss.Color("0"),
	LabelSuccessFg: lipgloss.Color("0"),
	LabelFeatureFg: lipgloss.Color("0"),
	LabelDocsFg:    lipgloss.Color("15"),
	LabelSubtleFg:  lipgloss.Color("15"),
	LabelDefaultFg: lipgloss.Color("15"),
}

// paletteParchment is a light theme built on warm antique-paper backgrounds.
// Jewel-tone accents — emerald (#1A7A4A), amethyst (#7C3D9B), sapphire
// (#1565C0), and ruby (#B5272B) — pop vividly against the creamy base.
var paletteParchment = Palette{
	Action: lipgloss.Color("#1A7A4A"), // emerald green — "do it" actions
	Meta:   lipgloss.Color("#7C3D9B"), // amethyst violet — structural / nav
	Danger: lipgloss.Color("#B5272B"), // ruby red — destructive

	Accent: lipgloss.Color("#1565C0"), // sapphire blue — cursor bar, spinner
	Title:  lipgloss.Color("#7C3D9B"), // amethyst — titles, app border
	Number: lipgloss.Color("#1565C0"), // sapphire — issue/PR numbers

	Text:      lipgloss.Color("#3C3228"), // warm espresso brown — body text
	TextBold:  lipgloss.Color("#1A1210"), // near-black — selected / highlighted
	TextMuted: lipgloss.Color("#756860"), // medium warm brown — authors, secondary
	TextDim:   lipgloss.Color("#9E8E80"), // lighter warm brown — timestamps
	TextFaint: lipgloss.Color("#C2B4A4"), // faint warm beige — separator lines

	BgBody:     lipgloss.Color("#F2ECD8"), // warm cream — main content area
	BgHeader:   lipgloss.Color("#E0D9C4"), // warm parchment — title bar
	BgTabs:     lipgloss.Color("#E8E1CD"), // slightly lighter — tab row
	BgSelected: lipgloss.Color("#D9D1BC"), // warm tan — selected item
	BgFooter:   lipgloss.Color("#E0D9C4"), // same as header — footer bar
	BgComfy:    lipgloss.Color("#D6CEB8"), // a touch darker — comfortable footer

	StatusOpen:   lipgloss.Color("#1A7A4A"), // emerald — open
	StatusClosed: lipgloss.Color("#B5272B"), // ruby — closed
	StatusMerged: lipgloss.Color("#7C3D9B"), // amethyst — merged
	StatusDraft:  lipgloss.Color("#D45D00"), // burnt sienna — draft

	CheckPass:    lipgloss.Color("#1A7A4A"),
	CheckFail:    lipgloss.Color("#B5272B"),
	CheckPending: lipgloss.Color("#D45D00"),

	LabelDanger:  lipgloss.Color("#B5272B"), // ruby red
	LabelWarn:    lipgloss.Color("#C07800"), // golden amber
	LabelSuccess: lipgloss.Color("#1A7A4A"), // emerald
	LabelFeature: lipgloss.Color("#1565C0"), // sapphire
	LabelDocs:    lipgloss.Color("#7C3D9B"), // amethyst
	LabelSubtle:  lipgloss.Color("#7A7060"), // muted warm grey-brown
	LabelDefault: lipgloss.Color("#5C3D1E"), // warm cocoa

	LabelDangerFg:  lipgloss.Color("15"),
	LabelWarnFg:    lipgloss.Color("15"),
	LabelSuccessFg: lipgloss.Color("15"),
	LabelFeatureFg: lipgloss.Color("15"),
	LabelDocsFg:    lipgloss.Color("15"),
	LabelSubtleFg:  lipgloss.Color("15"),
	LabelDefaultFg: lipgloss.Color("15"),
}

// resolvePalette maps a colour-theme name to its Palette, falling back to
// paletteDefault for unknown / empty values.
func resolvePalette(s string) Palette {
	switch s {
	case ColorThemeTranspiled:
		return paletteTranspiled
	case ColorThemeCobalt2:
		return paletteCobalt2
	case ColorThemeImageScoop:
		return paletteImageScoop
	case ColorThemeParchment:
		return paletteParchment
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
