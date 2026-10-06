#!/usr/bin/env python3
"""Build the OutletOwl screen prototypes (HTML design bundle, feature "app").

This file is the source of every S-*.html page in this folder. Edit it and run
    python3 docs/design/screens/app/build.py
Never edit the generated pages by hand: the next build overwrites them.

Fixture numbers (movers, urgent counts, heatmap, trends, digest) are computed
here once from the tables below, so every screen shows the same facts.
Invented brand and people only; e-mail addresses use example.in.
"""

import html
import os
import re

HERE = os.path.dirname(os.path.abspath(__file__))
PRODUCT = "OutletOwl"
BRAND = "Neem Tree Kitchens"
FONTS = (
    "https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;500;600;700;800"
    "&family=Noto+Sans+Devanagari:wght@400;500;600;700&display=swap"
)
E = html.escape

# ----------------------------------------------------------------- fixtures
ADMIN = ("Ritika Rao", "ritika.rao@example.in")
OUTLETS = [
    ("Indiranagar", "Neha Kulkarni"),
    ("Koramangala", "Arjun Mehta"),
    ("HSR Layout", "Farah Siddiqui"),
    ("Whitefield", "Rohan Dsouza"),
    ("Jayanagar", "Lakshmi Iyer"),
]
MANAGER_OUTLET = "Koramangala"
MANAGER = "Arjun Mehta"
THEMES = ["Food", "Wait time", "Staff", "Cleanliness", "Price"]
WEEK = "28 Sep to 4 Oct 2026"
PREV_WEEK = "21 to 27 Sep"
WINDOW4 = "7 Sep to 4 Oct 2026"

# Negative reviews per outlet and theme for the last 4 complete weeks (oldest first).
NEG = {
    "Indiranagar": {"Food": [2, 1, 2, 2], "Wait time": [1, 2, 1, 1], "Staff": [1, 0, 1, 1], "Cleanliness": [0, 1, 0, 0], "Price": [1, 2, 2, 5]},
    "Koramangala": {"Food": [2, 2, 1, 2], "Wait time": [2, 3, 3, 11], "Staff": [1, 1, 2, 3], "Cleanliness": [0, 0, 1, 1], "Price": [1, 1, 1, 1]},
    "HSR Layout": {"Food": [1, 2, 1, 1], "Wait time": [1, 1, 2, 2], "Staff": [0, 1, 1, 1], "Cleanliness": [1, 0, 1, 3], "Price": [1, 1, 0, 1]},
    "Whitefield": {"Food": [2, 1, 2, 2], "Wait time": [1, 1, 1, 1], "Staff": [3, 2, 4, 2], "Cleanliness": [0, 1, 0, 0], "Price": [1, 0, 1, 1]},
    "Jayanagar": {"Food": [2, 3, 3, 2], "Wait time": [1, 1, 0, 1], "Staff": [1, 1, 1, 1], "Cleanliness": [0, 0, 0, 1], "Price": [0, 1, 1, 1]},
}

# Weekly average rating and negative share (%) for 12 weeks, oldest first.
RATING = {
    "Indiranagar": [4.4, 4.3, 4.4, 4.5, 4.4, 4.3, 4.4, 4.4, 4.3, 4.4, 4.3, 4.1],
    "Koramangala": [4.3, 4.2, 4.4, 4.3, 4.2, 4.3, 4.4, 4.2, 4.3, 4.2, 4.1, 3.4],
    "HSR Layout": [4.1, 4.2, 4.0, 4.1, 4.2, 4.1, 4.0, 4.2, 4.1, 4.0, 4.1, 3.9],
    "Whitefield": [3.9, 4.0, 3.8, 3.9, 4.0, 3.9, 3.8, 3.9, 3.7, 3.8, 3.7, 3.9],
    "Jayanagar": [4.5, 4.6, 4.5, 4.4, 4.6, 4.5, 4.5, 4.6, 4.5, 4.4, 4.5, 4.4],
}
NEGSHARE = {
    "Indiranagar": [12, 14, 11, 10, 12, 13, 12, 11, 14, 12, 15, 33],
    "Koramangala": [14, 16, 12, 15, 17, 14, 12, 16, 15, 18, 20, 52],
    "HSR Layout": [18, 16, 20, 17, 15, 18, 19, 16, 18, 20, 18, 30],
    "Whitefield": [22, 20, 24, 21, 19, 22, 25, 23, 26, 24, 27, 18],
    "Jayanagar": [9, 8, 10, 11, 8, 9, 10, 8, 9, 12, 10, 17],
}
TOTALS = {"Indiranagar": 300, "Koramangala": 312, "HSR Layout": 288, "Whitefield": 296, "Jayanagar": 304}  # sums to 1,500
REVIEWS_WEEK = {"Indiranagar": 12, "Koramangala": 21, "HSR Layout": 10, "Whitefield": 11, "Jayanagar": 12}
REPLIED_WEEK = {"Indiranagar": 10, "Koramangala": 9, "HSR Layout": 10, "Whitefield": 8, "Jayanagar": 12}
UNTAGGED = 12

URGENT = [
    {"id": 1488, "outlet": "Jayanagar", "date": "2 Oct 2026", "kind": "Food safety", "source": "Google", "rating": 1,
     "reviewer": "Sunita Verma", "lang": "hi",
     "text": "खाने में कीड़ा मिला और रात को मेरे बेटे का पेट खराब हो गया। रसोई की सफ़ाई की जाँच कीजिए।"},
    {"id": 1452, "outlet": "Indiranagar", "date": "30 Sep 2026", "kind": "Legal threat", "source": "Zomato", "rating": 1,
     "reviewer": "Vikram Shetty", "lang": "en",
     "text": "If I don't get a refund for the spoiled biryani by Friday, I will file a complaint in consumer court."},
    {"id": 1437, "outlet": "Whitefield", "date": "29 Sep 2026", "kind": "Harassment", "source": "Google", "rating": 2,
     "reviewer": "Megha Pillai", "lang": "en",
     "text": "A staff member kept making remarks about my clothes after I asked him to stop."},
]

REVIEW_ROWS = [
    {"id": 1497, "date": "3 Oct 2026", "outlet": "Koramangala", "source": "Google", "rating": 2, "reviewer": "Karan Malhotra", "lang": "en",
     "text": "Waited 50 minutes for a table even with a booking, and the food took another 30. The new floor staff seemed lost. Food itself was good.",
     "themes": ["Wait time", "Staff", "Food"], "sentiment": "Negative", "urgent": None, "reply": "Draft ready"},
    URGENT[0] | {"themes": ["Food", "Cleanliness"], "sentiment": "Negative", "urgent": "Food safety", "reply": "Not replied"},
    {"id": 1479, "date": "2 Oct 2026", "outlet": "Koramangala", "source": "Zomato", "rating": 3, "reviewer": "Ishaan Bose", "lang": "en",
     "text": "Khana accha tha but 45 minute wait was too much yaar, and nobody told us why.",
     "themes": ["Wait time", "Food"], "sentiment": "Negative", "urgent": None, "reply": "Not replied"},
    {"id": 1466, "date": "1 Oct 2026", "outlet": "HSR Layout", "source": "Google", "rating": 5, "reviewer": "Ananya Ghosh", "lang": "en",
     "text": "Spotless tables, quick service and the paneer tikka was perfect. Will come back with family.",
     "themes": ["Cleanliness", "Wait time", "Food"], "sentiment": "Positive", "urgent": None, "reply": "Replied"},
    URGENT[1] | {"themes": ["Food", "Price"], "sentiment": "Negative", "urgent": "Legal threat", "reply": "Replied"},
    {"id": 1444, "date": "29 Sep 2026", "outlet": "Indiranagar", "source": "Website form", "rating": 3, "reviewer": "Pooja Nair", "lang": "en",
     "text": "Prices went up again this month but the portions got smaller.",
     "themes": ["Price", "Food"], "sentiment": "Negative", "urgent": None, "reply": "Not replied"},
    URGENT[2] | {"themes": ["Staff"], "sentiment": "Negative", "urgent": "Harassment", "reply": "Draft ready"},
    {"id": 1430, "date": "28 Sep 2026", "outlet": "Jayanagar", "source": "Google", "rating": 4, "reviewer": "Rahul Joshi", "lang": "en",
     "text": "Good filter coffee and friendly staff. Parking is hard on weekends.",
     "themes": ["Food", "Staff"], "sentiment": "Positive", "urgent": None, "reply": "Replied"},
]

SUCCESS_REVIEW = REVIEW_ROWS[0]
DRAFT_EN = ("Hi Karan, thank you for telling us about your visit on 3 October, and we are sorry you waited so long "
            "for your table and then for your food. That is not the evening we want for you, especially with a booking. "
            "This week we are working with our new floor team at Koramangala on seating and order timing. "
            "We are glad the food hit the mark, and we would love to welcome you back. Arjun, outlet manager, Koramangala")
DRAFT_HI = ("सुनीता जी, आपने जो बताया उसके लिए हम दिल से माफ़ी चाहते हैं। आपके बेटे की तबीयत के बारे में सुनकर हमें बहुत दुख हुआ। "
            "हमने आज ही जयनगर की रसोई की पूरी जाँच शुरू कर दी है और नतीजे आपको बताएँगे। कृपया रेस्टोरेंट से सीधे संपर्क करें ताकि हम बात कर सकें। "
            "लक्ष्मी, आउटलेट मैनेजर, जयनगर")


def movers(outlets=None):
    rows = []
    for o, themes in NEG.items():
        if outlets and o not in outlets:
            continue
        for t, w in themes.items():
            ch = w[3] - w[2]
            if ch:
                rows.append((o, t, w[2], w[3], ch))
    rows.sort(key=lambda r: (-abs(r[4]), -r[3], r[0]))
    return rows


def heat_total(o, t):
    return sum(NEG[o][t])


def bucket(n):
    return 0 if n == 0 else 1 if n <= 3 else 2 if n <= 6 else 3 if n <= 9 else 4 if n <= 14 else 5


# ----------------------------------------------------------------- icons (stroke = currentColor)
def icon(d):
    return ('<svg viewBox="0 0 24 24" aria-hidden="true" fill="none" stroke="currentColor" stroke-width="1.75" '
            'stroke-linecap="round" stroke-linejoin="round">%s</svg>' % d)


IC = {
    "overview": icon('<path d="M4 20V10M10 20V4M16 20v-7M22 20H2"/>'),
    "reviews": icon('<path d="M4 5h16M4 10h16M4 15h10M4 20h7"/>'),
    "themes": icon('<rect x="3" y="3" width="8" height="8"/><rect x="13" y="3" width="8" height="8"/><rect x="3" y="13" width="8" height="8"/><rect x="13" y="13" width="8" height="8"/>'),
    "outlets": icon('<path d="M3 9l2-5h14l2 5M4 9v11h16V9M9 20v-6h6v6"/>'),
    "import": icon('<path d="M12 15V3M7 8l5-5 5 5M4 15v5h16v-5"/>'),
    "digest": icon('<path d="M3 5h18v14H3zM3 6l9 7 9-7"/>'),
    "more": icon('<circle cx="5" cy="12" r="1.5"/><circle cx="12" cy="12" r="1.5"/><circle cx="19" cy="12" r="1.5"/>'),
    "back": icon('<path d="M15 5l-7 7 7 7"/>'),
    "alert": icon('<path d="M12 3l10 18H2L12 3zM12 10v5M12 18h.01"/>'),
}

# Product mark: a review bubble with a star. Colours come from screens.css classes, never attributes.
LOGO = ('<svg class="logo" viewBox="0 0 32 32" aria-hidden="true"><rect class="logo-b" x="2" y="3" width="28" height="22" rx="9"/>'
        '<path class="logo-b" d="M9 23l-1.5 7 8-7z"/><path class="logo-s" d="M16 8.5l2 4.1 4.5.6-3.3 3.1.8 4.4-4-2.1-4 2.1.8-4.4-3.3-3.1 4.5-.6z"/></svg>')

# Sign-in light streaks: decorative curves across the bottom of the navy page (no ids, so frames never clash).
STREAKS = ('<svg class="streaks" viewBox="0 0 1440 360" preserveAspectRatio="none" aria-hidden="true">'
           '<path class="sk sk3" d="M-40 300 C 300 200, 640 360, 980 260 S 1380 150, 1500 170"/>'
           '<path class="sk sk1" d="M-40 280 C 320 190, 660 330, 1000 240 S 1380 140, 1500 150"/>'
           '<path class="sk sk2" d="M-40 268 C 330 182, 670 316, 1010 232 S 1390 134, 1500 142"/>'
           '<path class="sk sk1 thin" d="M-40 320 C 340 230, 700 350, 1040 270 S 1400 180, 1500 196"/>'
           '<path class="sk sk2 thin" d="M-40 248 C 300 170, 640 290, 990 214 S 1380 120, 1500 124"/></svg>')

# Empty-state decoration: two floating cards in the style of the reference hero, no words in them.
STAR = '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 3l2.7 5.6 6.1.9-4.4 4.3 1 6.1L12 17l-5.4 2.9 1-6.1-4.4-4.3 6.1-.9z"/></svg>'
DECO = ('<div class="deco" aria-hidden="true"><div class="deco-card deco-a"><span class="deco-avatar"></span>'
        '<span class="deco-lines"><i></i><i></i></span><span class="deco-stars">%s</span></div>'
        '<div class="deco-card deco-b"><span class="deco-dot"></span><span class="deco-lines"><i></i></span></div></div>') % (STAR * 5)
