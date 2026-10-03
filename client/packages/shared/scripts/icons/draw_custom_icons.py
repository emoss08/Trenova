import glob, json, math, os, re, sys

HERE = os.path.dirname(os.path.abspath(__file__))
UNTITLED_DIST = sys.argv[1] if len(sys.argv) > 1 else os.path.join(HERE, "..", "..", "node_modules", "@untitledui", "icons", "dist")


def load_untitled(dist):
    icons = {}
    for path in glob.glob(os.path.join(dist, "*.mjs")):
        name = os.path.basename(path)[:-4]
        if name == "index":
            continue
        src = open(path).read()
        icons[name] = [dict(re.findall(r'(\w+):"([^"]*)"', obj)) for obj in re.findall(r'createElement\("path",\{(.*?)\}\)', src)]
    if not icons:
        sys.exit(f"no Untitled UI icons found in {dist}; pass the package's dist directory")
    return icons


U = load_untitled(UNTITLED_DIST)


def f(v):
    s = f"{round(v, 3):.3f}".rstrip("0").rstrip(".")
    return "0" if s in ("-0", "") else s


def P(*pts):
    return " ".join(f"{f(x)} {f(y)}" for x, y in pts)


def corner(c, d1, d2, r):
    cx, cy = c
    s = (cx - r * d1[0], cy - r * d1[1])
    c1 = (s[0] + 0.35 * r * d1[0], s[1] + 0.35 * r * d1[1])
    c2 = (s[0] + 0.525 * r * d1[0], s[1] + 0.525 * r * d1[1])
    p1 = (s[0] + 0.659 * r * d1[0] + 0.068 * r * d2[0], s[1] + 0.659 * r * d1[1] + 0.068 * r * d2[1])
    p2 = (cx - 0.068 * r * d1[0] + 0.341 * r * d2[0], cy - 0.068 * r * d1[1] + 0.341 * r * d2[1])
    q1 = (cx + 0.475 * r * d2[0], cy + 0.475 * r * d2[1])
    q2 = (cx + 0.65 * r * d2[0], cy + 0.65 * r * d2[1])
    e = (cx + r * d2[0], cy + r * d2[1])
    cross = d1[0] * d2[1] - d1[1] * d2[0]
    sweep = 1 if cross > 0 else 0
    ar = 0.625 * r
    return s, f"L{P(s)} C{P(c1, c2, p1)} A{f(ar)} {f(ar)} 0 0 {sweep} {P(p2)} C{P(q1, q2, e)}"


def unit(a, b):
    dx, dy = b[0] - a[0], b[1] - a[1]
    n = math.hypot(dx, dy)
    return (dx / n, dy / n)


def rpath(pts, r, closed=False):
    """Polyline with Untitled-style squircle corners. r may be a number or a per-vertex list."""
    n = len(pts)
    rs = r if isinstance(r, list) else [r] * n
    if closed:
        d1 = unit(pts[-1], pts[0])
        d2 = unit(pts[0], pts[1])
        r0 = rs[0]
        start = (pts[0][0] + r0 * d2[0], pts[0][1] + r0 * d2[1]) if r0 else pts[0]
        out = f"M{P(start)}"
        seq = list(range(1, n)) + [0]
        for i in seq:
            a, b, c = pts[i - 1], pts[i], pts[(i + 1) % n]
            if rs[i]:
                _, seg = corner(b, unit(a, b), unit(b, c), rs[i])
                out += " " + seg
            else:
                out += f" L{P(b)}"
        return out + "Z"
    out = f"M{P(pts[0])}"
    for i in range(1, n - 1):
        a, b, c = pts[i - 1], pts[i], pts[i + 1]
        if rs[i]:
            _, seg = corner(b, unit(a, b), unit(b, c), rs[i])
            out += " " + seg
        else:
            out += f" L{P(b)}"
    return out + f" L{P(pts[-1])}"


def rect(x0, y0, x1, y1, r):
    return rpath([(x0, y0), (x1, y0), (x1, y1), (x0, y1)], r, closed=True)


def circ(cx, cy, r):
    return f"M{f(cx - r)} {f(cy)}a{f(r)} {f(r)} 0 1 0 {f(2 * r)} 0a{f(r)} {f(r)} 0 1 0 {f(-2 * r)} 0Z"


