# Hubcap — Marketing Page Design Concept

**Date:** 2026-06-10
**Status:** Draft — for design team handoff
**Product:** hubcap v0.7.0
**Tagline:** GitHub, in your terminal.

---

## Overview

This document describes the design concept for a standalone marketing/landing page for **hubcap** — a keyboard-driven terminal UI for GitHub. The goal of the page is to communicate the product's value proposition quickly, earn credibility with a technical audience, and drive installs.

The audience is **terminal-first developers** — people who live in their editor and shell, are allergic to unnecessary browser tabs, and immediately respect tools that respect their workflow. The page should feel like it was built *by* that kind of developer *for* that kind of developer. It should not feel like a SaaS marketing site. It should feel sharp, dark, minimal, and confident.

---

## Aesthetic Direction

### Tone

**Confident, dry, developer-native.** No buzzwords. No "supercharge your workflow" fluff. Copy is terse and specific. The product does what it says. The page shows it doing it.

Think: how a well-maintained GitHub README feels when someone really put care into it — but as a full marketing page.

### Visual Language

- **Dark background** — the product lives in a terminal. The page should feel at home there. Deep near-black (#0d1117 or similar — GitHub dark territory, or the Transpiled near-black #0f1010) as the base.
- **Terminal aesthetic, refined** — monospace typefaces for code, commands, and callouts. The product's own color palette (amber, green, electric blue) bleeds into the page design as accent colors. Not garish — selective and deliberate.
- **Screenshots as the hero** — the actual UI is the most compelling thing to show. Real terminal screenshots (not illustrations or fake mockups) should be front and center. The UI looks good. Show it.
- **Minimal chrome** — no gradients everywhere, no stock photography, no hero illustrations. Let the terminal screenshots breathe against the dark background.
- **Precision over decoration** — borders, separators, and type hierarchy carry the design. Every element earns its place.

### Typography

- **Headline:** A bold geometric sans-serif (e.g., Inter, DM Sans, or similar developer-community-adjacent typeface). Large, confident, tight leading.
- **Body:** The same sans-serif, regular weight, generous line-height, medium size. Easy to read at pace.
- **Code / terminal elements:** Monospace (JetBrains Mono, Fira Code, or Berkeley Mono). Used for install commands, key bindings, and any terminal-flavored callouts.

### Color Palette

Pulled directly from hubcap's own default/Transpiled themes to create visual continuity between the product and its marketing page:

| Role | Value | Use |
|---|---|---|
| Background | `#0f1010` | Page background |
| Surface | `#141515` | Card/section backgrounds |
| Surface raised | `#1e2030` | Elevated panels |
| Accent (electric blue) | `#0098E4` | Links, highlights, active indicators |
| Action (neon green) | `#3CEE39` | CTAs, key action callouts |
| Meta (violet) | `#D05FEC` | Secondary accents, section headings |
| Danger / Pop | `#F34D2C` | Occasional contrast punches |
| Body text | `#B7B8BA` | Paragraph copy |
| Muted text | `#6E7275` | Labels, captions, secondary |
| Border | `#2a2c2e` | Dividers, card borders |

---

## Page Structure

### 1. Nav Bar (minimal)

- Left: hubcap logo (the hub cap icon) + wordmark "hubcap"
- Right: GitHub star count badge (live), "Install" anchor link, GitHub repo link
- Sticky, transparent on scroll start, slight surface blur when scrolled
- No hamburger menu. No login. No pricing. It's a single-page product.

---

### 2. Hero

**The most important section. Sets everything.**

**Headline (large, centered):**
> GitHub, in your terminal.

**Subhead (one line, restrained):**
> Browse issues, review PRs, check CI status, and take action — all with keyboard shortcuts. No browser tab needed.

**Install command block** (monospace, copyable, prominent):
```
brew tap TranspiledCode/tap
brew install hubcap
```
A single click-to-copy button beside it. Nothing more. The install path *is* the CTA — if someone reads this and wants it, they should be able to have it in under 30 seconds.

**Hero screenshot:**
Below or alongside the copy, a large, high-quality screenshot of the hubcap UI — ideally the Issues tab with real-looking data populated, showing the color-coded labels, CI indicators, keyboard shortcut footer bar. This is the product's best-looking face. Optionally: a looping video/GIF showing keyboard navigation in action (arrow keys scrolling, opening an issue, pressing `d` to develop a branch). Motion communicates the keyboard-driven UX better than any sentence.

**Visual treatment:** The terminal screenshot sits inside a subtle macOS-style window chrome (traffic lights, title bar) or just a simple rounded dark frame. It glows very slightly — a soft diffused color bloom behind it in electric blue/violet that pulls from the UI's own palette.

---

### 3. The Problem (brief, no header needed)

A short, punchy paragraph or trio of statements — not a named section, just context that flows after the hero. Tone: knowing, not preachy.

Example copy direction:
> *You're in the middle of a coding session. You need to check a PR.*
> *You open a browser. GitHub loads. You lose the thread.*
> *Hubcap keeps you where you work.*

This can be rendered as three short lines with minimal vertical spacing — almost like terminal output. Not a wall of text.

---

### 4. Feature Highlights (three columns)

Three feature cards in a tight horizontal grid, each with:
- A minimal icon (monoline, not filled — keeping it sharp)
- A short bold title
- Two to three sentences of copy

**Card 1 — My Work Dashboard**
*"Your day, on one screen."*
Review requests, your open PRs, and assigned issues — all in one view when you launch. No hunting. No filtering. Just what you need to start working.

**Card 2 — Issues & PRs, fully actionable**
*"Not just read-only."*
Create, assign, label, develop branches, merge — all from the keyboard. CI check status shows pass/fail/pending next to every PR row. The full GitHub workflow without leaving the terminal.

**Card 3 — Instant, filtered, yours**
*"Fast because it has to be."*
Smart caching means the UI appears instantly. Background refresh keeps data current. Filter by state, assignee, label, or milestone. Keyboard shortcuts make navigation feel native.

---

### 5. "How It Works" (three steps, horizontal)

Minimal numbered steps. No diagram needed — the simplicity is the point.

**Step 1 — Authenticate once**
You already have `gh` set up. Hubcap uses that. No new accounts, no tokens to manage.

**Step 2 — Run it anywhere**
`cd` into any GitHub repo and run `hubcap`. It detects the remote automatically.

**Step 3 — Stay in flow**
Navigate with arrow keys. Act with single keystrokes. Never open a browser tab for routine GitHub work again.

---

### 6. Theme Showcase

This section exists to show off something most TUI tools don't have: **beautiful color themes**. It also signals craft and personality.

Layout: A horizontal scroll or grid of 4–6 terminal screenshots, each showing the same UI in a different theme (Default, Cobalt 2, Transpiled, Parchment, Latte, ImageScoop). Each has its theme name as a caption below.

Headline for the section:
> *Looks good wherever you work.*

Short copy:
> Six built-in color themes — including light modes. Cycle with `t`, or set your default in config.

This section doubles as a visual feast and a subtle signal: this tool is polished enough to have a Catppuccin Latte port.

---

### 7. Keyboard Shortcut Reference (collapsible / ambient)

A compact table or styled block showing the primary shortcuts — not the full reference, just the highlights. Rendered in the monospace font, styled to look like it could have come out of hubcap itself (dark surface background, colored key badges matching the product's footer bar style).

| Key | Action |
|---|---|
| `↑ ↓` | Navigate list |
| `Enter` | Open item |
| `Tab` | Switch tab |
| `n` | New issue / PR |
| `f` | Filter |
| `d` | Develop branch |
| `m` | Merge PR |
| `t` | Cycle theme |
| `?` | All shortcuts |

This section serves double duty: it shows off the depth of the keyboard interface, and it answers "but how do I actually use it?" for skeptical visitors.

---

### 8. Install (final CTA, full-width)

A section near the bottom — full-width, high contrast — that repeats the install path cleanly. Dark section with a slightly elevated card treatment.

**Headline:**
> Get started in 30 seconds.

**Requirements block:**
```
Requirements: gh CLI installed and authenticated
```

**Install block:**
```sh
brew tap TranspiledCode/tap
brew install hubcap
```

**Then:**
```sh
cd your-github-repo
hubcap
```

Optional: a "Build from source" toggle/disclosure for non-Homebrew users.

Two secondary links below: GitHub repo, README/docs.

---

### 9. Footer

Ultra minimal:
- Left: hubcap logo + "by TranspiledCode"
- Center: GitHub link, Homebrew tap link, License
- Right: nothing, or a "Built with Go + Charm" credit (earns points in the developer community)

No newsletter. No social links beyond GitHub. No cookie banner theater.

---

## Content & Copy Principles

1. **Terse by default.** Every sentence should be able to defend its existence. If it's padding, cut it.
2. **Show, don't tell.** Screenshots and the install command communicate more than marketing copy ever will. Let them lead.
3. **Respect the audience's intelligence.** Developers can read a feature list and infer the value. Don't over-explain.
4. **No feature lists.** Features are framed as outcomes ("Your day, on one screen") not capabilities ("Supports filtering by assignee").
5. **The install command is the CTA.** There is no "Sign up" or "Get a demo" — just `brew install hubcap`. Make that path frictionless and obvious.

---

## Responsive Considerations

- **Desktop (primary):** Three-column feature cards, hero screenshot full-width below copy, theme showcase as horizontal scroll.
- **Tablet:** Two-column feature cards, hero screenshot scales down, same structure otherwise.
- **Mobile:** Single column throughout. Theme showcase becomes a swipeable carousel. Install commands use a full-width copyable code block. The page is functional but desktop is the real target — the audience uses terminals.

---

## Assets Needed from Product Team

| Asset | Notes |
|---|---|
| High-res UI screenshots (each tab) | Issues, PRs, Dashboard — with realistic data populated |
| Per-theme screenshots | One clean screenshot per theme for the showcase section |
| Demo GIF or video | 15–30 seconds of keyboard navigation — the money shot |
| Final logo files | SVG preferred; current asset is PNG |
| Homebrew tap URL | Confirm current tap path for install command |
| GitHub repo URL | For nav bar, footer links |
| Live GitHub star count | For badge in nav (can be fetched dynamically via GitHub API) |

---

## What This Page Is Not

- Not a docs site. No sidebar nav, no deep reference content.
- Not a SaaS landing page. No pricing, no feature comparison table, no "Book a demo" CTA.
- Not a portfolio piece. The product is the hero, not the brand.
- Not over-designed. The constraint is the point. A TUI tool's marketing page that looks like a Figma playground would undermine the entire message.

---

*End of design concept document.*