EMPTY_OPEN = re.compile(r'(<(?:div|section) class="[^"]*\bempty\b[^"]*"[^>]*>)')

# One word of each page title in the blue gradient.
H1 = re.compile(r'<h1>([^<]+)</h1>')


def headline(fragment):
    def wrap(m):
        t = m.group(1)
        if ": " in t:
            a, b = t.split(": ", 1)
            return '<h1>%s: <span class="hl">%s</span></h1>' % (a, b)
        if t.startswith("Review from "):
            return '<h1>Review from <span class="hl">%s</span></h1>' % t[len("Review from "):]
        if t == "Review not available":
            return '<h1>Review <span class="hl">not available</span></h1>'
        words = t.rsplit(" ", 1)
        return '<h1>%s<span class="hl">%s</span></h1>' % ((words[0] + " ") if len(words) == 2 else "", words[-1])
    return H1.sub(wrap, fragment)


MOTION = ("every screen: content cards fade in and rise 10 px once on load, 60 ms apart (420 ms); cards and buttons lift 2 px on hover "
          "with a 200 ms transition; all of it off under reduced motion and in screenshot mode")

FILES = {
    "S-01":"S-01-sign-in.html", "S-02": "S-02-overview.html", "S-03": "S-03-themes.html",
    "S-04": "S-04-reviews.html", "S-05": "S-05-review-reply.html", "S-06": "S-06-outlets.html",
    "S-07": "S-07-import.html", "S-08": "S-08-digest.html", "S-09": "S-09-more.html",
}
NAV_ADMIN = [("Overview", "S-02", "overview"), ("Reviews", "S-04", "reviews"), ("Themes", "S-03", "themes"),
             ("Outlets", "S-06", "outlets"), ("Import", "S-07", "import"), ("Digest", "S-08", "digest")]
NAV_MANAGER = NAV_ADMIN[:3]
TABS_ADMIN = [("Overview", "S-02", "overview"), ("Reviews", "S-04", "reviews"), ("Themes", "S-03", "themes"),
              ("Import", "S-07", "import"), ("More", "S-09", "more")]
TABS_MANAGER = NAV_MANAGER


def link(sid, state=None):
    return FILES[sid] + ("#state=" + state if state else "")


# ----------------------------------------------------------------- shared pieces
STATUS = {
    "ok": ("ok", "All reviews tagged"),
    "partial": ("warn", "%d reviews not tagged yet" % UNTAGGED),
    "budget": ("danger", "Model budget used up"),
    "paused": ("warn", "Tagging paused"),
    "tagging": ("warn", "Tagging 482 new reviews"),
    "tagging479": ("warn", "Tagging 479 new reviews"),
}


def badge(kind, text):
    return '<span class="badge %s">%s</span>' % (kind, E(text))


def status_badge(status):
    k, t = STATUS[status]
    return '<a class="status badge %s" href="%s" data-component="Status strip">%s</a>' % (k, link("S-02", "partial" if status == "partial" else None), E(t))


def topbar(role, status="ok"):
    who = "%s, brand admin" % ADMIN[0] if role == "admin" else "%s, manager, %s" % (MANAGER, MANAGER_OUTLET)
    return ('<header class="topbar" data-component="Navigation">'
            '<span class="brand-sub">%s</span>'
            '<span class="spacer"></span>%s<span class="muted who">%s</span>'
            '<a class="btn btn-secondary btn-small" href="%s">Sign out</a></header>') % (
        E(BRAND), status_badge(status), E(who), link("S-01"))


def sidenav(role, current):
    items = NAV_ADMIN if role == "admin" else NAV_MANAGER
    out = []
    for label, sid, ic in items:
        cur = ' aria-current="page"' if sid == current else ""
        out.append('<a href="%s"%s>%s%s</a>' % (link(sid), cur, IC[ic], label))
    return ('<div class="rail"><a class="brand" href="%s">%s<span>%s</span></a>'
            '<nav class="sidenav" aria-label="Sections" data-component="Navigation">%s</nav></div>') % (link("S-02"), LOGO, PRODUCT, "".join(out))


def desktop(role, current, content, status="ok", cols=""):
    return ('<div data-frame="desktop"><p class="frame-label" data-doc>Desktop, 1440 (at 768 the navy side nav becomes a navy bar across the top)</p>'
            '<div class="app" data-app-chrome="side-nav">%s%s<div class="content %s" data-component="Page">%s</div></div></div>') % (
        sidenav(role, current), topbar(role, status), cols, headline(content))


def m_top(title, back=None, status=None):
    b = ('<a class="m-back" href="%s" aria-label="Back to %s">%s</a>' % (link(back[0]), E(back[1]), IC["back"])) if back else ""
    s = (' ' + status_badge(status)) if status and status != "ok" else ""
    return '<header class="m-top" data-component="Navigation">%s<p>%s</p><span class="spacer"></span>%s</header>' % (b, E(title), s)


def tabbar(role, current):
    items = TABS_ADMIN if role == "admin" else TABS_MANAGER
    out = []
    for label, sid, ic in items:
        cur = ' aria-current="page"' if sid == current or (sid == "S-09" and current in ("S-06", "S-08", "S-09")) else ""
        out.append('<a href="%s"%s>%s%s</a>' % (link(sid), cur, IC[ic], label))
    return '<nav class="tabbar n%d" aria-label="Sections" data-component="Navigation">%s</nav>' % (len(items), "".join(out))


def mobile(role, current, title, body, note, dock="", back=None, status="ok"):
    d = ('<div class="m-dock">%s</div>' % dock) if dock else ""
    return ('<div data-frame="mobile"><p class="frame-label" data-doc>Mobile, 375: %s</p>'
            '<div class="phone" data-app-chrome="bottom-tab-bar">%s<div class="m-body" data-component="Page">%s</div>%s%s</div></div>') % (
        E(note), m_top(title, back, status), body, d, tabbar(role, current))


def notice(kind, title, text, action=""):
    return ('<div class="notice %s" role="status" data-component="Notice"><p><strong>%s</strong> %s</p>%s</div>' % (
        kind, E(title), E(text), action))


def skel(width="100%", h="1em"):
    return '<span class="skeleton" aria-hidden="true" style="max-width:%s;min-height:%s"></span>' % (width, h)


def spark(values, label, kind="rating"):
    w, h, pad = 120, 32, 3
    lo, hi = min(values), max(values)
    if hi == lo:
        hi = lo + 1
    pts = []
    for i, v in enumerate(values):
        x = pad + i * (w - 2 * pad) / (len(values) - 1)
        y = h - pad - (v - lo) * (h - 2 * pad) / (hi - lo)
        pts.append((round(x, 1), round(y, 1)))
    poly = " ".join("%s,%s" % p for p in pts)
    lx, ly = pts[-1]
    return ('<svg class="spark" viewBox="0 0 %d %d" role="img" aria-label="%s">'
            '<polyline points="%s"/><circle cx="%s" cy="%s" r="2.5"/></svg>') % (w, h, E(label), poly, lx, ly)


def stars(n):
    return '<span class="num rating" aria-label="%d out of 5">%d/5</span>' % (n, n)


def sentiment(s):
    k = {"Negative": "neg", "Positive": "ok", "Neutral": "neutral"}[s]
    return badge(k, s)


def urgent_tag(kind):
    return '<span class="badge danger">%sUrgent: %s</span>' % (IC["alert"], E(kind))


def para(text, lang):
    return '<p class="review-text" lang="%s">%s</p>' % (lang, E(text))


# ----------------------------------------------------------------- page assembly
ID_ATTR = re.compile(r'\b(id|for|aria-describedby|aria-labelledby)="([^"#][^"]*)"')


def suffix_ids(fragment, suffix):
    """Every id inside one frame of one state gets its own suffix, so labels and
    descriptions stay bound when all states sit in one document."""
    return ID_ATTR.sub(lambda m: '%s="%s"' % (m.group(1), " ".join(v + "-" + suffix for v in m.group(2).split())), fragment)



def page(sid, name, job, serves, nav_desktop, nav_mobile, states, na=(), revisions="", motion=MOTION, platform="both"):
    names = [s["name"] for s in states]
    head = """<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="color-scheme" content="light">
<title>app: {sid} {name}</title>
<!--
  Screen {sid}: {name}   (design.json key app/{sid})
  Job: {job}
  Serves: {serves}   Platform: {platform}
  Framework: react-shadcn later; HTML bundle now (no app or dependencies yet). Components: no contract.
  Tokens: linked from ../../tokens.css, the one file that names a colour.
  States: {states}
  Navigation, desktop: {navd}
  Navigation, mobile: {navm}
  Motion moment: {motion}
  Events: no sheet
  Generic-default revisions: {rev}
  Generated by build.py in this folder; edit that file, not this page.
-->
{na}
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link rel="stylesheet" href="{fonts}">
<link rel="stylesheet" href="../../tokens.css">
<link rel="stylesheet" href="screens.css">
</head>
<body>
<a class="skip" href="#main">Skip to content</a>
<header class="doc-head" data-doc>
  <p class="crumb"><a href="../../index.html">All screens</a> / <a href="index.html">app</a></p>
  <h1>{sid} {name}: every state</h1>
  <p>{job} <span class="muted">Serves {serves}.</span></p>
  <p><strong>Navigation, desktop:</strong> {navd}<br><strong>Navigation, mobile:</strong> {navm}</p>
</header>
<nav class="states" aria-label="Screen states" data-component="prototype-chrome">
  <span class="label">{sid} state:</span>
  <button type="button" data-state-target="all" aria-pressed="false">All states</button>
  {buttons}
</nav>
<main id="main">
""".format(sid=sid, name=E(name), job=E(job), serves=E(serves), platform=platform, states=", ".join(names),
           navd=E(nav_desktop), navm=E(nav_mobile), motion=E(motion), rev=E(revisions), fonts=E(FONTS),
           na="\n".join("<!-- n/a: %s because %s -->" % (s, E(r)) for s, r in na),
           buttons="".join('<button type="button" data-state-target="%s" aria-pressed="false">%s</button>' % (n, n) for n in names))
    body = []
    for i, s in enumerate(states):
        hidden = "" if i == 0 else " hidden"
        body.append("""<section class="state" data-state="{n}" aria-labelledby="st-{n}"{h}>
  <div class="annotation" data-doc><p class="tag">State</p><h2 id="st-{n}">{t}</h2><p data-annotation>{a}</p></div>
  <div class="frames">{d}{m}</div>
</section>""".format(n=s["name"], h=hidden, t=E(s["title"]), a=E(s["annotation"]), d=suffix_ids(EMPTY_OPEN.sub(r"\1" + DECO, s.get("desktop", "")), s["name"] + "-d"), m=suffix_ids(EMPTY_OPEN.sub(r"\1" + DECO, s["mobile"]), s["name"] + "-m")))
    tail = """
</main>
<script>
  // The only script: state switcher and screenshot mode. Nothing is posted anywhere.
  (function () {
    var root = document.documentElement;
    var panels = document.querySelectorAll('section[data-state]');
    var buttons = document.querySelectorAll('[data-state-target]');
    function params() { var out = {}; location.hash.replace(/^#/, '').split('&').forEach(function (kv) {
      var p = kv.split('='); if (p[0]) out[p[0]] = p[1] || ''; }); return out; }
    function show(name) { var found = false;
      panels.forEach(function (p) { var on = name === 'all' || p.dataset.state === name; p.hidden = !on; found = found || on; });
      if (!found && panels[0]) { panels[0].hidden = false; name = panels[0].dataset.state; }
      buttons.forEach(function (b) { b.setAttribute('aria-pressed', String(b.dataset.stateTarget === name)); }); }
    function applyHash() { var p = params(); var shot = p.chrome === '0';
      if (shot) root.setAttribute('data-chrome', '0'); else root.removeAttribute('data-chrome');
      show(p.state || (shot ? (panels[0] && panels[0].dataset.state) : 'all')); }
    buttons.forEach(function (b) { b.addEventListener('click', function () { var p = params(); p.state = b.dataset.stateTarget;
      location.hash = Object.keys(p).map(function (k) { return k + '=' + p[k]; }).join('&'); }); });
    addEventListener('hashchange', applyHash); applyHash();
  })();
</script>
</body>
</html>
"""
    with open(os.path.join(HERE, FILES[sid]), "w", encoding="utf-8") as f:
        f.write(head + "\n".join(body) + tail)
    return {"key": "app/" + sid, "name": name, "purpose": job, "platform": platform,
            "serves": [x.strip() for x in serves.split(",")], "states": names, "path": "screens/app/" + FILES[sid]}


NAVD_ADMIN = "side nav Overview, Reviews, Themes, Outlets, Import, Digest; top bar with the tagging and budget status and Sign out"
NAVM_ADMIN = "bottom tabs Overview, Reviews, Themes, Import, More (Outlets, Digest, Sign out)"
NAVD_MGR = "side nav Overview, Reviews, Themes (own outlet only); top bar with status and Sign out"
NAVM_MGR = "bottom tabs Overview, Reviews, Themes"
SCREENS = []