def dot(cx, cy):
    return circ(cx, cy, 1)


def arc(cx, cy, r, a0, a1):
    """Arc from angle a0 to a1 (degrees, clockwise on screen, 0 = east)."""
    p0 = (cx + r * math.cos(math.radians(a0)), cy + r * math.sin(math.radians(a0)))
    p1 = (cx + r * math.cos(math.radians(a1)), cy + r * math.sin(math.radians(a1)))
    large = 1 if (a1 - a0) % 360 > 180 else 0
    return f"M{P(p0)} A{f(r)} {f(r)} 0 {large} 1 {P(p1)}"


def u(name):
    return " ".join(p["d"] for p in U[name])


def paths(*ds):
    return "".join(f'<path d="{d}"/>' for d in ds if d)


SHIELD = U["Shield01"][0]["d"]
CAL_TOP = "M21 10H3m13-8v4M8 2v4"
FILE_LINES = "M14 11H8m2 4H8m8-8H8"
SCAN_CORNERS = u("Scan").split("M3 12h.01")[0]
SEARCH_LENS = "m21 21-3.5-3.5m2.5-6a8.5 8.5 0 1 1-17 0 8.5 8.5 0 0 1 17 0Z"
CLIPBOARD = u("Clipboard")
BRAIN_LEFT = "M12 5.2A2.9 2.9 0 0 0 6.7 3.9 3 3 0 0 0 3.9 7.6 3.1 3.1 0 0 0 2.6 12.4 3.1 3.1 0 0 0 4 17.2 3 3 0 0 0 7.6 20.4 2.8 2.8 0 0 0 12 19.8"
BRAIN_RIGHT = "M12 5.2A2.9 2.9 0 0 1 17.3 3.9 3 3 0 0 1 20.1 7.6 3.1 3.1 0 0 1 21.4 12.4 3.1 3.1 0 0 1 20 17.2 3 3 0 0 1 16.4 20.4 2.8 2.8 0 0 1 12 19.8"
BRAIN_LEFT_SULCI = "M6.7 3.9c.2 1.7 1.4 2.8 3 3M3.9 7.6c1.6.2 2.7 1.2 3 2.7M2.6 12.4c1.5-.6 3.2-.4 4.4.8M4 17.2c1.3-.9 1.9-2.2 1.8-3.6M7.6 20.4c.1-1.6 1-2.8 2.5-3.3"


def file_open():
    return rpath([(12, 22), (4, 22), (4, 2), (20, 2), (20, 12.5)], 4.8)


def calendar_open():
    return CAL_TOP + " " + rpath([(11, 22), (3, 22), (3, 4), (21, 4), (21, 11)], 4.8)


def mail_open():
    return "m2 7 8.165 5.715c.661.463.992.695 1.351.784a2 2 0 0 0 .968 0c.36-.09.69-.32 1.351-.784L22 7 " + rpath(
        [(12, 20), (2, 20), (2, 4), (22, 4), (22, 12)], 4.8
    )


def clock_badge(cx, cy, r=4.5):
    return circ(cx, cy, r) + f" M{f(cx)} {f(cy - 2.2)}V{f(cy)}l1.6 1"


def polar(a, d, c=(12, 12.5)):
    return (c[0] + d * math.cos(math.radians(a)), c[1] + d * math.sin(math.radians(a)))


def gavel_head():
    c = (9, 9)
    a = (1 / math.sqrt(2), -1 / math.sqrt(2))
    b = (1 / math.sqrt(2), 1 / math.sqrt(2))
    L, W = 6.4, 3
    pts = [
        (c[0] + s1 * L * a[0] + s2 * W * b[0], c[1] + s1 * L * a[1] + s2 * W * b[1])
        for s1, s2 in ((-1, -1), (1, -1), (1, 1), (-1, 1))
    ]
    return rpath(pts, 1.6, closed=True)


def dashed_circle(cx, cy, r, n=8, gap=14):
    seg = 360 / n
    return " ".join(arc(cx, cy, r, i * seg + gap / 2 - 90, (i + 1) * seg - gap / 2 - 90) for i in range(n))


