# Screen design changes: app

Path: docs/design/screens/app/
Inventory: stories (docs/product/backlog.md), PRD and HLD; no flows file   Tokens: docs/design/tokens.css (proposed, round 2)
Framework: web (HTML bundle now; React and Vite views later, ADR-0002)

One section per round, newest last. Every change names the file and
line so the next round and the review can find it. A round with no
comment is still recorded, because three of them end the loop.

The pages are generated: edit `build.py` (and `screens.css`), then run
`python3 docs/design/screens/app/build.py`. A hand edit to an S-*.html page is lost at the next build.

## Round 1: 2026-10-06

First round: no review comments yet. Decisions taken with the product owner before drawing:
visual direction Instrument panel; light and dark both designed; navigation as proposed (admin side nav with six destinations,
phone tabs Overview, Reviews, Themes, Import, More; manager Overview, Reviews, Themes).

Self-review fixes made before showing (found on the screenshots): overview urgent panel left dead space at 1440
(now spans two rows beside movers and the comparison); movers table overflowed at 768 (content grid capped at the frame width);
comparison table clipped its last column (headers shortened, Reviews folded into Replied as 9/21);
sign-in desktop layout collapsed (wrapper kept as a block); dates in mono read as spaced out (body face with tabular figures);
reply label repeated its heading (now names the reviewer); entrance motion disabled in screenshot mode so shots show the settled screen.

Comments received: 0   Changes: 0   Re-shot: all 9 screens (shots/, 54 PNGs)

| # | Screen | Comment (the user's words) | Change | Where |
| --- | --- | --- | --- | --- |
| none | | | | |

Not changed: none
Stop condition: continuing

## Round 2: 2026-10-06

Full visual redesign requested by the product owner: the round 1 look was discarded, and every screen's content,
data, states and flows were kept. Reference: webinar.gg, for qualities only; no logo, text or image copied.
Decisions taken with the product owner before building: dark theme dropped (light only); status keeps red,
amber and green with words; gradient words on light pages use #0B5CFF to #3B82F6 because the requested
#3B82F6 to #BFDBFE measures 1.36:1 at its light end on #F8FAFF; tokens stay in tokens.css, which every page,
the feature index and the gallery link.

Comments received: 1 (the redesign brief)   Changes: 9   Re-shot: all 9 screens (shots/, 54 PNGs; see Not changed)

| # | Screen | Comment (the user's words) | Change | Where |
| --- | --- | --- | --- | --- |
| 1 | all | "throw away the current visual style completely" | Applied. New direction Night launch replaces Instrument panel | tokens.css; screens/app/screens.css (rewritten) |
| 2 | all | "COLOR PALETTE (use exactly these)" | Applied: navy #010D35 to #051B59, #0B5CFF primary, #00053D text, slate #475569 muted, #F8FAFF canvas, #FFFFFF cards, #E6F0FE soft blue, glow rgba(59,130,246,0.3). Applied with a change: status colours kept, and the light-page title gradient darkened for contrast | tokens.css :root |
| 3 | all | "Plus Jakarta Sans from Google Fonts, used everywhere" | Applied, with Noto Sans Devanagari after it in every stack because Plus Jakarta Sans has no Devanagari. Kalam (optional) not used: no small notes exist that would not add new copy | build.py FONTS; tokens.css --font-* |
| 4 | all | "Buttons: fully rounded pills", "Cards: white, rounded 16 to 24px", "Inputs: rounded, light border, blue focus ring", "Badges ... soft-blue pills", "Tables ... #E6F0FE hover" | Applied. Field edges stay #7C8AA8 (3.5:1) so fields keep their 3:1 boundary | screens.css controls, badges, cards, tables |
| 5 | all except S-01 | "light canvas with the dark navy navigation" | Applied: navy gradient side rail with the product mark and nav (active item a blue pill with glow), light top bar with brand, status, person and Sign out; phone top bar navy, tab bar white with a soft-blue pill | build.py sidenav(), topbar(), desktop(); screens.css app and phone chrome |
| 6 | S-01 | "full dark navy gradient background ... glowing curved light streaks ... a white or glass card" | Applied: navy gradient with a bottom glow, inline SVG light streaks, white form card, gradient words "review intelligence" | build.py signin_screen(), STREAKS; screens.css sign-in |
| 7 | all | "screen headings can have one word in the blue gradient"; "Charts and stats ... big bold numbers, blue palette" | Applied: one word of each page title in the gradient; movers numbers, reply count and import counts at 40 px 800; sparklines blue; heatmap now a blue ramp of rounded cells (was amber) | build.py headline(); screens.css .lead, .big, .facts, heatmap; tokens.css --color-heat-* |
| 8 | empty states | "floating chat-bubble or notification-card style elements" | Applied: two floating, wordless cards above every empty state, hidden from screen readers | build.py DECO, EMPTY_OPEN; screens.css .deco |
| 9 | all | "fade-and-slide-in on load, hover lift on cards and buttons, smooth 200ms transitions" | Applied: 420 ms fade and 10 px rise, 60 ms apart; 2 px lift; 200 ms transitions; off under reduced motion and in screenshot mode. PRD L149 to L150 asked to avoid unnecessary animation; the product owner kept the motion on 2026-10-06 and the PRD constraint was reworded | screens.css motion; design.json concerns |

Also changed for consistency: the feature index (screens/app/index.html, from build.py INDEX_CSS) and the gallery
(docs/design/index.html, rendered with the new repository template docs/design/gallery.template.html) restyled to
match; theme toggle and its script removed from every page; annotations that named IBM Plex or the amber ramp reworded;
redlines list the new token names; design.json has version 2, the new direction, brief and two updated concerns.
screens.json lists screens and states only; it holds no tokens, so it changed only by being rebuilt.

Self-review fixes made before showing (found on Chrome screenshots): the comparison table clipped its Replied column
inside the new card padding (tighter cell padding, 76 px sparklines); side cards stretched to the row height and left
empty white space (cards now start-aligned).

Not changed: shots/ dark PNGs are still written by evidence.py and now equal the light ones, so its check reports
36 dark-theme problems, all expected after dropping dark. The browse engine that evidence.py prefers drops some word
spaces in Plus Jakarta Sans ("Allreviewstagged"); real Chrome renders them correctly, checked on S-02 to S-08.
Stop condition: continuing