# ================================================================= S-01 sign in
def signin_form(error="", busy=False, info=""):
    err = '<p class="error" id="signin-error" role="alert">%s</p>' % E(error) if error else '<p class="error" id="signin-error"></p>'
    inv = ' aria-invalid="true" aria-describedby="signin-error"' if error else ""
    btn = ('<button class="btn btn-primary btn-block" type="submit" aria-busy="true" disabled>Signing in</button>' if busy
           else '<button class="btn btn-primary btn-block" type="submit">Sign in</button>')
    inf = notice("warn", "Signed out.", info) if info else ""
    return ('<form class="signin" data-component="Sign-in form">%s'
            '<div><label for="email{u}">Work email</label><input id="email{u}" type="email" autocomplete="username" value="ritika.rao@example.in"%s></div>'
            '<div><label for="pw{u}">Password</label><input id="pw{u}" type="password" autocomplete="current-password" value="........"%s></div>'
            '%s%s<p class="muted small">Accounts are set up by your administrator. To get access or reset a password, ask them.</p></form>') % (
        inf, inv, inv, err, btn)


def signin_screen(form, note):
    u = str(id(form))[-4:]
    form = form.replace("{u}", u)
    d = ('<div data-frame="desktop"><p class="frame-label" data-doc>Desktop, 1440</p>'
         '<div class="app signin-app" data-app-chrome="none: the sign-in page comes before any navigation exists">'
         '<div class="signin-wrap"><div class="signin-intro"><p class="brand big">%s<span>%s</span></p><h1>Sign in to <span class="hl">review intelligence</span> for %s</h1>'
         '<p class="muted">Movers, urgent reviews and reply drafts for every outlet.</p></div>%s</div>%s</div></div>') % (LOGO, PRODUCT, E(BRAND), form, STREAKS)
    m = ('<div data-frame="mobile"><p class="frame-label" data-doc>Mobile, 375: %s</p>'
         '<div class="phone signin-phone" data-app-chrome="none: the sign-in page comes before any navigation exists">'
         '<div class="m-body"><p class="brand big">%s<span>%s</span></p><h1 class="m-h1">Sign <span class="hl">in</span></h1>%s</div>%s</div></div>') % (E(note), LOGO, PRODUCT, form.replace('id="email', 'id="m-email').replace('for="email', 'for="m-email').replace('id="pw', 'id="m-pw').replace('for="pw', 'for="m-pw').replace('signin-error', 'm-signin-error'), STREAKS)
    return d, m


st = []
d, m = signin_screen(signin_form(), "one column; the button is the full width so it sits under the thumb")
st.append({"name": "success", "title": "Ready to sign in", "desktop": d, "mobile": m,
           "annotation": "The default: two labelled fields and one verb button; the intro names the brand so a manager knows they are in the right place, with no marketing copy."})
d, m = signin_screen(signin_form(busy=True), "the button shows progress in place; fields stay filled")
st.append({"name": "loading", "title": "Signing in", "desktop": d, "mobile": m,
           "annotation": "After Sign in is pressed the button reads Signing in and is disabled, so a double press cannot send two requests; the fields keep their values."})
d, m = signin_screen(signin_form(error="That email and password do not match an account. Check both and try again."), "the message sits under the fields, above the button")
st.append({"name": "error", "title": "Wrong email or password (401)", "desktop": d, "mobile": m,
           "annotation": "A 401 marks both fields invalid and says what to do; it never says which field was wrong, so accounts cannot be guessed."})
d, m = signin_screen(signin_form(info="Your session ended after 8 hours. Sign in again to continue."), "the notice sits above the fields")
st.append({"name": "session-expired", "title": "Session expired (8 hour token)", "desktop": d, "mobile": m,
           "annotation": "A request with an expired token returns here with a notice, so the 8 hour limit from design choice 4 is explained instead of looking like a crash."})
SCREENS.append(page("S-01", "Sign in", "Sign in with an account from the users file to reach the overview.", "US-00-001",
                    "none (signed out)", "none (signed out)", st,
                    na=[("empty", "the form is never empty of content; there is nothing to list"), ("partial", "sign-in either succeeds or fails")],
                    revisions="default centred card on a full-bleed photo replaced by a two-column navy page: product line with gradient words on the left, the white form card on the right, inline SVG light streaks instead of an image"))

# ================================================================= S-02 overview
def movers_table(rows, link_state="success"):
    trs = []
    for o, t, a, b, ch in rows:
        word = "up" if ch > 0 else "down"
        cls = "up" if ch > 0 else "down"
        trs.append('<tr><th scope="row"><a href="%s">%s</a></th><td>%s</td><td class="num">%d</td><td class="num">%d</td>'
                   '<td class="num change %s">%s %d</td></tr>' % (link("S-04", link_state), E(o), E(t), a, b, cls, word, abs(ch)))
    return ('<div class="table-scroll"><table class="data" data-component="Movers table"><caption class="sr-only">Biggest movers, negative reviews by outlet and theme</caption>'
            '<thead><tr><th scope="col">Outlet</th><th scope="col">Theme</th><th scope="col" class="num">%s</th><th scope="col" class="num">This week</th>'
            '<th scope="col" class="num">Change</th></tr></thead><tbody>%s</tbody></table></div>') % ("Last week", "".join(trs))


def urgent_list(items, compact=False):
    if not items:
        return '<p class="muted">No urgent reviews this week.</p>'
    lis = []
    for u in items:
        lis.append('<li><p class="u-head">%s <span class="muted">%s, %s</span></p>%s<a href="%s">Open review %d</a></li>' % (
            urgent_tag(u["kind"]), E(u["outlet"]), E(u["date"]),
            '<p class="review-text clamp" lang="%s">%s</p>' % (u["lang"], E(u["text"])), link("S-05", "hindi" if u["lang"] == "hi" else "success"), u["id"]))
    return '<ul class="urgent-list" data-component="Urgent list">%s</ul>' % "".join(lis)


def outlets_table(outs):
    trs = []
    for o, _ in OUTLETS:
        if o not in outs:
            continue
        r = RATING[o]
        dr = round(r[-1] - r[-2], 1)
        word = "up" if dr > 0 else "down" if dr < 0 else "no change"
        trs.append('<tr><th scope="row">%s</th><td class="num">%.1f</td><td class="num change %s">%s</td><td>%s</td>'
                   '<td class="num">%d%%</td><td>%s</td><td class="num">%d/%d</td></tr>' % (
                       E(o), r[-1], "down" if dr < 0 else "up" if dr > 0 else "", ("%s %.1f" % (word, abs(dr))) if dr else word,
                       spark(r, "%s rating over 12 weeks, %.1f to %.1f" % (o, r[0], r[-1])),
                       NEGSHARE[o][-1], spark(NEGSHARE[o], "%s negative share over 12 weeks, %d%% to %d%%" % (o, NEGSHARE[o][0], NEGSHARE[o][-1])),
                       REPLIED_WEEK[o], REVIEWS_WEEK[o]))
    return ('<div class="table-scroll"><table class="data compare" data-component="Outlet comparison">'
            '<caption class="sr-only">Outlets compared, latest complete week and 12 week trends</caption>'
            '<thead><tr><th scope="col">Outlet</th><th scope="col" class="num">Rating</th><th scope="col" class="num">Change</th>'
            '<th scope="col">12 weeks</th><th scope="col" class="num">Negative</th><th scope="col">12 weeks</th>'
            '<th scope="col" class="num">Replied</th></tr></thead><tbody>%s</tbody></table></div>') % "".join(trs)


def overview_head(extra=""):
    tot = sum(REVIEWS_WEEK.values())
    return ('<div class="span-all page-head"><h1>Overview</h1><p class="muted">Latest complete week: %s, compared with %s. '
            '<span class="num">%d</span> reviews this week.</p>%s</div>') % (WEEK, PREV_WEEK, tot, extra)


def overview_desktop(role="admin", status="ok", extra=""):
    if role == "admin":
        mv = movers()[:5]
        top = mv[0]
        lead = '<p class="lead">%s, %s: negative reviews up from <span class="num">%d</span> to <span class="num">%d</span>.</p>' % (E(top[0]), E(top[1]).lower(), top[2], top[3])
        main = ('<section aria-labelledby="h-movers"><h2 id="h-movers">Biggest movers</h2>%s%s'
                '<p class="small muted">Ranked by the change in negative reviews per outlet and theme. Open a row to read those reviews.</p></section>') % (lead, movers_table(mv))
        side = ('<aside aria-labelledby="h-urgent" class="urgent-panel tall"><h2 id="h-urgent">Urgent this week <span class="count num">%d</span></h2>%s'
                '<a class="btn btn-secondary" href="%s">Show all urgent reviews</a></aside>') % (len(URGENT), urgent_list(URGENT), link("S-04", "urgent-filter"))
        comp = '<section aria-labelledby="h-compare"><h2 id="h-compare">Outlets compared</h2>%s</section>' % outlets_table([o for o, _ in OUTLETS])
        return overview_head(extra) + main + side + comp
    mv = movers([MANAGER_OUTLET])[:5]
    top = mv[0]
    lead = '<p class="lead">%s: negative reviews up from <span class="num">%d</span> to <span class="num">%d</span>.</p>' % (E(top[1]), top[2], top[3])
    head = ('<div class="span-all page-head"><h1>Overview: %s</h1><p class="muted">Latest complete week: %s, compared with %s. '
            '<span class="num">%d</span> reviews this week, <span class="num">%d</span> replied.</p>%s</div>') % (
        MANAGER_OUTLET, WEEK, PREV_WEEK, REVIEWS_WEEK[MANAGER_OUTLET], REPLIED_WEEK[MANAGER_OUTLET], extra)
    main = '<section aria-labelledby="h-movers-m"><h2 id="h-movers-m">What moved at %s</h2>%s%s</section>' % (MANAGER_OUTLET, lead, movers_table(mv))
    side = ('<aside aria-labelledby="h-todo" class="urgent-panel"><h2 id="h-todo">To reply</h2><p><span class="num big">12</span> reviews this week have no reply yet. '
            'No urgent reviews this week.</p><a class="btn btn-primary" href="%s">Reply to reviews</a></aside>') % link("S-04", "manager")
    comp = '<section class="span-all" aria-labelledby="h-trend-m"><h2 id="h-trend-m">%s over 12 weeks</h2>%s</section>' % (MANAGER_OUTLET, outlets_table([MANAGER_OUTLET]))
    return head + main + side + comp


def overview_mobile(role="admin", extra=""):
    if role == "admin":
        mv = movers()[:3]
        top = mv[0]
        rows = "".join('<li><a href="%s"><span>%s, %s</span><span class="num change %s">%s %d</span></a></li>' % (
            link("S-04"), E(o), E(t).lower(), "up" if ch > 0 else "down", "up" if ch > 0 else "down", abs(ch)) for o, t, a, b, ch in mv)
        outs = "".join('<li><span>%s</span><span class="num">%.1f</span>%s</li>' % (E(o), RATING[o][-1], spark(RATING[o], "%s rating over 12 weeks" % o)) for o, _ in OUTLETS)
        return ('%s<p class="muted small">Week of %s</p>'
                '<section class="urgent-panel"><h2>Urgent this week <span class="count num">%d</span></h2>%s<a href="%s">Show all urgent reviews</a></section>'
                '<section><h2>Biggest movers</h2><p class="lead">%s, %s: up from <span class="num">%d</span> to <span class="num">%d</span>.</p><ul class="m-list">%s</ul></section>'
                '<section><h2>Rating this week</h2><ul class="m-list spark-list">%s</ul></section>') % (
            extra, WEEK, len(URGENT), urgent_list(URGENT[:2]), link("S-04", "urgent-filter"), E(top[0]), E(top[1]).lower(), top[2], top[3], rows, outs)
    mv = movers([MANAGER_OUTLET])[:3]
    rows = "".join('<li><a href="%s"><span>%s</span><span class="num change %s">%s %d</span></a></li>' % (
        link("S-04", "manager"), E(t), "up" if ch > 0 else "down", "up" if ch > 0 else "down", abs(ch)) for o, t, a, b, ch in mv)
    return ('%s<p class="muted small">%s, week of %s</p><section class="urgent-panel"><h2>To reply</h2><p><span class="num big">12</span> reviews without a reply. No urgent reviews.</p></section>'
            '<section><h2>What moved</h2><ul class="m-list">%s</ul></section>'
            '<section><h2>Rating, 12 weeks</h2><p><span class="num big">%.1f</span> this week %s</p></section>') % (
        extra, MANAGER_OUTLET, WEEK, rows, RATING[MANAGER_OUTLET][-1], spark(RATING[MANAGER_OUTLET], "Koramangala rating over 12 weeks"))


COLS = "cols-2"
st = []
st.append({"name": "success", "title": "This week at a glance",
           "desktop": desktop("admin", "S-02", overview_desktop(), cols=COLS + " moment"),
           "mobile": mobile("admin", "S-02", "Overview", overview_mobile(), "urgent reviews come first, then the movers as one-line rows, then a rating list with sparklines; the comparison table is dropped for the list"),
           "annotation": "The first thing read is one sentence naming the outlet and theme that moved most, then the ranked movers table beside the urgent list; outlets are compared in one ruled table with 12 week sparklines instead of a card per outlet."})