C = {}

C["ShieldAlert"] = paths("M12 8v4m0 4h.01", SHIELD)
C["ShieldX"] = paths("m14.5 9-5 5m0-5 5 5", SHIELD)
C["ShieldQuestion"] = paths("M9.6 9.3a2.5 2.5 0 0 1 4.85.85c0 1.65-2.45 2.35-2.45 2.35m.05 3.3h.01", SHIELD)
C["ShieldBan"] = paths(circ(12, 11.5, 3.75), "m9.35 8.85 5.3 5.3", SHIELD)
OCTAGON = "M2 8.523" + u("AlertOctagon").split("M2 8.523", 1)[1]
C["XOctagon"] = paths("m15 9-6 6m0-6 6 6", OCTAGON)
C["CheckDouble"] = paths("M2 12.5 7 17.5 18 6.5", "m13.5 16 1.5 1.5L22 10.5")
C["Pause"] = paths(rect(5.5, 4, 9.5, 20, 1.6), rect(14.5, 4, 18.5, 20, 1.6))
C["Spinner"] = paths("M12 2a10 10 0 1 0 10 10")
C["GripVertical"] = paths(*(dot(x, y) for x in (9, 15) for y in (5, 12, 19)))
C["Dot"] = paths(dot(12, 12))
C["CircleDot"] = paths(circ(12, 12, 10), circ(12, 12, 2))
C["CircleDashed"] = paths(dashed_circle(12, 12, 10))
C["ChevronCollapseVertical"] = paths("m7 20 5-5 5 5M7 4l5 5 5-5")
C["CalendarClock"] = paths(calendar_open(), clock_badge(17.5, 17.5))
C["CalendarRange"] = paths(u("Calendar"), "M7 14h5m-5 4h2m4 0h4m-3-4h3")
C["CalendarSync"] = paths(
    calendar_open(),
    "M14.6 17.3a3 3 0 0 1 5.4-1.5m.4-2.3v2.3h-2.3M21.4 18.7a3 3 0 0 1-5.4 1.5m-.4 2.3v-2.3h2.3",
)
C["ClockAlert"] = paths(arc(12, 12, 10, 25, 335), "M12 6v6l3 1.5", "M21 10.5v3.5m0 3.8h.01")
C["FileAlert"] = paths(file_open(), FILE_LINES, "M18 14.5v3m0 3.5h.01")
C["FileClock"] = paths(file_open(), FILE_LINES, clock_badge(17.5, 17.5, 4))
C["FileEdit"] = paths(
    file_open(),
    FILE_LINES,
    "M13.5 21.5v-1.8c0-.34 0-.51.04-.67a1.5 1.5 0 0 1 .17-.41c.09-.14.21-.26.45-.5l4.34-4.34a1.5 1.5 0 0 1 2.12 2.12l-4.34 4.34c-.24.24-.36.36-.5.45a1.5 1.5 0 0 1-.41.17c-.16.04-.33.04-.67.04Z",
)
C["FileSpreadsheet"] = paths(u("File04"), rect(8, 12, 16, 18, 0), "M12 12v6M8 15h8")
C["FileUpload"] = paths(file_open(), FILE_LINES, "m15 18 3-3m0 0 3 3m-3-3v6")
C["FilterFunnelX"] = paths(
    "M13.5 11.6 14.4 10.8l6.396-5.543c.203-.176.305-.264.378-.37a1 1 0 0 0 .14-.31c.034-.125.034-.26.034-.528V4.6c0-.56 0-.84-.109-1.054a1 1 0 0 0-.437-.437C20.24 3 19.96 3 19.4 3H3.6c-.56 0-.84 0-1.054.109a1 1 0 0 0-.437.437C2 3.76 2 4.04 2 4.6v.67c0 .268 0 .403.033.528a1 1 0 0 0 .141.31c.073.106.175.194.378.37l6.396 5.543c.203.177.305.265.378.371a1 1 0 0 1 .141.31c.033.125.033.26.033.529v6.586c0 .397 0 .595.083.72a.5.5 0 0 0 .315.214c.148.03.332-.043.7-.19L12 19.6",
    "m16 15 5 5m0-5-5 5",
)
C["ArchiveRestore"] = paths(u("Archive").replace("M10 13h4", "M12 18v-6m-3 3 3-3 3 3"))
C["ArchiveX"] = paths(u("Archive").replace("M10 13h4", "M10 12l4 4m0-4-4 4"))
C["ListChecks"] = paths("M13 6h8M13 12h8M13 18h8", "m3 6 1.75 1.75L8.5 4", "m3 16 1.75 1.75L8.5 14")
C["ListOrdered"] = paths(
    "M10 6h11M10 12h11M10 18h11",
    "M4 4h1.5v5M4 9h3",
    "M7 20H4c0-1 3-1.8 3-3.25a1.5 1.5 0 0 0-3-.25",
)
C["ListPlus"] = paths("M3 6h14M3 12h14M3 18h8", "M18 15v6m-3-3h6")
C["ListX"] = paths("M3 6h14M3 12h14M3 18h8", "m15.5 15.5 5 5m0-5-5 5")
C["SortAscending"] = paths("M6 4v16m0 0-3-3m3 3 3-3", "M13 6h3M13 12h5.5M13 18h8")
C["SortDescending"] = paths("M6 4v16m0 0-3-3m3 3 3-3", "M13 6h8M13 12h5.5M13 18h3")
C["ArrowLeftToLine"] = paths("M4 5v14", "M20 12H8m0 0 6 6m-6-6 6-6")
C["ArrowRightToLine"] = paths("M20 5v14", "M4 12h12m0 0-6 6m6-6-6-6")
C["Plug"] = paths("M9 2v5m6-5v5", "M6 7h12v3.5a6 6 0 0 1-12 0V7Z", "M12 16.5V22")
C["PlugOff"] = paths("M9 2v5m6-5v5", "M10 7h8v3.5c0 1.2-.35 2.32-.96 3.26M14.5 16a6 6 0 0 1-8.5-5.5V7", "M12 16.5V22", "M3 3l18 18")
C["PowerOff"] = paths("M12 2v6", "M18.36 6.64a9 9 0 0 1 2.4 8.26M17.9 18.9A9 9 0 0 1 5.64 6.64", "M3 3l18 18")
C["PinOff"] = paths(
    "M12 15v7",
    "M9.5 15H7.33c-1.066 0-1.599 0-1.873-.219A1 1 0 0 1 5.08 14c0-.35.333-.766 1-1.599l1.569-1.962c.13-.162.195-.243.241-.334a1 1 0 0 0 .09-.254C8 9.752 8 9.648 8 9.44V8",
    "M8.364 2h7.273c.792 0 1.188 0 1.44.167a1 1 0 0 1 .427.63c.06.295-.086.662-.38 1.397l-1.008 2.52a2 2 0 0 0-.08.215 1 1 0 0 0-.028.15C16 7.135 16 7.193 16 7.309V9.44c0 .208 0 .312.02.411a1 1 0 0 0 .09.254c.046.091.11.172.24.334l1.57 1.962c.15.187.282.353.393.5",
    "M3 3l18 18",
)
C["MarkerPinOff"] = paths(
    "M7.3 3.6A8 8 0 0 1 20 10c0 1.6-.53 3.1-1.38 4.52M16.4 17.4C15.04 18.96 13.5 20.5 12 22c-4-4-8-7.582-8-12 0-1.3.31-2.53.86-3.62",
    "M3 3l18 18",
)
C["SearchX"] = paths(SEARCH_LENS, "m14 9-5 5m0-5 5 5")
C["SearchCheck"] = paths(SEARCH_LENS, "m8.5 11.5 2 2 4-4")
C["ScanSearch"] = paths(SCAN_CORNERS, circ(11.5, 11.5, 3.5), "m17 17-3-3")
C["ScanText"] = paths(SCAN_CORNERS, "M7.5 8.5h9M7.5 12h9M7.5 15.5h5")
C["ServerAlert"] = paths(
    "M6 6h.01",
    rect(2, 2, 22, 10, 3.2),
    "m13 13-3 4.5h4L11 22",
    "M5 14H5.2c-1.12 0-1.68 0-2.108.218a2 2 0 0 0-.874.874C2 15.52 2 16.08 2 17.2v1.6c0 1.12 0 1.68.218 2.108a2 2 0 0 0 .874.874C3.52 22 4.08 22 5.2 22H7M17 22h1.8c1.12 0 1.68 0 2.108-.218a2 2 0 0 0 .874-.874C22 20.48 22 19.92 22 18.8v-1.6c0-1.12 0-1.68-.218-2.108a2 2 0 0 0-.874-.874C20.48 14 19.92 14 18.8 14H17",
)
C["MailCheck"] = paths(mail_open(), "m15 18 2 2 4.5-4.5")
C["MailAlert"] = paths(mail_open(), "M18 14.5v3m0 3.5h.01")
C["ClipboardList"] = paths(CLIPBOARD, "M9 11h6M9 15h6M9 19h3")
C["ClipboardEdit"] = paths(
    CLIPBOARD,
    "M9 19v-1.6c0-.3 0-.45.035-.59a1.2 1.2 0 0 1 .144-.348c.076-.124.182-.23.394-.442l3.977-3.977a1.414 1.414 0 0 1 2 2l-3.977 3.977c-.212.212-.318.318-.442.394a1.2 1.2 0 0 1-.348.144c-.14.035-.29.035-.59.035H9Z",
)
C["BookCheck"] = paths(u("BookClosed"), "m9 9 2 2 4-4")
C["ScrollText"] = paths(
    "M6 5.5a2.5 2.5 0 0 0-5 0V8h5",
    "M3.5 3H16.2c1.12 0 1.68 0 2.108.218a2 2 0 0 1 .874.874C19.4 4.52 19.4 5.08 19.4 6.2V16",
    "M6 5.5v14a2.5 2.5 0 0 0 5 0V18h11v1.5a2.5 2.5 0 0 1-2.5 2.5h-11",
    "M10 8h6m-6 4h6",
)
C["IdCard"] = paths(
    rect(2, 4, 22, 20, 3.2),
    circ(8.5, 10.5, 2),
    "M5.5 16.5a3 3 0 0 1 6 0",
    "M14.5 10h4m-4 4h4",
)
C["Fuel"] = paths(
    rpath([(4, 22), (4, 2), (14, 2), (14, 22)], 3.2),
    "M2.5 22h13",
    "M4 10h10",
    "M14 12.5h1.5a2 2 0 0 1 2 2V17a1.5 1.5 0 0 0 3 0V9.33c0-.53-.21-1.04-.59-1.41L17 5",
)
C["Bot"] = paths(
    rect(4, 8, 20, 20, 3.2),
    "M12 8V5",
    dot(12, 4),
    "M9.5 13v1.5m5-1.5v1.5",
    "M2 13v3m20-3v3",
)
C["Brain"] = paths(
    BRAIN_LEFT,
    BRAIN_RIGHT,
    "M12 5.2v14.6",
    BRAIN_LEFT_SULCI,
    "M17.3 3.9c-.2 1.7-1.4 2.8-3 3M20.1 7.6c-1.6.2-2.7 1.2-3 2.7M21.4 12.4c-1.5-.6-3.2-.4-4.4.8M20 17.2c-1.3-.9-1.9-2.2-1.8-3.6M16.4 20.4c-.1-1.6-1-2.8-2.5-3.3",
)
C["BrainCircuit"] = paths(
    BRAIN_LEFT,
    "M12 5.2v14.6",
    BRAIN_LEFT_SULCI,
    "M12 8h4.5M12 12.5h2.5l2 2h1.5M12 17h3.5v2.5",
    circ(18, 8, 1.5),
    circ(20, 14.5, 1.5),
    circ(17, 19.5, 1.5),
)
C["GitCompare"] = paths(
    circ(6, 6, 3), circ(18, 18, 3), rpath([(12, 6), (18, 6), (18, 15)], 3.2), rpath([(12, 18), (6, 18), (6, 9)], 3.2)
)
C["GitCompareArrows"] = paths(
    circ(5, 6, 3),
    circ(19, 18, 3),
    rpath([(12, 6), (19, 6), (19, 15)], 3.2),
    "m15 9-3-3 3-3",
    rpath([(12, 18), (5, 18), (5, 9)], 3.2),
    "m9 15 3 3-3 3",
)
C["ReceiptText"] = paths(u("Receipt"), "M8 8h8M8 11.5h8M8 15h5")
C["Handshake"] = paths(
    "M2 12.5 5.5 6.5l3.2 1.3",
    "M22 12.5 18.5 6.5l-3.8 1.4",
    "M14.7 7.9 11.6 6.6a2 2 0 0 0-1.8.15L7.3 8.6a1.3 1.3 0 0 0 1.4 2.2l2.5-1.3 5.6 5.1",
    "M16.8 14.6a1.3 1.3 0 0 1-1.8 1.8l-.9-.9m.9.9a1.3 1.3 0 0 1-1.8 1.8l-.9-.9m.9.9a1.3 1.3 0 0 1-1.8 1.8L9.5 18.6 4.8 13.6",
    "m19.2 13.4-2.4 1.2",
)
C["Biohazard"] = paths(
    *(arc(*polar(a, 5.4), 3.9, a + 48, a + 312) for a in (-90, 30, 150)),
    *(arc(12, 12.5, 5.2, a + 32, a + 88) for a in (-90, 30, 150)),
    circ(12, 12.5, 1.5),
)
C["Flame"] = paths(
    "M12 22a7 7 0 0 0 7-7c0-3.5-2.3-6-4.2-9.6C14.3 3.5 14 2.5 14 2c-2.2 1.3-3.8 3.7-3.8 6.4 0 .8.2 1.6.5 2.3-1.3-.4-2.4-1.5-2.8-2.9C6.1 9.6 5 12.2 5 15a7 7 0 0 0 7 7Z",
    "M12 22a3 3 0 0 0 3-3c0-1.9-1.5-3-3-5-1.5 2-3 3.1-3 5a3 3 0 0 0 3 3Z",
)
C["Gavel"] = paths(gavel_head(), "m12.8 12.8 7.7 7.7", "M2.5 21.5h9")
C["Headset"] = paths(
    "M4 15v-3a8 8 0 0 1 16 0v3",
    rect(2.5, 13, 6.5, 19, 1.6),
    rect(17.5, 13, 21.5, 19, 1.6),
    "M19.5 19v.5a2.5 2.5 0 0 1-2.5 2.5h-3",
)
C["Radar"] = paths(
    arc(12, 12, 10, -40, 270),
    "M12 6a6 6 0 1 0 6 6",
    "m12 12 7.07-7.07",
    dot(12, 12),
)
C["Factory"] = paths(
    rpath([(2, 21), (2, 10), (8, 13.5), (8, 10), (14, 13.5), (14, 3), (19, 3), (19, 9), (22, 9), (22, 21)], [0, 1.6, 0, 0, 0, 0, 1.6, 0, 1.6, 0], closed=True),
    "M6 17.5h2m4 0h2m4 0h.01",
)
C["Warehouse"] = paths(
    rpath([(2, 21), (2, 9), (12, 3), (22, 9), (22, 21)], [0, 3.2, 2.4, 3.2, 0]),
    "M6 21v-8.4c0-.56 0-.84.109-1.054a1 1 0 0 1 .437-.437C6.76 11 7.04 11 7.6 11h8.8c.56 0 .84 0 1.054.109a1 1 0 0 1 .437.437C18 11.76 18 12.04 18 12.6V21",
    "M6 15h12M6 18h12M1 21h22",
)
C["ShippingContainer"] = paths(rect(2, 5, 22, 19, 3.2), "M6.5 9v6m4-6v6m4-6v6m4-6v6")
C["Boxes"] = paths(
    rect(2, 13, 11, 21, 1.6),
    rect(13, 13, 22, 21, 1.6),
    rect(7.5, 3, 16.5, 11, 1.6),
    "M5 13v2.5h3V13m8 0v2.5h3V13M10.5 3v2.5h3V3",
)
C["Trailer"] = paths(
    rect(2, 3.5, 22, 15, 1.6),
    "M6 3.5V15m4-11.5V15m4-11.5V15m4-11.5V15",
    "M5 15v3.5m-1.5 0h3",
    circ(15.5, 18, 2),
    circ(19.8, 18, 2),
)
C["SemiTruck"] = paths(
    rect(1, 4, 14.5, 14.5, 1.6),
    rpath([(15.5, 14.5), (15.5, 7.5), (19.2, 7.5), (22.5, 10.8), (22.5, 14.5)], [0, 1.2, 1.2, 1.2, 0]),
    "M17.5 7.5v3.5h5",
    "M1 14.5h21.5",
    circ(4.3, 17.6, 1.9),
    circ(8.4, 17.6, 1.9),
    circ(14, 17.6, 1.9),
    circ(20.2, 17.6, 1.9),
)
C["TrafficCone"] = paths(
    "M10.2 3.3c.3-.8.7-1.3 1.8-1.3s1.5.5 1.8 1.3L18.3 18H5.7Z",
    "M8.9 7.6c2 .7 4.2.7 6.2 0M8.3 10c2.4.8 5 .8 7.4 0M7.1 13.6c3.2 1.1 6.6 1.1 9.8 0M6.5 15.9c3.6 1.2 7.4 1.2 11 0",
    rpath([(2.5, 18), (21.5, 18), (21.5, 21.5), (2.5, 21.5)], 1.2, closed=True),
)
C["Construction"] = paths(
    rect(2, 6, 22, 13, 1.6),
    "m7 6-3.5 7m9-7L9 13m9-7-3.5 7m6.5-4.5L19.5 13",
    "M6 13v8m12-8v8M4 21h4m8 0h4",
)
C["Coffee"] = paths(
    "M4 9h12v5a6 6 0 0 1-6 6 6 6 0 0 1-6-6V9Z",
    "M16 10.5h1.5a2.5 2.5 0 0 1 0 5H16",
    "M7.5 2.5v3m5-3v3",
    "M2 22h16",
)
C["DoorOpen"] = paths(
    "M2 21h20",
    "M13 3.2v17.8L5 19.6V5.3c0-.65 0-.98.12-1.24a1.5 1.5 0 0 1 .56-.66c.24-.16.56-.22 1.2-.33L13 2Z",
    "M13 4h3.6c.84 0 1.26 0 1.58.16a1.5 1.5 0 0 1 .66.66C19 5.14 19 5.56 19 6.4V21",
    "M10 12h.01",
)
C["Milestone"] = paths(
    "M12 13v9M12 2v4",
    rpath([(4, 6), (17.2, 6), (20.5, 9.5), (17.2, 13), (4, 13)], [1.6, 1.2, 0.8, 1.2, 1.6], closed=True),
)
C["Newspaper"] = paths(
    rpath([(6, 21), (21, 21), (21, 3), (6, 3), (6, 19)], [0, 3.2, 3.2, 1.6, 0]),
    "M6 19.5a1.5 1.5 0 0 1-3 0V9.6c0-.56 0-.84.109-1.054a1 1 0 0 1 .437-.437C3.76 8 4.04 8 4.6 8H6",
    rect(9.5, 7, 13.5, 11, 0.8),
    "M16.5 7h1M16.5 11h1M9.5 15h8M9.5 18h5",
)
C["Stamp"] = paths(
    "M9.5 13V9.4a3 3 0 1 1 5 0V13",
    rect(3, 13, 21, 18, 1.6),
    "M5 21.5h14",
)
C["Sigma"] = paths("M18 7V4H6l6 8-6 8h12v-3")
C["Split"] = paths(
    "M12 22v-7.17c0-.53-.21-1.04-.59-1.42L5 7",
    "M12 14.83c0-.53.21-1.04.59-1.42L19 7",
    "M5 12V7h5M14 7h5v5",
)

ASSIST = (
    '<path d="M2.7 10.3a2.41 2.41 0 0 0 0 3.41l7.59 7.59a2.41 2.41 0 0 0 3.41 0l7.59-7.59a2.41 2.41 0 0 0 0-3.41l-7.59-7.59a2.41 2.41 0 0 0-3.41 0Z"/>'
    '<circle cx="12" cy="12" r="2" fill="currentColor"/>'
)
C["AssistMark"] = ASSIST

json.dump(C, open(os.path.join(HERE, "custom-icons.json"), "w"), indent=1)
print(f"wrote {len(C)} custom icons")
