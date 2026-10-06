# OutletOwl: pushing this bundle to Claude Design

Built by screen-design on 2026-10-06, design bundle v2 (round 2 redesign).
Nothing in this folder has been pushed anywhere. The push is one command
you run yourself.

## What to run

From a Claude Code session started in `docs/design/`:

```
/design-sync
```

It lists your design-system projects, or offers to create one. Pick the
target, read the plan it shows (every file and the folder it comes from),
and approve. Declining changes nothing.

## What is in the bundle

- `tokens.css`: every colour, font, space, radius, shadow and duration, light only; nothing else declares a colour
- `screens/app/`: 9 screen files (S-01 to S-09), each with every state at desktop and 375 px, plus `screens.css`, `index.html` and `build.py` (the source of the pages)
- `index.html`: the gallery, for reading the bundle locally (rendered with `gallery.template.html`)
- `design.json`: the manifest (brief, directions, sitemap, navigation, screens, concerns, audit)

No `design-system.html` yet: `design-system` has not run.

Leave out: `gallery.template.html`, `screens/*/shots/`, `*.console.txt`, `*.theme.json`, `screens/*/CHANGES.md`, `screens/app/screens.json`.

## Before you approve

1. The audit ran on the files being pushed: `design-bundle check: 9 screens (1 features), 56 state panels, 1086 links checked, 0 flows files, 19 stories, 0 audit findings (missing screens 0, missing states 0, missing chrome 0, unannotated states 0, dead links 0, missing nav links 0, external refs 0, hardcoded colours 0, unlinked tokens 0, uncovered stories 0), 0 problems` (rerun `bundle.py check --design docs/design` and compare before pushing).
2. The concerns in `index.html` under "Raised while designing" are answered or accepted; open ones travel with the bundle.
3. The target project is the product's own, not a shared sandbox.

## After it lands

Record the date and the target project in `screens/app/CHANGES.md`.
A later round is pushed the same way: rebuild with `python3 docs/design/screens/app/build.py`, render the gallery with `bundle.py gallery --design docs/design --template docs/design/gallery.template.html`, rerun the audit, then `/design-sync`.