st.append({"name": "loading", "title": "Loading the week",
           "desktop": desktop("admin", "S-02", overview_head() + '<section aria-busy="true"><h2>Biggest movers</h2>%s%s%s</section><aside aria-busy="true"><h2>Urgent this week</h2>%s%s</aside>' % (
               skel("60ch"), skel("100%", "12rem"), "", skel("100%", "3rem"), skel("100%", "3rem")), cols=COLS),
           "mobile": mobile("admin", "S-02", "Overview", '<div aria-busy="true">%s%s%s</div>' % (skel("70%"), skel("100%", "6rem"), skel("100%", "8rem")), "headings render at once, blocks hold their size"),
           "annotation": "Headings and the week line render immediately; the movers table and urgent list keep their final size as skeletons so nothing jumps when the numbers arrive."})
st.append({"name": "empty", "title": "No reviews imported yet",
           "desktop": desktop("admin", "S-02", '<div class="span-all page-head"><h1>Overview</h1></div><div class="span-all empty" data-component="Empty state"><h2>No reviews yet</h2>'
                              '<p>Import a CSV of reviews for your outlets. Movers, urgent reviews and trends appear here once the first complete week is in.</p>'
                              '<a class="btn btn-primary" href="%s">Import reviews</a></div>' % link("S-07"), cols=COLS),
           "mobile": mobile("admin", "S-02", "Overview", '<div class="empty"><h2>No reviews yet</h2><p>Import a CSV of reviews to see movers and urgent reviews.</p></div>',
                            "the invitation and the first action only", dock='<a class="btn btn-primary btn-block" href="%s">Import reviews</a>' % link("S-07")),
           "annotation": "A fresh installation invites the one action that fills it, importing reviews; a manager never sees this state because their outlet comes from the seed or users file with reviews."})
st.append({"name": "error", "title": "Overview could not load",
           "desktop": desktop("admin", "S-02", '<div class="span-all page-head"><h1>Overview</h1></div><div class="span-all"><p class="error" role="alert">The overview could not load because the server did not answer. '
                              'Check that OutletOwl is running, then reload.</p><button class="btn btn-secondary" type="button">Reload overview</button></div>', cols=COLS),
           "mobile": mobile("admin", "S-02", "Overview", '<p class="error" role="alert">The overview could not load because the server did not answer. Check that OutletOwl is running, then reload.</p>',
                            "message, then the retry docked", dock='<button class="btn btn-secondary btn-block" type="button">Reload overview</button>'),
           "annotation": "A network or 5xx failure keeps the navigation so the person can still reach other pages, says what failed and what to do, and offers one retry."})
part = notice("warn", "%d reviews from these two weeks are not tagged yet." % UNTAGGED, "Movers and urgent reviews may be incomplete until tagging finishes; this page updates on reload.")
st.append({"name": "partial", "title": "Some reviews not tagged yet",
           "desktop": desktop("admin", "S-02", overview_desktop(status="partial", extra=part), status="partial", cols=COLS),
           "mobile": mobile("admin", "S-02", "Overview", overview_mobile(extra=part), "the notice sits above the urgent block", status="partial"),
           "annotation": "When reviews dated in the two compared weeks are still untagged, the page says how many and that movers may be incomplete (HLD review fix), and the status strip carries the same count."})
bud = notice("danger", "Model budget used up: USD 8.03 of USD 8.00.", "New reviews are not tagged and reply drafts are unavailable. Movers, search, replies and the digest still work. 5 new reviews are waiting.")
st.append({"name": "budget-stop", "title": "Budget stop (above USD 8 or OpenRouter 402)",
           "desktop": desktop("admin", "S-02", overview_desktop(status="budget", extra=bud), status="budget", cols=COLS),
           "mobile": mobile("admin", "S-02", "Overview", overview_mobile(extra=bud), "the notice sits first", status="budget"),
           "annotation": "When the gateway refuses calls, the admin sees the spend against the limit and exactly what still works, so the stop reads as a rule being kept, not an outage (REQ-031)."})
pau = notice("warn", "Tagging is switched off in this installation's settings.", "7 new reviews are waiting. Ask whoever runs OutletOwl to turn tagging back on.")
st.append({"name": "tagging-paused", "title": "Tagging paused by the operator",
           "desktop": desktop("admin", "S-02", overview_desktop(status="paused", extra=pau), status="paused", cols=COLS),
           "mobile": mobile("admin", "S-02", "Overview", overview_mobile(extra=pau), "the notice sits first", status="paused"),
           "annotation": "The operator switch from the HLD review (TAGGING_ENABLED) is visible to the admin as a plain sentence, kept apart from the budget stop because the fix is different."})
st.append({"name": "manager", "title": "Outlet manager: own outlet only",
           "desktop": desktop("manager", "S-02", overview_desktop(role="manager"), cols=COLS),
           "mobile": mobile("manager", "S-02", "Koramangala", overview_mobile(role="manager"), "the reply count comes first with the movers below",
                            dock='<a class="btn btn-primary btn-block" href="%s">Reply to reviews</a>' % link("S-04", "manager")),
           "annotation": "A manager sees one outlet (Q-002): the comparison table becomes their own 12 week row, and the side panel turns into the reply queue because replying is their job."})
SCREENS.append(page("S-02", "Overview", "See which outlet and theme moved most this week, the urgent reviews, and how outlets compare.",
                    "US-01-005, US-01-007, US-00-001, US-02-001", NAVD_ADMIN, NAVM_ADMIN, st,
                    revisions="default KPI tiles with big numbers replaced by one sentence naming the top mover plus a ruled movers table; per-outlet cards replaced by one comparison table with sparklines",
                    motion=MOTION))

# ================================================================= S-03 themes heatmap
def heatmap(outs, caption):
    head = "".join('<th scope="col" class="num">%s</th>' % E(t) for t in THEMES)
    rows = []
    for o in outs:
        cells = []
        for t in THEMES:
            n = heat_total(o, t)
            cells.append('<td class="cell h%d num"><a href="%s" aria-label="%s, %s: %d negative reviews">%d</a></td>' % (
                bucket(n), link("S-04"), E(o), E(t), n, n))
        rows.append('<tr><th scope="row">%s</th>%s<td class="num total">%d</td></tr>' % (E(o), "".join(cells), sum(heat_total(o, t) for t in THEMES)))
    legend = "".join('<span class="lg"><span class="sw h%d" aria-hidden="true"></span>%s</span>' % (i, lab) for i, lab in
                     enumerate(["0", "1 to 3", "4 to 6", "7 to 9", "10 to 14", "15 or more"]))
    return ('<div class="table-scroll"><table class="data heat" data-component="Theme heatmap"><caption class="sr-only">%s</caption>'
            '<thead><tr><th scope="col">Outlet</th>%s<th scope="col" class="num">Total</th></tr></thead><tbody>%s</tbody></table></div>'
            '<p class="legend small" aria-label="Legend: negative reviews per cell">%s</p>') % (E(caption), head, "".join(rows), legend)


def themes_head(extra="", who="All outlets"):
    return ('<div class="span-all page-head"><h1>Themes</h1><p class="muted">%s. Negative reviews per theme over the last 4 complete weeks, %s. '
            'A review with two themes counts once under each.</p>%s</div>') % (who, WINDOW4, extra)


def themes_mobile(outs, extra=""):
    blocks = []
    for o in outs:
        cells = "".join('<li class="cell h%d"><span>%s</span><span class="num">%d</span></li>' % (bucket(heat_total(o, t)), E(t), heat_total(o, t)) for t in THEMES)
        blocks.append('<section><h2>%s</h2><ul class="heat-row">%s</ul></section>' % (E(o), cells))
    return extra + '<p class="muted small">Negative reviews, %s</p>' % WINDOW4 + "".join(blocks)


ALL = [o for o, _ in OUTLETS]
st = []
st.append({"name": "success", "title": "Heatmap of themes by outlet",
           "desktop": desktop("admin", "S-03", themes_head() + '<div class="span-all">%s</div>' % heatmap(ALL, "Negative reviews by outlet and theme, last 4 weeks")),
           "mobile": mobile("admin", "S-03", "Themes", themes_mobile(ALL), "one outlet per block, its five themes as a shaded row; the grid would need sideways scrolling at 375"),
           "annotation": "One blue ramp shows where complaints cluster, with the number printed in every cell so colour is never the only signal; each cell opens the review list for that outlet and theme."})
st.append({"name": "loading", "title": "Loading the heatmap",
           "desktop": desktop("admin", "S-03", themes_head() + '<div class="span-all" aria-busy="true">%s</div>' % skel("100%", "16rem")),
           "mobile": mobile("admin", "S-03", "Themes", '<div aria-busy="true">%s%s</div>' % (skel("100%", "5rem"), skel("100%", "5rem")), "blocks hold their size"),
           "annotation": "The explanation line renders at once so the person knows the window being counted while the grid loads."})
st.append({"name": "empty", "title": "No tagged reviews in the window",
           "desktop": desktop("admin", "S-03", themes_head() + '<div class="span-all empty"><h2>No tagged reviews in the last 4 weeks</h2><p>Import reviews or wait for tagging to finish; the heatmap fills as reviews are tagged.</p>'
                              '<a class="btn btn-primary" href="%s">Import reviews</a></div>' % link("S-07")),
           "mobile": mobile("admin", "S-03", "Themes", '<div class="empty"><h2>No tagged reviews yet</h2><p>Import reviews to fill the heatmap.</p></div>', "invitation only",
                            dock='<a class="btn btn-primary btn-block" href="%s">Import reviews</a>' % link("S-07")),
           "annotation": "An empty grid of zeros would look like good news, so the page says there is nothing tagged in the window and offers the import."})
st.append({"name": "error", "title": "Heatmap could not load",
           "desktop": desktop("admin", "S-03", themes_head() + '<div class="span-all"><p class="error" role="alert">The heatmap could not load because the server did not answer. Check that OutletOwl is running, then reload.</p><button class="btn btn-secondary" type="button">Reload heatmap</button></div>'),
           "mobile": mobile("admin", "S-03", "Themes", '<p class="error" role="alert">The heatmap could not load because the server did not answer. Check that OutletOwl is running, then reload.</p>', "message, then the retry docked",
                            dock='<button class="btn btn-secondary btn-block" type="button">Reload heatmap</button>'),
           "annotation": "Same failure copy pattern as the overview: what failed, what to check, one retry."})
st.append({"name": "partial", "title": "Some reviews not tagged yet",
           "desktop": desktop("admin", "S-03", themes_head(notice("warn", "%d reviews in this window are not tagged yet." % UNTAGGED, "Counts may rise when tagging finishes.")) + '<div class="span-all">%s</div>' % heatmap(ALL, "Negative reviews by outlet and theme"), status="partial"),
           "mobile": mobile("admin", "S-03", "Themes", themes_mobile(ALL, notice("warn", "%d reviews not tagged yet." % UNTAGGED, "Counts may rise.")), "notice first", status="partial"),
           "annotation": "Counts are a lower bound while reviews are untagged, and the page says so above the grid."})
st.append({"name": "manager", "title": "Outlet manager: one row",
           "desktop": desktop("manager", "S-03", themes_head(who=MANAGER_OUTLET) + '<div class="span-all">%s</div>' % heatmap([MANAGER_OUTLET], "Negative reviews by theme at Koramangala")),
           "mobile": mobile("manager", "S-03", "Themes", themes_mobile([MANAGER_OUTLET]), "the single outlet row"),
           "annotation": "A manager sees only their outlet's row (Q-002); the ramp and legend stay the same so the numbers read the same way as the admin's."})
SCREENS.append(page("S-03", "Themes", "See where negative reviews cluster by outlet and theme.", "US-01-006, US-00-001", NAVD_ADMIN, NAVM_ADMIN, st,
                    revisions="default rainbow heatmap with a colour-only legend replaced by one blue ramp of rounded cells with the count printed in every cell and a labelled legend"))

# ================================================================= S-04 reviews
def filters(urgent=False, manager=False, q=""):
    outlet = "" if manager else ('<div><label for="f-outlet">Outlet</label><select id="f-outlet"><option>All outlets</option>%s</select></div>' % "".join("<option>%s</option>" % E(o) for o in ALL))
    return ('<form class="filters" role="search" data-component="Review filters">'
            '<div class="f-search"><label for="f-q">Search reviews</label><input id="f-q" type="search" placeholder="Words in the review or the reviewer name" value="%s"></div>%s'
            '<div><label for="f-theme">Theme</label><select id="f-theme"><option>All themes</option>%s</select></div>'
            '<div><label for="f-sent">Sentiment</label><select id="f-sent"><option>Any</option><option>Positive</option><option>Neutral</option><option>Negative</option></select></div>'
            '<div><label for="f-reply">Reply</label><select id="f-reply"><option>Any</option><option>Not replied</option><option>Draft ready</option><option>Replied</option></select></div>'
            '<div class="f-check"><input id="f-urgent" type="checkbox"%s><label for="f-urgent">Urgent only</label></div>'
            '</form>') % (E(q), outlet, "".join("<option>%s</option>" % E(t) for t in THEMES), " checked" if urgent else "")


def review_table(rows, untagged_ids=()):
    trs = []
    for r in rows:
        if r["id"] in untagged_ids:
            tags = '<span class="muted">Not tagged yet</span>'
            sent = '<span class="muted">Pending</span>'
        else:
            tags = " ".join('<span class="chip">%s</span>' % E(t) for t in r["themes"])
            sent = sentiment(r["sentiment"])
        urg = urgent_tag(r["urgent"]) if r.get("urgent") else ""
        rep = {"Replied": badge("ok", "Replied"), "Draft ready": badge("neutral", "Draft ready"), "Not replied": badge("warn", "Not replied")}[r["reply"]]
        state = "hindi" if r["lang"] == "hi" else "success"
        trs.append('<tr><td class="date nowrap">%s</td><td>%s</td><td class="num">%s</td><td class="review-cell">%s<p class="review-text clamp" lang="%s"><a href="%s">%s</a></p>'
                   '<p class="small muted">%s, %s</p></td><td>%s</td><td>%s</td><td>%s</td></tr>' % (
                       E(r["date"]), E(r["outlet"]), stars(r["rating"]), urg, r["lang"], link("S-05", state), E(r["text"]),
                       E(r["reviewer"]), E(r["source"]), tags, sent, rep))
    return ('<div class="table-scroll"><table class="data reviews" data-component="Review table"><caption class="sr-only">Reviews, newest first</caption>'
            '<thead><tr><th scope="col">Date</th><th scope="col">Outlet</th><th scope="col" class="num">Rating</th><th scope="col">Review</th>'
            '<th scope="col">Themes</th><th scope="col">Sentiment</th><th scope="col">Reply</th></tr></thead><tbody>%s</tbody></table></div>') % "".join(trs)


def review_mlist(rows, untagged_ids=()):
    lis = []
    for r in rows:
        urg = urgent_tag(r["urgent"]) if r.get("urgent") else ""
        rep = {"Replied": "Replied", "Draft ready": "Draft ready", "Not replied": "Not replied"}[r["reply"]]
        tg = "Not tagged yet" if r["id"] in untagged_ids else ", ".join(r["themes"])
        state = "hindi" if r["lang"] == "hi" else "success"
        lis.append('<li><a href="%s">%s<p class="small muted"><span class="num">%s</span> · %s · %s</p><p class="review-text clamp" lang="%s">%s</p>'
                   '<p class="small"><span class="muted">%s</span> · %s</p></a></li>' % (
                       link("S-05", state), urg, E(r["date"]), E(r["outlet"]), stars(r["rating"]), r["lang"], E(r["text"]), E(tg), E(rep)))
    return '<ul class="m-list reviews-m" data-component="Review list">%s</ul>' % "".join(lis)


def reviews_head(count, extra="", who=""):
    return '<div class="span-all page-head"><h1>Reviews%s</h1><p class="muted"><span class="num">%d</span> reviews, newest first.</p>%s</div>' % (who, count, extra)


def m_filter_bar(n_active):
    return '<div class="m-filterbar"><label class="sr-only" for="m-q">Search reviews</label><input id="m-q" type="search" placeholder="Search reviews"><button class="btn btn-secondary" type="button">Filters%s</button></div>' % (
        " (%d)" % n_active if n_active else "")


TOTAL_REVIEWS = 1500
st = []
st.append({"name": "success", "title": "All reviews, newest first",
           "desktop": desktop("admin", "S-04", reviews_head(TOTAL_REVIEWS) + '<div class="span-all">%s%s</div>' % (filters(), review_table(REVIEW_ROWS))),
           "mobile": mobile("admin", "S-04", "Reviews", m_filter_bar(0) + review_mlist(REVIEW_ROWS), "search stays visible, the five filters move behind one Filters button, rows become a two-line list"),
           "annotation": "Search and every filter from Q-020 sit in one row above a ruled table; urgent reviews carry a red label with the word, and each review opens its reply page."})
st.append({"name": "loading", "title": "Loading reviews",
           "desktop": desktop("admin", "S-04", reviews_head(TOTAL_REVIEWS) + '<div class="span-all" aria-busy="true">%s%s</div>' % (filters(), skel("100%", "20rem"))),
           "mobile": mobile("admin", "S-04", "Reviews", m_filter_bar(0) + '<div aria-busy="true">%s%s</div>' % (skel("100%", "4rem"), skel("100%", "4rem")), "filters stay usable"),
           "annotation": "Filters stay usable while results load so a person can keep narrowing; the table area holds its height."})
st.append({"name": "empty", "title": "No reviews imported yet",
           "desktop": desktop("admin", "S-04", reviews_head(0) + '<div class="span-all empty"><h2>No reviews yet</h2><p>Reviews appear here after a CSV import.</p><a class="btn btn-primary" href="%s">Import reviews</a></div>' % link("S-07")),
           "mobile": mobile("admin", "S-04", "Reviews", '<div class="empty"><h2>No reviews yet</h2><p>Reviews appear here after a CSV import.</p></div>', "invitation only",
                            dock='<a class="btn btn-primary btn-block" href="%s">Import reviews</a>' % link("S-07")),
           "annotation": "Before any import the filters are hidden, because there is nothing to filter, and the page offers the import."})
st.append({"name": "no-results", "title": "No review matches the filters",
           "desktop": desktop("admin", "S-04", reviews_head(0) + '<div class="span-all">%s<div class="empty"><h2>No reviews match</h2><p>Nothing matches "refund" with Theme: Cleanliness and Urgent only. Clear a filter to see more.</p>'
                              '<button class="btn btn-secondary" type="button">Clear filters</button></div></div>' % filters(urgent=True, q="refund")),
           "mobile": mobile("admin", "S-04", "Reviews", m_filter_bar(2) + '<div class="empty"><h2>No reviews match</h2><p>Nothing matches "refund" with 2 filters.</p></div>', "the clear action is docked",
                            dock='<button class="btn btn-secondary btn-block" type="button">Clear filters</button>'),
           "annotation": "Distinct from empty: the filters stay on screen, the message repeats what was searched, and one action clears them."})
st.append({"name": "error", "title": "Reviews could not load",
           "desktop": desktop("admin", "S-04", reviews_head(0) + '<div class="span-all">%s<p class="error" role="alert">Reviews could not load because the server did not answer. Your filters are kept; check that OutletOwl is running, then try again.</p>'
                              '<button class="btn btn-secondary" type="button">Load reviews again</button></div>' % filters()),
           "mobile": mobile("admin", "S-04", "Reviews", m_filter_bar(0) + '<p class="error" role="alert">Reviews could not load because the server did not answer. Check that OutletOwl is running, then try again.</p>', "retry docked",
                            dock='<button class="btn btn-secondary btn-block" type="button">Load reviews again</button>'),
           "annotation": "The filters survive the failure so the person does not rebuild their search after a retry."})
st.append({"name": "partial", "title": "Some reviews not tagged yet",
           "desktop": desktop("admin", "S-04", reviews_head(TOTAL_REVIEWS, notice("warn", "%d reviews are not tagged yet." % UNTAGGED, "They show here but are left out of theme, sentiment and urgent filters until tagging finishes.")) +
                              '<div class="span-all">%s%s</div>' % (filters(), review_table(REVIEW_ROWS, untagged_ids=(1479, 1444))), status="partial"),
           "mobile": mobile("admin", "S-04", "Reviews", notice("warn", "%d reviews not tagged yet." % UNTAGGED, "Filters skip them for now.") + m_filter_bar(0) + review_mlist(REVIEW_ROWS, untagged_ids=(1479, 1444)), "notice first", status="partial"),
           "annotation": "Untagged reviews are listed with Not tagged yet instead of empty cells, and the notice explains why a theme filter can miss them."})
urg_rows = [r for r in REVIEW_ROWS if r.get("urgent")]
st.append({"name": "urgent-filter", "title": "Urgent reviews, opened from the overview",
           "desktop": desktop("admin", "S-04", reviews_head(len(urg_rows), '<p class="small">Showing urgent reviews from the week of %s. <a href="%s">Show all reviews</a></p>' % (WEEK, link("S-04"))) +
                              '<div class="span-all">%s%s</div>' % (filters(urgent=True), review_table(urg_rows))),
           "mobile": mobile("admin", "S-04", "Urgent reviews", m_filter_bar(1) + review_mlist(urg_rows), "the filter count shows 1", back=("S-02", "Overview")),
           "annotation": "The one step from the overview (REQ-046): the list opens with Urgent only ticked and says which week it covers, with a way back to everything."})
mgr_rows = [r for r in REVIEW_ROWS if r["outlet"] == MANAGER_OUTLET]
st.append({"name": "manager", "title": "Outlet manager: own outlet only",
           "desktop": desktop("manager", "S-04", reviews_head(312, who=": %s" % MANAGER_OUTLET) + '<div class="span-all">%s%s</div>' % (filters(manager=True), review_table(mgr_rows))),
           "mobile": mobile("manager", "S-04", "Reviews", m_filter_bar(0) + review_mlist(mgr_rows), "same list, one outlet"),
           "annotation": "A manager has no outlet filter because they only ever see their own outlet (Q-002); the server applies that scope, not this page."})
st.append({"name": "forbidden", "title": "Another outlet's reviews requested (403)",
           "desktop": desktop("manager", "S-04", reviews_head(0, who=": %s" % MANAGER_OUTLET) + '<div class="span-all"><p class="error" role="alert">You can only see reviews for Koramangala. That link points to another outlet.</p><a class="btn btn-secondary" href="%s">Show Koramangala reviews</a></div>' % link("S-04", "manager")),
           "mobile": mobile("manager", "S-04", "Reviews", '<p class="error" role="alert">You can only see reviews for Koramangala. That link points to another outlet.</p>', "way back docked",
                            dock='<a class="btn btn-secondary btn-block" href="%s">Show Koramangala reviews</a>' % link("S-04", "manager")),
           "annotation": "A 403 from a shared or old link says which outlet the manager can see and takes them there, instead of a generic error."})
SCREENS.append(page("S-04", "Reviews", "Search and filter every review, and open one to reply.", "US-01-008, US-00-001", NAVD_ADMIN, NAVM_ADMIN, st,
                    revisions="default card grid of reviews replaced by a ruled table with the review text in the widest column; filter chips replaced by labelled selects so every filter has a visible label"))

# ================================================================= S-05 review and reply
def review_block(r, untagged=False):
    tags = ('<p class="muted">Not tagged yet. Themes and sentiment appear when tagging finishes.</p>' if untagged else
            '<p>%s %s</p>' % (" ".join('<span class="chip">%s</span>' % E(t) for t in r["themes"]), sentiment(r["sentiment"])))
    urg = ('<div class="notice danger" role="note"><p><strong>Urgent: %s.</strong> Reply today and follow your brand\'s escalation steps.</p></div>' % E(r["urgent"])) if r.get("urgent") else ""
    return ('<section class="review-panel" aria-labelledby="h-rev" data-component="Review">%s<h2 id="h-rev" class="sr-only">Review</h2>'
            '<p class="meta">%s · %s · %s · %s</p>%s<p class="small muted">%s</p>%s</section>') % (
        urg, E(r["date"]), E(r["outlet"]), E(r["source"]), stars(r["rating"]), para(r["text"], r["lang"]), E(r["reviewer"]), tags)


def reply_panel(text="", lang="en", mode="draft", note="", to="Karan Malhotra"):
    if mode == "drafting":
        body = '<p class="muted" aria-live="polite">Drafting a reply in English with reply prompt v3. This takes up to 45 seconds.</p>%s%s' % (skel("100%", "8rem"), "")
        actions = '<button class="btn btn-primary" type="button" disabled>Mark as replied</button>'
    elif mode == "replied":
        body = '<p>%s</p><div class="replied-text" lang="%s">%s</div>' % (badge("ok", "Replied"), lang, E(text)) + '<p class="small muted">Marked replied by Arjun Mehta on 4 Oct 2026, 11:20. Prompt v3, edited.</p>'
        actions = ""
    elif mode == "readonly":
        body = '<p class="small muted">%s</p><div class="replied-text" lang="%s">%s</div>' % (E(note), lang, E(text))
        actions = ""
    else:
        dis = ""
        body = ('%s<label for="reply-%s">Reply to %s</label><textarea id="reply-%s" lang="%s" rows="9"%s>%s</textarea>'
                '<p class="small muted">%s</p>') % (note, mode, E(to), mode, lang, dis, E(text),
                                                     "Drafted with reply prompt v3 in the brand's tone. Edit freely; nothing is posted for you." if text else "Write the reply here. Nothing is posted for you.")
        actions = ('<button class="btn btn-secondary" type="button">Save draft</button>'
                   '<button class="btn btn-primary" type="button">Mark as replied</button>')
    act = ('<div class="actions" data-component="Reply actions">%s</div><p class="small muted">Mark as replied after you post this text on %s yourself.</p>' % (actions, "the review site")) if actions else ""
    return '<section class="reply-panel" aria-labelledby="h-reply" data-component="Reply editor"><h2 id="h-reply">Your reply</h2>%s%s</section>' % (body, act)


def rr_desktop(r, panel, role="manager", status="ok", untagged=False):
    head = '<div class="span-all page-head"><p class="small"><a href="%s">Reviews</a> / %d</p><h1>Review from %s</h1></div>' % (link("S-04"), r["id"], E(r["reviewer"]))
    return desktop(role, "S-04", head + review_block(r, untagged=untagged) + panel, status=status, cols="cols-even")


def rr_mobile(r, panel_m, dock, role="manager", status="ok", note="the review first, then the reply editor; actions dock at the bottom"):
    return mobile(role, "S-04", "Review %d" % r["id"], review_block(r) + panel_m, note, dock=dock, back=("S-04", "Reviews"), status=status)


DOCK = '<div class="dock-row"><button class="btn btn-secondary" type="button">Save draft</button><button class="btn btn-primary" type="button">Mark as replied</button></div>'
HI = URGENT[0] | REVIEW_ROWS[1]
r0 = SUCCESS_REVIEW
st = []
st.append({"name": "success", "title": "Draft ready to edit",
           "desktop": rr_desktop(r0, reply_panel(DRAFT_EN)),
           "mobile": rr_mobile(r0, reply_panel(DRAFT_EN).replace('<div class="actions" data-component="Reply actions"><button class="btn btn-secondary" type="button">Save draft</button><button class="btn btn-primary" type="button">Mark as replied</button></div>', ""), DOCK),
           "annotation": "The review and its draft sit side by side so the manager checks the reply against what was said; Mark as replied is the approval (Q-008) and the only glowing blue button."})
st.append({"name": "loading", "title": "Drafting (row claimed, up to 45 seconds)",
           "desktop": rr_desktop(r0, reply_panel(mode="drafting")),
           "mobile": rr_mobile(r0, reply_panel(mode="drafting"), '<button class="btn btn-primary btn-block" type="button" disabled>Mark as replied</button>'),
           "annotation": "Opening a review with no draft claims it and drafts once; a second tab sees this same state instead of paying for a second draft (HLD review fix), and the line says how long it can take."})
st.append({"name": "error", "title": "Draft did not arrive (timeout or provider error)",
           "desktop": rr_desktop(r0, reply_panel(note='<p class="error" role="alert">The draft did not arrive within 45 seconds. Write the reply yourself, or try drafting again.</p><p><button class="btn btn-secondary" type="button">Draft again</button></p>')),
           "mobile": rr_mobile(r0, reply_panel(note='<p class="error" role="alert">The draft did not arrive within 45 seconds. Write the reply yourself, or try drafting again.</p><p><button class="btn btn-secondary btn-block" type="button">Draft again</button></p>').replace('<div class="actions" data-component="Reply actions"><button class="btn btn-secondary" type="button">Save draft</button><button class="btn btn-primary" type="button">Mark as replied</button></div>', ""), DOCK),
           "annotation": "A timeout or 5xx leaves an empty, usable editor and two ways forward; the manager is never blocked from replying by the model."})
st.append({"name": "partial", "title": "Review not tagged yet",
           "desktop": rr_desktop(r0, reply_panel(DRAFT_EN), untagged=True),
           "mobile": mobile("manager", "S-04", "Review %d" % r0["id"], review_block(r0, untagged=True) + reply_panel(DRAFT_EN).replace('<div class="actions" data-component="Reply actions"><button class="btn btn-secondary" type="button">Save draft</button><button class="btn btn-primary" type="button">Mark as replied</button></div>', ""),
                            "tags line says pending", dock=DOCK, back=("S-04", "Reviews")),
           "annotation": "Drafting does not wait for tagging, so a review can have a draft while its themes still say Not tagged yet."})
st.append({"name": "replied", "title": "Marked as replied",
           "desktop": rr_desktop(r0, reply_panel(DRAFT_EN, mode="replied")),
           "mobile": rr_mobile(r0, reply_panel(DRAFT_EN, mode="replied"), "", note="read-only reply with who and when; no dock"),
           "annotation": "After approval the reply is read-only with who marked it and when; marking again changes nothing (idempotent per HLD section 5)."})
st.append({"name": "budget-stop", "title": "Drafting unavailable (budget stop)",
           "desktop": rr_desktop(r0, reply_panel(note=notice("danger", "Drafting is unavailable.", "The model budget is used up. Write the reply yourself; saving and marking as replied still work.")), status="budget"),
           "mobile": rr_mobile(r0, reply_panel(note=notice("danger", "Drafting is unavailable.", "The model budget is used up. Write the reply yourself.")).replace('<div class="actions" data-component="Reply actions"><button class="btn btn-secondary" type="button">Save draft</button><button class="btn btn-primary" type="button">Mark as replied</button></div>', ""), DOCK, status="budget"),
           "annotation": "AC-US-00-002-7: with the budget stop on, the editor is empty and writable, and the notice says what still works."})
st.append({"name": "hindi", "title": "Hindi review, Devanagari draft, urgent",
           "desktop": rr_desktop(HI, reply_panel(DRAFT_HI, lang="hi", to="Sunita Verma"), role="manager"),
           "mobile": rr_mobile(HI, reply_panel(DRAFT_HI, lang="hi", to="Sunita Verma").replace('<div class="actions" data-component="Reply actions"><button class="btn btn-secondary" type="button">Save draft</button><button class="btn btn-primary" type="button">Mark as replied</button></div>', ""), DOCK),
           "annotation": "Devanagari review and draft use Noto Sans Devanagari (Plus Jakarta Sans has no Devanagari) with a taller line height so vowel marks never touch; the urgent banner states the kind in words (Q-009)."})
st.append({"name": "admin-view", "title": "Brand admin: read only",
           "desktop": rr_desktop(r0, reply_panel(DRAFT_EN, mode="readonly", note="Draft by reply prompt v3. Only Arjun Mehta, the Koramangala manager, can edit or mark it replied."), role="admin"),
           "mobile": mobile("admin", "S-04", "Review %d" % r0["id"], review_block(r0) + reply_panel(DRAFT_EN, mode="readonly", note="Only Arjun Mehta, the Koramangala manager, can edit or mark it replied."), "read-only, no dock", back=("S-04", "Reviews")),
           "annotation": "The admin reads the draft but has no reply actions (Q-002); the line names who can act, so the absence of buttons is explained."})
st.append({"name": "no-manager", "title": "Outlet has no manager",
           "desktop": rr_desktop(r0 | {"outlet": "Electronic City", "id": 1501}, reply_panel("", mode="readonly", note="No manager is assigned to Electronic City, so nobody can reply to its reviews. Add a manager for this outlet to the users file and restart OutletOwl."), role="admin"),
           "mobile": mobile("admin", "S-04", "Review 1501", review_block(r0 | {"outlet": "Electronic City"}) + reply_panel("", mode="readonly", note="No manager is assigned to Electronic City. Add one to the users file and restart."), "read-only, no dock", back=("S-04", "Reviews")),
           "annotation": "The combined rule from reconciliation: admins cannot reply and this outlet has no manager, so the page says exactly how to unblock it (HLD users file)."})
st.append({"name": "forbidden", "title": "Another outlet's review (403)",
           "desktop": desktop("manager", "S-04", '<div class="span-all page-head"><h1>Review not available</h1><p class="error" role="alert">This review belongs to another outlet. You can reply to Koramangala reviews only.</p><a class="btn btn-secondary" href="%s">Show Koramangala reviews</a></div>' % link("S-04", "manager"), cols="cols-even"),
           "mobile": mobile("manager", "S-04", "Review", '<p class="error" role="alert">This review belongs to another outlet. You can reply to Koramangala reviews only.</p>', "way back docked",
                            dock='<a class="btn btn-secondary btn-block" href="%s">Show Koramangala reviews</a>' % link("S-04", "manager"), back=("S-04", "Reviews")),
           "annotation": "A 403 on a review id from another outlet names the outlet the manager owns and links there."})
SCREENS.append(page("S-05", "Review and reply", "Read one review, edit its drafted reply and mark it replied.", "US-00-002, US-00-003", NAVD_MGR, NAVM_MGR, st,
                    na=[("empty", "a review page always has a review; a missing id is the 403 or 404 state")],
                    revisions="default chat-bubble layout replaced by review and editor side by side at equal width; three action buttons cut to two, with Mark as replied as the single accent"))

# ================================================================= S-06 outlets
def outlet_rows(extra=None):
    rows = [(o, m, TOTALS[o], 0) for o, m in OUTLETS]
    trs = "".join('<tr><th scope="row">%s</th><td>%s</td><td class="num">%d</td><td class="num">%d</td></tr>' % (E(o), E(m), n, u) for o, m, n, u in rows)
    if extra:
        trs += '<tr><th scope="row">%s</th><td>%s</td><td class="num">0</td><td class="num">0</td></tr>' % (E(extra), badge("warn", "No manager: add one to the users file"))
    return ('<div class="table-scroll"><table class="data" data-component="Outlet table"><caption class="sr-only">Outlets</caption><thead><tr><th scope="col">Outlet</th><th scope="col">Manager</th>'
            '<th scope="col" class="num">Reviews</th><th scope="col" class="num">Not tagged</th></tr></thead><tbody>%s</tbody></table></div>') % trs


def add_form(value="", error=""):
    inv = ' aria-invalid="true" aria-describedby="outlet-err"' if error else ""
    return ('<form class="add-outlet" data-component="Add outlet form"><h2>Add an outlet</h2><label for="o-name">Outlet name</label>'
            '<input id="o-name" value="%s"%s><p class="error" id="outlet-err"%s>%s</p><p class="small muted">The name must match the outlet column in your CSV files. '
            'The outlet\'s manager is added in the users file.</p><button class="btn btn-primary" type="submit">Add outlet</button></form>') % (
        E(value), inv, ' role="alert"' if error else "", E(error))


def outlets_head(n):
    return '<div class="span-all page-head"><h1>Outlets</h1><p class="muted"><span class="num">%d</span> outlets.</p></div>' % n


def outlets_m(extra=None):
    lis = "".join('<li><span>%s</span><span class="small muted">%s</span></li>' % (E(o), E(m)) for o, m in OUTLETS)
    if extra:
        lis += '<li><span>%s</span>%s</li>' % (E(extra), badge("warn", "No manager"))
    return '<ul class="m-list">%s</ul>' % lis


st = []
st.append({"name": "success", "title": "Outlets and their managers",
           "desktop": desktop("admin", "S-06", outlets_head(5) + '<section>%s</section><aside>%s</aside>' % (outlet_rows(), add_form()), cols=COLS),
           "mobile": mobile("admin", "S-06", "Outlets", outlets_m() + add_form().replace("<h2>", "<h2 class=\"m-h2\">"), "the list first, the add form below it", back=("S-09", "More")),
           "annotation": "One table answers who manages each outlet; the add form sits beside it and explains that the name must match the CSV outlet column, which is how imports find outlets."})
st.append({"name": "loading", "title": "Loading outlets",
           "desktop": desktop("admin", "S-06", outlets_head(0).replace('<span class="num">0</span> outlets.', "Loading outlets") + '<section aria-busy="true">%s</section><aside>%s</aside>' % (skel("100%", "12rem"), add_form()), cols=COLS),
           "mobile": mobile("admin", "S-06", "Outlets", '<div aria-busy="true">%s</div>' % skel("100%", "10rem"), "list placeholder", back=("S-09", "More")),
           "annotation": "The add form is usable while the list loads, since adding does not depend on it."})
st.append({"name": "empty", "title": "No outlets yet",
           "desktop": desktop("admin", "S-06", outlets_head(0) + '<section class="empty"><h2>No outlets yet</h2><p>Add each outlet once, using the same name your CSV files use. Then import reviews.</p></section><aside>%s</aside>' % add_form(), cols=COLS),
           "mobile": mobile("admin", "S-06", "Outlets", '<div class="empty"><h2>No outlets yet</h2><p>Add each outlet with the name your CSV files use.</p></div>' + add_form(), "invitation, then the form", back=("S-09", "More")),
           "annotation": "The first-run state explains the order of work: outlets first, then imports."})
st.append({"name": "error", "title": "Outlets could not load",
           "desktop": desktop("admin", "S-06", outlets_head(0) + '<section><p class="error" role="alert">Outlets could not load because the server did not answer. Check that OutletOwl is running, then reload.</p><button class="btn btn-secondary" type="button">Reload outlets</button></section><aside>%s</aside>' % add_form(), cols=COLS),
           "mobile": mobile("admin", "S-06", "Outlets", '<p class="error" role="alert">Outlets could not load because the server did not answer. Check that OutletOwl is running, then reload.</p>', "retry docked",
                            dock='<button class="btn btn-secondary btn-block" type="button">Reload outlets</button>', back=("S-09", "More")),
           "annotation": "Load failure copy follows the shared pattern; the form stays on desktop."})
st.append({"name": "partial", "title": "New outlet without a manager",
           "desktop": desktop("admin", "S-06", outlets_head(6) + '<section>%s</section><aside>%s</aside>' % (outlet_rows("Electronic City"), add_form()), cols=COLS),
           "mobile": mobile("admin", "S-06", "Outlets", outlets_m("Electronic City"), "the new row shows its missing manager", back=("S-09", "More")),
           "annotation": "An outlet added here has no manager until the users file names one; the row says so in words, because otherwise its reviews can never be replied to."})
st.append({"name": "duplicate-name", "title": "Outlet name already exists (409)",
           "desktop": desktop("admin", "S-06", outlets_head(5) + '<section>%s</section><aside>%s</aside>' % (outlet_rows(), add_form("koramangala", "An outlet named Koramangala already exists. Names are matched without regard to capitals, so use a different name.")), cols=COLS),
           "mobile": mobile("admin", "S-06", "Outlets", add_form("koramangala", "An outlet named Koramangala already exists. Use a different name."), "the form with its error first", back=("S-09", "More")),
           "annotation": "Outlet names are unique ignoring capitals (HLD review fix), so a near-duplicate is refused with the existing name shown."})
SCREENS.append(page("S-06", "Outlets", "Add outlets and see who manages each one.", "US-01-001", NAVD_ADMIN, NAVM_ADMIN, st,
                    revisions="default modal dialog for adding replaced by a form beside the table, so the admin sees the existing names while typing a new one"))

# ================================================================= S-07 import
REQUIRED = "outlet, source, date, rating, text, reviewer name"


def import_form(busy=False, error=""):
    btn = ('<button class="btn btn-primary" type="submit" aria-busy="true" disabled>Importing</button>' if busy else
           '<button class="btn btn-primary" type="submit">Import reviews</button>')
    inv = ' aria-invalid="true" aria-describedby="csv-err"' if error else ""
    return ('<form class="import-form" data-component="Import form"><label for="csv">CSV file</label><input id="csv" type="file" accept=".csv,text/csv"%s>'
            '<p class="error" id="csv-err"%s>%s</p><p class="small muted">Columns needed: %s. The outlet column must match an outlet name. '
            'Rows already imported are skipped, so you can import a corrected file again.</p>%s</form>') % (
        inv, ' role="alert"' if error else "", E(error), REQUIRED, btn)


def import_result(imported, skipped, rejected):
    rows = "".join('<tr><td class="num">%d</td><td>%s</td></tr>' % (n, E(why)) for n, why in rejected)
    rej = ('<h3>Rejected rows</h3><div class="table-scroll"><table class="data" data-component="Rejected rows"><thead><tr><th scope="col" class="num">Row</th><th scope="col">Reason</th></tr></thead><tbody>%s</tbody></table></div>'
           '<p class="small muted">Fix these rows in the file and import it again; rows already imported are skipped.</p>') % rows if rejected else ""
    return ('<section class="result" aria-labelledby="h-res" data-component="Import result"><h2 id="h-res">reviews-sept.csv imported</h2>'
            '<dl class="facts"><dt>Imported</dt><dd class="num">%d</dd><dt>Already imported, skipped</dt><dd class="num">%d</dd><dt>Rejected</dt><dd class="num">%d</dd></dl>'
            '<p>%s Tagging has started; <a href="%s">the overview</a> shows progress.</p>%s</section>') % (
        imported, skipped, len(rejected), badge("warn", "Tagging %d new reviews" % imported), link("S-02", "partial"), rej)


def import_head():
    return '<div class="span-all page-head"><h1>Import reviews</h1><p class="muted">Upload a CSV export of reviews. Google reviews will connect here later; only CSV is available now.</p></div>'


REJ = [(37, "Unknown outlet \"Koramangla\". Check the spelling against your outlet names."), (118, "Rating is \"4.5\"; ratings must be a whole number from 1 to 5."), (240, "Date is empty.")]
st = []
st.append({"name": "success", "title": "Imported, tagging started",
           "desktop": desktop("admin", "S-07", import_head() + '<section>%s</section><aside>%s</aside>' % (import_result(482, 0, []), import_form()), status="tagging", cols=COLS),
           "mobile": mobile("admin", "S-07", "Import", import_result(482, 0, []), "the result first; a new upload is docked", dock='<a class="btn btn-secondary btn-block" href="%s">Import another file</a>' % link("S-07", "ready"), status="tagging"),
           "annotation": "The result answers three counts the admin needs (imported, skipped, rejected) and says tagging started on its own (Q-022), with a link to watch progress."})
st.append({"name": "loading", "title": "Importing",
           "desktop": desktop("admin", "S-07", import_head() + '<section aria-busy="true"><p class="muted" aria-live="polite">Reading reviews-sept.csv and checking each row.</p>%s</section><aside>%s</aside>' % (skel("100%", "6rem"), import_form(busy=True)), cols=COLS),
           "mobile": mobile("admin", "S-07", "Import", '<p class="muted" aria-live="polite">Reading reviews-sept.csv and checking each row.</p>' + skel("100%", "6rem"), "progress line only",
                            dock='<button class="btn btn-primary btn-block" type="button" disabled aria-busy="true">Importing</button>'),
           "annotation": "The button is disabled while the request runs, and the request id makes a repeat return the same result anyway (tenet 8)."})
st.append({"name": "empty", "title": "No outlets yet",
           "desktop": desktop("admin", "S-07", import_head() + '<div class="span-all empty"><h2>Add an outlet first</h2><p>Each CSV row is matched to an outlet by name, so add your outlets before importing.</p><a class="btn btn-primary" href="%s">Add outlets</a></div>' % link("S-06", "empty"), cols=COLS),
           "mobile": mobile("admin", "S-07", "Import", '<div class="empty"><h2>Add an outlet first</h2><p>CSV rows are matched to outlets by name.</p></div>', "the one prerequisite",
                            dock='<a class="btn btn-primary btn-block" href="%s">Add outlets</a>' % link("S-06", "empty")),
           "annotation": "Importing with no outlets would reject every row, so the page sends the admin to add outlets first."})
st.append({"name": "error", "title": "File cannot be read (422)",
           "desktop": desktop("admin", "S-07", import_head() + '<section></section><aside>%s</aside>' % import_form(error="This file has no rating column, so nothing was imported. The first row must name these columns: %s." % REQUIRED), cols=COLS),
           "mobile": mobile("admin", "S-07", "Import", import_form(error="This file has no rating column, so nothing was imported. The first row must name: %s." % REQUIRED), "the form with its error"),
           "annotation": "A whole-file problem (wrong columns, not a CSV) imports nothing and says which column is missing; row problems are the partial state instead."})
st.append({"name": "partial", "title": "Imported with skipped and rejected rows",
           "desktop": desktop("admin", "S-07", import_head() + '<section>%s</section><aside>%s</aside>' % (import_result(479, 14, REJ), import_form()), status="tagging479", cols=COLS),
           "mobile": mobile("admin", "S-07", "Import", import_result(479, 14, REJ), "rejected rows as a two-column table", dock='<a class="btn btn-secondary btn-block" href="%s">Import corrected file</a>' % link("S-07", "ready"), status="tagging479"),
           "annotation": "Q-023 and the HLD review fix together: valid rows are kept, duplicates are counted as skipped, and each rejected row gives its number and a reason the admin can act on."})
st.append({"name": "ready", "title": "Ready to import",
           "desktop": desktop("admin", "S-07", import_head() + '<section class="howto"><h2>Before you import</h2><p>Export reviews as CSV with one row per review. Dates may be any day; reviews are grouped into Monday to Sunday weeks in India time.</p></section><aside>%s</aside>' % import_form(), cols=COLS),
           "mobile": mobile("admin", "S-07", "Import", import_form().replace('<button class="btn btn-primary" type="submit">Import reviews</button>', ""), "the file field first; the button docks",
                            dock='<button class="btn btn-primary btn-block" type="submit">Import reviews</button>'),
           "annotation": "The starting state: one file field with the required columns spelled out and a note that weeks are counted in India time (HLD review fix)."})
SCREENS.append(page("S-07", "Import reviews", "Upload a CSV of reviews and see what was imported, skipped and rejected.", "US-01-002, US-01-003, US-01-004", NAVD_ADMIN, NAVM_ADMIN, st,
                    revisions="default drag-and-drop hero zone replaced by a plain labelled file field; result shown as three counts and a rejected-rows table instead of a toast"))

# ================================================================= S-08 digest
def email_preview(partial=False):
    mv = movers()[:5]
    top = mv[0]
    first = ("%d reviews from these two weeks are not tagged yet; movers and urgent reviews may be incomplete." % UNTAGGED) if partial else ""
    lines = "".join("<li>%s, %s: %d to %d (%s %d)</li>" % (E(o), E(t).lower(), a, b, "up" if ch > 0 else "down", abs(ch)) for o, t, a, b, ch in mv)
    urg = "".join('<li lang="%s">%s, %s, %s: %s</li>' % (u["lang"], E(u["kind"]), E(u["outlet"]), E(u["date"]), E(u["text"])) for u in URGENT)
    return ('<div class="email" data-component="Digest e-mail preview"><p class="small muted">To: %s<br>Subject: Weekly digest, %s: %s %s up %d</p>'
            '%s<p><strong>Biggest mover: %s, %s.</strong> Negative reviews %d to %d.</p><h3>Movers</h3><ul>%s</ul><h3>Urgent reviews (%d)</h3><ul>%s</ul>'
            '<p class="small muted">Sent by OutletOwl for %s. Generated on demand.</p></div>') % (
        ADMIN[1], WEEK, E(top[0]), E(top[1]).lower(), top[4],
        ('<p class="email-warn"><strong>%s</strong></p>' % E(first)) if first else "",
        E(top[0]), E(top[1]).lower(), top[2], top[3], lines, len(URGENT), urg, E(BRAND))


def digest_head():
    return '<div class="span-all page-head"><h1>Weekly digest</h1><p class="muted">Covers the latest complete week, %s, compared with %s. Sent to %s.</p></div>' % (WEEK, PREV_WEEK, ADMIN[1])


GEN = '<button class="btn btn-primary" type="button">Generate and send digest</button>'
st = []
st.append({"name": "success", "title": "Digest sent",
           "desktop": desktop("admin", "S-08", digest_head() + '<section><p>%s Sent to %s at 10:42 on 6 Oct 2026. Open MailHog to see it as received.</p>%s</section>'
                              '<aside><h2>Send again</h2><p class="small muted">A new digest is a new e-mail. Pressing again for the same request never sends twice.</p><button class="btn btn-secondary" type="button">Generate a new digest</button></aside>' % (badge("ok", "Sent"), ADMIN[1], email_preview()), cols=COLS),
           "mobile": mobile("admin", "S-08", "Digest", '<p>%s Sent at 10:42.</p>%s' % (badge("ok", "Sent"), email_preview()), "confirmation line, then the e-mail as sent", back=("S-09", "More")),
           "annotation": "After sending, the page shows the e-mail exactly as composed, so the admin checks what went out without leaving OutletOwl."})
st.append({"name": "loading", "title": "Sending",
           "desktop": desktop("admin", "S-08", digest_head() + '<section aria-busy="true"><p class="muted" aria-live="polite">Building the digest and sending it to MailHog.</p>%s</section><aside><button class="btn btn-primary" type="button" disabled aria-busy="true">Sending digest</button></aside>' % skel("100%", "14rem"), cols=COLS),
           "mobile": mobile("admin", "S-08", "Digest", '<p class="muted" aria-live="polite">Building the digest and sending it.</p>' + skel("100%", "12rem"), "progress line",
                            dock='<button class="btn btn-primary btn-block" type="button" disabled aria-busy="true">Sending digest</button>', back=("S-09", "More")),
           "annotation": "The digest row is claimed as sending before the e-mail goes out (HLD review fix), so a double press shows this state instead of a second e-mail."})
st.append({"name": "empty", "title": "No complete week yet",
           "desktop": desktop("admin", "S-08", '<div class="span-all page-head"><h1>Weekly digest</h1></div><div class="span-all empty"><h2>No complete week of reviews yet</h2><p>The digest compares the latest complete Monday to Sunday week with the week before. Import reviews covering at least two complete weeks.</p><a class="btn btn-primary" href="%s">Import reviews</a></div>' % link("S-07"), cols=COLS),
           "mobile": mobile("admin", "S-08", "Digest", '<div class="empty"><h2>No complete week yet</h2><p>The digest needs two complete weeks of reviews.</p></div>', "invitation",
                            dock='<a class="btn btn-primary btn-block" href="%s">Import reviews</a>' % link("S-07"), back=("S-09", "More")),
           "annotation": "Without two complete weeks there is nothing to compare, so the page explains the rule from Q-005 instead of sending an empty e-mail."})
st.append({"name": "error", "title": "Not sent: mail catcher down",
           "desktop": desktop("admin", "S-08", digest_head() + '<section><p class="error" role="alert">The digest was not sent because the mail catcher (MailHog) did not answer. Start MailHog, then generate the digest again.</p>%s</section>' % GEN, cols=COLS),
           "mobile": mobile("admin", "S-08", "Digest", '<p class="error" role="alert">The digest was not sent because MailHog did not answer. Start MailHog, then try again.</p>', "retry docked",
                            dock='<button class="btn btn-primary btn-block" type="button">Generate and send digest</button>', back=("S-09", "More")),
           "annotation": "An SMTP failure names the service to start; the row is marked failed so the next press sends a new digest."})
st.append({"name": "partial", "title": "Digest with untagged reviews",
           "desktop": desktop("admin", "S-08", digest_head() + '<section><p>%s Sent at 10:42 with a warning line.</p>%s</section><aside>%s</aside>' % (
               badge("warn", "Sent, incomplete"), email_preview(partial=True), notice("warn", "%d reviews were not tagged when this was sent." % UNTAGGED, "The e-mail's first line says so.")), status="partial", cols=COLS),
           "mobile": mobile("admin", "S-08", "Digest", '<p>%s</p>%s' % (badge("warn", "Sent, incomplete"), email_preview(partial=True)), "the warning line is the e-mail's first line", back=("S-09", "More"), status="partial"),
           "annotation": "The HLD review fix: a digest built while reviews are untagged says so on its first line, so a missing urgent review is never silent."})
st.append({"name": "ready", "title": "Ready to send",
           "desktop": desktop("admin", "S-08", digest_head() + '<section><h2>What it will contain</h2><ul><li>The outlet and theme that moved most, and the next four movers</li><li>Every urgent review dated in the week (%d now)</li><li>A warning line if any review in the two weeks is not tagged</li></ul></section><aside>%s<p class="small muted">Sends one e-mail to %s through MailHog.</p></aside>' % (len(URGENT), GEN, ADMIN[1]), cols=COLS),
           "mobile": mobile("admin", "S-08", "Digest", '<h2>What it will contain</h2><ul><li>Top mover and four more</li><li>Urgent reviews (%d)</li><li>A warning if reviews are untagged</li></ul>' % len(URGENT), "contents list, button docked",
                            dock='<button class="btn btn-primary btn-block" type="button">Generate and send digest</button>', back=("S-09", "More")),
           "annotation": "Before sending, the page lists what the e-mail will contain and who gets it, so on demand never means a surprise."})
SCREENS.append(page("S-08", "Weekly digest", "Generate the weekly digest on demand and check what was sent.", "US-01-009", NAVD_ADMIN, NAVM_ADMIN, st,
                    revisions="default success toast replaced by the full e-mail preview on the page; a schedule picker deliberately absent (Q-005)"))

# ================================================================= S-09 more (phone only)
more = ('<ul class="m-list more" data-component="More menu"><li><a href="%s">%sOutlets</a></li><li><a href="%s">%sWeekly digest</a></li>'
        '<li><a href="%s">Sign out</a></li></ul><p class="small muted">Signed in as %s, brand admin.</p>') % (
    link("S-06"), IC["outlets"], link("S-08", "ready"), IC["digest"], link("S-01"), ADMIN[0])
st = [{"name": "success", "title": "More menu (phone)", "mobile": mobile("admin", "S-09", "More", more, "the admin's less frequent destinations"),
       "annotation": "At 375 there is room for five tabs, so Outlets, Digest and Sign out live here; on desktop they are in the side nav and this page does not exist."}]
SCREENS.append(page("S-09", "More", "Reach the admin's less frequent pages on a phone.", "US-01-001, US-01-009", "not used on desktop (side nav holds these)", NAVM_ADMIN, st,
                    na=[("loading", "a static menu with no data"), ("empty", "the menu always has its three entries"), ("error", "no request is made"), ("partial", "no request is made")],
                    platform="mobile"))

# ================================================================= feature index (redlines)
REDLINES = {
    "S-01": [("Page", "Sign-in page", "headline --text-hero 700, tracking --tracking-title; gradient words", "--gradient-navy, --gradient-highlight-dark, --color-streak-1 to 3", "page --space-7, intro to card --space-6"),
             ("Sign-in form", "Sign-in form", "label 14/600, body 16", "--color-surface card, --elevation-hero, --color-border-strong fields, --color-accent pill with --glow-primary", "card --space-5, field gap --space-3, --radius-overlay")],
    "S-02": [("Navigation", "Navigation", "nav 15/600", "--gradient-navy rail, --color-on-navy-muted, --color-accent active pill", "rail 248 px, items 44 px"),
             ("Movers sentence", "Page", "--text-title 700, numbers --text-stat 800 in --gradient-highlight-light", "--color-text", "--space-3 below"),
             ("Movers table", "Movers table", "header --text-label 600 uppercase, figures tabular", "--color-border dividers, --color-row-hover, --color-danger up, --color-success down", "cell --space-half x --space-3"),
             ("Urgent panel", "Urgent list", "badge --text-label 600 pill", "--color-danger, --color-danger-subtle", "card --space-4, --radius-container"),
             ("Outlet comparison", "Outlet comparison", "body 16, figures tabular", "--color-chart-line, --color-chart-mark", "sparkline 88 x 32")],
    "S-03": [("Heatmap", "Theme heatmap", "cell figures 700 tabular", "--color-heat-0 to --color-heat-5 with --color-heat-N-text", "cell min 48 px, 4 px spacing, --radius-control")],
    "S-04": [("Filters", "Review filters", "label 14/600", "--color-bg-subtle panel, --color-border-strong fields", "grid gap --space-3, min column 10rem"),
             ("Review table", "Review table", "review text body 16, clamp 2 lines", "--color-text link, --color-danger urgent, --color-bg-subtle chips", "cell --space-half x --space-3")],
    "S-05": [("Review", "Review", "review text --text-title, Devanagari line height 1.8", "--color-surface card", "card --space-4"),
             ("Reply editor", "Reply editor", "textarea body 16", "--color-accent with --glow-primary on Mark as replied only", "actions gap --space-2")],
    "S-06": [("Outlet table", "Outlet table", "figures tabular", "--color-warning for No manager", "cell --space-half x --space-3"),
             ("Add outlet", "Add outlet form", "label 14/600", "--color-danger on 409", "card --space-4")],
    "S-07": [("Import form", "Import form", "label 14/600", "--color-danger on 422", "card --space-4"),
             ("Result", "Import result", "counts --text-stat 800", "--color-warning tagging badge", "three stat columns, gap --space-4")],
    "S-08": [("E-mail preview", "Digest e-mail preview", "body 16", "--color-surface card, --color-bg-subtle header strip, --color-warning-subtle warning line", "card --space-5")],
    "S-09": [("More menu", "More menu", "body 16/600", "--color-surface list card, --color-border dividers", "row min 56 px")],
}

INDEX_CSS = """
  *, *::before, *::after { box-sizing: border-box; }
  body { margin: 0; background: var(--color-bg); color: var(--color-text); font-family: var(--font-body); font-size: var(--text-body); line-height: var(--leading-body); }
  .hero { background: var(--gradient-navy); color: var(--color-on-navy); padding: var(--space-6) var(--space-3) var(--space-7); }
  .hero a { color: var(--color-on-navy-muted); }
  .hero .muted, .hero dt { color: var(--color-on-navy-muted); }
  .hero :focus-visible { outline-color: var(--color-focus-on-navy); }
  .wrap { max-width: 1280px; margin-inline: auto; }
  main.wrap { padding: 0 var(--space-3) var(--space-6); margin-top: calc(-1 * var(--space-6)); }
  @media (min-width: 768px) { .hero { padding-inline: var(--space-5); } main.wrap { padding-inline: var(--space-5); } }
  h1 { font-size: var(--text-hero); font-weight: var(--weight-stat); letter-spacing: var(--tracking-title); line-height: 1.1; margin: 0 0 var(--space-3); }
  h2 { font-size: var(--text-display); font-weight: var(--weight-title); letter-spacing: var(--tracking-title); margin: 0 0 var(--space-2); }
  h3 { font-size: var(--text-label); text-transform: uppercase; letter-spacing: .08em; color: var(--color-text-muted); margin: var(--space-4) 0 var(--space-2); }
  .hl { background: var(--gradient-highlight-dark); -webkit-background-clip: text; background-clip: text; color: transparent; }
  p { margin: 0 0 var(--space-2); max-width: 72ch; }
  .muted { color: var(--color-text-muted); }
  a { color: var(--color-link); }
  .summary { display: grid; gap: var(--space-2) var(--space-4); grid-template-columns: max-content 1fr; margin: var(--space-4) 0 0; font-size: var(--text-label); }
  .summary > :nth-child(even) { margin: 0; }
  .screen { background: var(--color-surface); border: 1px solid var(--color-border); border-radius: var(--radius-overlay); box-shadow: var(--elevation-1);
    padding: var(--space-4); margin-bottom: var(--space-4); transition: transform var(--duration-base) var(--ease-standard), box-shadow var(--duration-base) var(--ease-standard); }
  .screen:hover { transform: var(--lift); box-shadow: var(--elevation-2); }
  @media (min-width: 768px) { .screen { padding: var(--space-5); } }
  .states { display: flex; flex-wrap: wrap; gap: var(--space-2); padding: 0; margin: 0; list-style: none; }
  .states a { display: inline-flex; align-items: center; min-height: 44px; padding: 0 var(--space-3); border-radius: var(--radius-pill); text-decoration: none;
    background: var(--color-bg-subtle); color: var(--color-text); font-weight: var(--weight-label); font-size: var(--text-label); transition: background var(--duration-fast), color var(--duration-fast); }
  .states a:hover { background: var(--color-accent); color: var(--color-on-accent); }
  .shots { display: grid; gap: var(--space-3); grid-template-columns: 1fr; align-items: start; }
  @media (min-width: 1100px) { .shots { grid-template-columns: 375fr 768fr 1440fr; } }
  .shots figure { margin: 0; }
  .shots figcaption { color: var(--color-text-muted); font-size: var(--text-label); font-weight: var(--weight-label); margin-bottom: var(--space-2); }
  .shots img, .shots iframe { width: 100%; border: 1px solid var(--color-border); border-radius: var(--radius-control); background: var(--color-bg-subtle); display: block; box-shadow: var(--elevation-1); }
  .table-wrap { overflow-x: auto; }
  table { border-collapse: collapse; width: 100%; font-size: var(--text-label); }
  th, td { text-align: left; vertical-align: top; padding: var(--space-half) var(--space-3); border-bottom: 1px solid var(--color-border); }
  th { color: var(--color-text-muted); font-weight: var(--weight-label); }
  tbody tr:hover { background: var(--color-row-hover); }
  code { font-family: var(--font-mono); font-size: var(--text-label); }
"""


def write_index(screens):
    blocks = []
    for s in screens:
        sid = s["key"].split("/")[1]
        f = FILES[sid]
        base = f[:-5]
        states = "".join('<li><a href="%s#state=%s">%s</a></li>' % (f, n, n) for n in s["states"])
        shots = "".join(
            '<figure><figcaption>%s</figcaption><img src="shots/%s-%s.png" alt="%s at %s px" loading="lazy" '
            'onerror="this.replaceWith(Object.assign(document.createElement(\'iframe\'),{src:\'%s#chrome=0\',title:\'%s\',loading:\'lazy\'}))"></figure>'
            % (w, base, w, E(s["name"]), w, f, E(s["name"])) for w in ("375", "768", "1440"))
        rows = "".join("<tr><td>%s</td><td><code>%s</code></td><td>%s</td><td><code>%s</code></td><td>%s</td></tr>" % tuple(E(x) for x in r) for r in REDLINES[sid])
        blocks.append(
            '<section class="screen" id="%s" aria-labelledby="h-%s"><h2 id="h-%s">%s %s</h2><p>%s <span class="muted">Serves %s.</span></p>'
            '<h3>States</h3><ul class="states">%s</ul><h3>At three widths</h3><div class="shots">%s</div>'
            '<h3>Redlines</h3><div class="table-wrap"><table><thead><tr><th>Region</th><th>Component</th><th>Type role</th><th>Tokens</th><th>Spacing</th></tr></thead><tbody>%s</tbody></table></div></section>'
            % (sid, sid, sid, sid, E(s["name"]), E(s["purpose"]), E(", ".join(s["serves"])), states, shots, rows))
    n_states = sum(len(s["states"]) for s in screens)
    summary = [
        ("Inventory", "stories in docs/product/backlog.md, the PRD and the HLD (no flows file; navigation proposed and confirmed by the product owner)"),
        ("Tokens", "docs/design/tokens.css, proposed in round 2 (Night launch, light only)"),
        ("Framework", "HTML bundle now; React and Vite views later (ADR-0002); contract: no contract"),
        ("Events", "no sheet"),
        ("Screenshots", "shots/, taken with evidence.py (browse) at 375, 768 and 1440"),
        ("Screens", "%d, states rendered %d" % (len(screens), n_states)),
    ]
    dl = "".join("<dt>%s</dt><%s>%s</%s>" % (E(k), "d" + "d", E(v), "d" + "d") for k, v in summary)
    out = ('<!doctype html>\n<html lang="en">\n<head>\n<meta charset="utf-8">\n<meta name="viewport" content="width=device-width, initial-scale=1">\n'
           '<meta name="color-scheme" content="light">\n<title>app: screens, round 2</title>\n'
           '<!-- Per-feature index, generated by build.py. The product-wide gallery is ../../index.html, rendered by bundle.py from design.json. -->\n'
           '<link rel="preconnect" href="https://fonts.googleapis.com">\n<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>\n'
           '<link rel="stylesheet" href="%s">\n<link rel="stylesheet" href="../../tokens.css">\n<style>%s</style>\n</head>\n<body>\n'
           '<header class="hero"><div class="wrap"><p class="muted"><a href="../../index.html">All screens</a> / app</p>\n<h1>app: screens, <span class="hl">round 2</span></h1>\n'
           '<p class="muted">One prototype per screen. Open a state link to see that state; #state=all shows every state as the whole screen, desktop beside the designed 375 px arrangement. Comment per screen and per state in the session; each round is recorded in CHANGES.md.</p>\n'
           '<dl class="summary">%s</dl></div></header>\n<main class="wrap">\n%s\n</main>\n</body>\n</html>\n') % (E(FONTS), INDEX_CSS, dl, "\n".join(blocks))
    with open(os.path.join(HERE, "index.html"), "w", encoding="utf-8") as fh:
        fh.write(out)


if __name__ == "__main__":
    import json
    write_index(SCREENS)
    json.dump(SCREENS, open(os.path.join(HERE, "screens.json"), "w"), indent=1, ensure_ascii=False)
    print("built %d screens, %d states" % (len(SCREENS), sum(len(s["states"]) for s in SCREENS)))
