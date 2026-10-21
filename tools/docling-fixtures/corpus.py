"""Writes the imaging benchmark corpus.

The documents the imaging benchmark (services/tms/internal/testutil/imagingbench and the
imaging worker's criterion benches) runs every operation over. It extends generate.py's
synthetic freight documents with the shapes real uploads take: scanned bills of lading,
phone photos (with EXIF orientation, and as HEIC), native PDFs from 1 to 200 pages, capture
pages with and without cover-sheet codes, documents of 5, 20 and 50 MB, and malformed and
hostile PDFs.

    uv run --with pillow --with pillow-heif --with reportlab --with qrcode --with numpy \\
        python3 -I tools/docling-fixtures/corpus.py <output-dir>

Every company, person, address and number is invented. The output is deterministic for a
given set of library versions (noise is seeded). Next to every document whose text is known
the generator writes <name>.truth.txt, the text a perfect OCR would read, so the benchmark
can score character error rate. manifest.json lists every file with its kind.
"""

from __future__ import annotations

import io
import json
import random
import sys
import zlib
from pathlib import Path

import qrcode
from PIL import Image, ImageFilter, ImageOps
from reportlab.lib.pagesizes import letter
from reportlab.lib.units import inch
from reportlab.pdfgen import canvas as pdfcanvas

sys.path.insert(0, str(Path(__file__).resolve().parent))
import generate  # noqa: E402

CARRIERS = ["Blue Ridge Transport Inc.", "Cascade Line Haul LLC", "Prairie Wind Freight", "Iron Mesa Logistics"]
CITIES = [
    ("Fresno", "CA", "93725"),
    ("Reno", "NV", "89502"),
    ("Boise", "ID", "83705"),
    ("Tulsa", "OK", "74107"),
    ("Omaha", "NE", "68110"),
    ("Dayton", "OH", "45414"),
]
COMMODITIES = ["Packaged snack foods", "Canned vegetables", "Paper towels", "Bottled water", "Pet food", "Auto parts"]


class Manifest:
    def __init__(self, root: Path) -> None:
        self.root = root
        self.entries: list[dict] = []

    def add(self, name: str, kind: str, **extra) -> Path:
        self.entries.append({"file": name, "kind": kind, **extra})
        return self.root / name

    def truth(self, name: str, lines: list[str]) -> None:
        (self.root / f"{name}.truth.txt").write_text("\n".join(lines) + "\n", encoding="utf-8")

    def write(self) -> None:
        (self.root / "manifest.json").write_text(json.dumps(self.entries, indent=2) + "\n", encoding="utf-8")


def bol_lines(rng: random.Random, number: int) -> list[str]:
    origin, dest = rng.sample(CITIES, 2)
    carrier = rng.choice(CARRIERS)
    commodity = rng.choice(COMMODITIES)
    pallets = rng.randint(4, 26)
    weight = rng.randint(8000, 44000)
    return [
        "STRAIGHT BILL OF LADING",
        f"BOL # HF-2026-{10000 + number}",
        f"Ship date 10/{rng.randint(1, 28):02d}/2026",
        f"Ship from {origin[0]}, {origin[1]} {origin[2]}",
        f"Ship to {dest[0]}, {dest[1]} {dest[2]}",
        f"Carrier {carrier}",
        f"Trailer {rng.randint(10000, 99999)}  Seal {rng.randint(1000000, 9999999)}",
        f"PO {rng.randint(10000, 99999)}  Load RC-{rng.randint(10000, 99999)}",
        f"{pallets} PLT {commodity}",
        f"Weight {weight:,} lbs  Class {rng.choice([55, 65, 70, 85, 100])}",
        "Received in good order except as noted",
    ]


def draw_sheet(lines: list[str], dpi: int) -> generate.Sheet:
    sheet = generate.Sheet(8.5, 11, dpi)
    sheet.text(0.7, 0.7, lines[0], size=18, bold=True)
    y = 1.5
    for line in lines[1:]:
        sheet.text(0.7, y, line, size=12)
        y += 0.42
    sheet.box(0.55, 1.35, 7.4, y - 1.2)
    return sheet


def scan_noise(img: Image.Image, rng: random.Random, skew: float, speckles: int) -> Image.Image:
    img = img.rotate(skew, resample=Image.BICUBIC, expand=False, fillcolor=255)
    pixels = img.load()
    for _ in range(speckles):
        x, y = rng.randrange(img.width), rng.randrange(img.height)
        pixels[x, y] = 0 if pixels[x, y] > 128 else 255
    return img.filter(ImageFilter.GaussianBlur(0.6))


def scanned_bols(m: Manifest, rng: random.Random) -> None:
    for pages, bilevel in [(1, True), (1, False), (5, False), (20, True)]:
        name = f"scan-bol-{pages}p-{'g4' if bilevel else 'gray'}.pdf"
        images, truth = [], []
        for page in range(pages):
            lines = bol_lines(rng, page)
            truth.extend(lines)
            img = scan_noise(draw_sheet(lines, 200).image, rng, rng.uniform(-1.5, 1.5), 6000)
            images.append(img.point(lambda v: 0 if v < 150 else 255, mode="1") if bilevel else img)
        images[0].save(m.add(name, "scanned-pdf", pages=pages), "PDF", resolution=200, save_all=True, append_images=images[1:])
        m.truth(name, truth)


def native_pdf(m: Manifest, rng: random.Random, pages: int) -> None:
    name = f"native-{pages}p.pdf"
    path = m.add(name, "native-pdf", pages=pages)
    c = pdfcanvas.Canvas(str(path), pagesize=letter, pageCompression=1)
    c.setTitle(f"Freight statement, {pages} pages")
    c.setAuthor("Northwind Freight Brokerage LLC")
    truth = []
    for page in range(pages):
        c.setFont("Helvetica-Bold", 14)
        title = f"Freight statement page {page + 1} of {pages}"
        c.drawString(0.75 * inch, 10.2 * inch, title)
        truth.append(title)
        c.setFont("Helvetica", 9)
        y = 9.7 * inch
        for row in range(34):
            origin, dest = rng.sample(CITIES, 2)
            cells = [
                f"RC-{rng.randint(10000, 99999)}",
                f"{origin[0]}, {origin[1]}",
                f"{dest[0]}, {dest[1]}",
                f"{rng.randint(80, 2400)} mi",
                f"${rng.randint(400, 9800):,}.{rng.randint(0, 99):02d}",
            ]
            for col, (x, cell) in enumerate(zip([0.75, 1.75, 3.25, 4.75, 6.0], cells)):
                c.drawString(x * inch, y, cell)
            truth.append(" ".join(cells))
            y -= 0.26 * inch
        c.showPage()
    c.save()
    m.truth(name, truth)


def phone_photos(m: Manifest, rng: random.Random) -> None:
    lines = bol_lines(rng, 900)
    page = draw_sheet(lines, 300).image.convert("RGB")
    canvas = Image.new("RGB", (3024, 4032), (92, 84, 74))
    width, height = page.size
    coefficients = generate._perspective_coefficients(
        [(0, 0), (width, 0), (width, height), (0, height)],
        [(90, 60), (width - 40, 130), (width - 120, height - 20), (30, height - 90)],
    )
    warped = page.transform(page.size, Image.PERSPECTIVE, coefficients, Image.BICUBIC, fillcolor=(92, 84, 74))
    canvas.paste(warped, ((3024 - width) // 2, (4032 - height) // 2))
    canvas = canvas.filter(ImageFilter.GaussianBlur(1.0))

    canvas.save(m.add("photo-bol.jpg", "photo", format="jpeg"), "JPEG", quality=85)
    m.truth("photo-bol.jpg", lines)

    rotated = canvas.transpose(Image.Transpose.ROTATE_90)
    exif = Image.Exif()
    exif[0x0112] = 6
    rotated.save(m.add("photo-bol-exif6.jpg", "photo", format="jpeg", orientation=6), "JPEG", quality=85, exif=exif)
    m.truth("photo-bol-exif6.jpg", lines)

    canvas.resize((1512, 2016)).save(m.add("photo-bol.png", "photo", format="png"), "PNG", optimize=False)
    m.truth("photo-bol.png", lines)
    canvas.save(m.add("photo-bol.webp", "photo", format="webp"), "WEBP", quality=80)
    m.truth("photo-bol.webp", lines)
    canvas.save(m.add("photo-bol.tiff", "photo", format="tiff"), "TIFF", compression="tiff_lzw")
    m.truth("photo-bol.tiff", lines)
    canvas.resize((756, 1008)).convert("P", palette=Image.Palette.ADAPTIVE).save(m.add("photo-bol.gif", "photo", format="gif"), "GIF")

    import pillow_heif

    pillow_heif.register_heif_opener()
    canvas.save(m.add("photo-bol.heic", "photo", format="heic"), "HEIF", quality=80)
    m.truth("photo-bol.heic", lines)


def capture_pages(m: Manifest, rng: random.Random) -> None:
    def one_page(name: str, img: Image.Image, **extra) -> None:
        img.save(m.add(name, "capture-page", **extra), "PDF", resolution=200)

    cover = generate.Sheet(8.5, 11, 200)
    cover.text(0.8, 0.8, "TRENOVA CAPTURE COVER SHEET", size=20, bold=True)
    cover.text(0.8, 1.5, "Shipment S-2026-118842  BOL HF-2026-10433", size=12)
    code = qrcode.QRCode(error_correction=qrcode.constants.ERROR_CORRECT_Q, box_size=10, border=4)
    code.add_data("trenova:cs:01JD3Q5V8Z2K7W4M6N9P0R1T2X")
    code.make()
    qr = code.make_image(fill_color="black", back_color="white").convert("L").resize((600, 600), Image.NEAREST)
    cover.image.paste(qr, (500, 700))
    second = qrcode.make("trenova:cs:01JD3Q5V8Z2K7W4M6N9P0R1T3Y").convert("L").resize((300, 300), Image.NEAREST)
    cover.image.paste(second, (1100, 1500))
    one_page("capture-cover-2qr.pdf", scan_noise(cover.image, rng, 1.2, 3000), codes=2)

    blank = Image.new("L", (1700, 2200), 246)
    pixels = blank.load()
    for _ in range(4000):
        pixels[rng.randrange(1700), rng.randrange(2200)] = rng.randint(120, 200)
    for y in range(2200):
        for x in range(0, 40):
            pixels[x, y] = 90
    one_page("capture-blank.pdf", blank, blank=True)

    for index in range(3):
        lines = bol_lines(rng, 500 + index)
        one_page(f"capture-text-{index}.pdf", scan_noise(draw_sheet(lines, 200).image, rng, rng.uniform(-1, 1), 4000))
        m.truth(f"capture-text-{index}.pdf", lines)


def sized_document(m: Manifest, rng: random.Random, megabytes: int) -> None:
    name = f"large-{megabytes}mb.pdf"
    target = megabytes << 20
    images: list[Image.Image] = []
    size, page_bytes = 0, 0
    while size + page_bytes // 2 < target:
        lines = bol_lines(rng, len(images))
        img = draw_sheet(lines, 200).image.convert("RGB")
        noise = Image.effect_noise(img.size, 40).convert("RGB")
        img = Image.blend(img, noise, 0.25)
        buf = io.BytesIO()
        img.save(buf, "PDF", resolution=200, quality=92)
        page_bytes = buf.tell()
        size += page_bytes
        images.append(img)
    images[0].save(m.add(name, "large-pdf", pages=len(images)), "PDF", resolution=200, save_all=True, append_images=images[1:], quality=92)


def hostile(m: Manifest) -> None:
    good = (m.root / "native-1p.pdf").read_bytes()

    m.add("hostile-truncated.pdf", "hostile", expect="invalid").write_bytes(good[: len(good) // 2])
    m.add("hostile-garbage.pdf", "hostile", expect="invalid").write_bytes(random.Random(7).randbytes(64 << 10))
    m.add("hostile-empty.pdf", "hostile", expect="invalid").write_bytes(b"")
    broken = good.replace(b"xref", b"xrxf").replace(b"startxref", b"startxrxf")
    m.add("hostile-broken-xref.pdf", "hostile", expect="repairable").write_bytes(broken)

    c = pdfcanvas.Canvas(str(m.add("hostile-encrypted.pdf", "hostile", expect="encrypted")), encrypt="hunter2")
    c.drawString(72, 720, "Encrypted rate confirmation")
    c.showPage()
    c.save()

    def raw_pdf(objects: list[bytes]) -> bytes:
        out = bytearray(b"%PDF-1.7\n%\xe2\xe3\xcf\xd3\n")
        offsets = []
        for number, body in enumerate(objects, start=1):
            offsets.append(len(out))
            out += b"%d 0 obj\n" % number + body + b"\nendobj\n"
        xref = len(out)
        out += b"xref\n0 %d\n0000000000 65535 f \n" % (len(objects) + 1)
        for offset in offsets:
            out += b"%010d 00000 n \n" % offset
        out += b"trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n" % (len(objects) + 1, xref)
        return bytes(out)

    giant = raw_pdf([
        b"<< /Type /Catalog /Pages 2 0 R >>",
        b"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
        b"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 14400 14400] /UserUnit 75000 /Contents 4 0 R >>",
        b"<< /Length 44 >>\nstream\n0 0 m 14400 14400 l S BT /F1 12 Tf ET\nendstream",
    ])
    m.add("hostile-giant-page.pdf", "hostile", expect="pixel-budget").write_bytes(giant)

    compressor = zlib.compressobj(9)
    zero_block = b"\x00" * (1 << 20)
    bomb = b"".join(compressor.compress(zero_block) for _ in range(512)) + compressor.flush()
    flate = raw_pdf([
        b"<< /Type /Catalog /Pages 2 0 R >>",
        b"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
        b"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /XObject << /Im0 5 0 R >> >> /Contents 4 0 R >>",
        b"<< /Length 31 >>\nstream\nq 612 0 0 792 0 0 cm /Im0 Do Q\nendstream",
        b"<< /Type /XObject /Subtype /Image /Width 23170 /Height 23170 /ColorSpace /DeviceGray /BitsPerComponent 8 /Filter /FlateDecode /Length %d >>\nstream\n" % len(bomb)
        + bomb
        + b"\nendstream",
    ])
    m.add("hostile-flate-bomb.pdf", "hostile", expect="resource").write_bytes(flate)

    depth = 20000
    nested = b"<< /Type /Catalog /Pages 2 0 R /Deep " + b"[" * depth + b"]" * depth + b" >>"
    m.add("hostile-deep-nesting.pdf", "hostile", expect="invalid-or-ok").write_bytes(raw_pdf([
        nested,
        b"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
        b"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>",
    ]))

    loop = raw_pdf([
        b"<< /Type /Catalog /Pages 2 0 R >>",
        b"<< /Type /Pages /Kids [2 0 R 3 0 R] /Count 2 >>",
        b"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>",
    ])
    m.add("hostile-page-tree-loop.pdf", "hostile", expect="invalid-or-ok").write_bytes(loop)

    jpeg = io.BytesIO()
    Image.new("L", (64, 64), 255).save(jpeg, "JPEG")
    data = bytearray(jpeg.getvalue())
    sof = data.find(b"\xff\xc0")
    data[sof + 5 : sof + 9] = (65000).to_bytes(2, "big") + (65000).to_bytes(2, "big")
    m.add("hostile-pixel-bomb.jpg", "hostile", expect="pixel-budget").write_bytes(bytes(data))


def main() -> None:
    if len(sys.argv) != 2:
        raise SystemExit("usage: corpus.py <output-dir>")
    out = Path(sys.argv[1])
    out.mkdir(parents=True, exist_ok=True)
    rng = random.Random(20261020)
    m = Manifest(out)

    for pages in (1, 10, 50, 200):
        native_pdf(m, rng, pages)
    generate.rate_confirmation_pdf(m.add("rate-confirmation.pdf", "native-pdf", pages=2))
    generate.faxed_bol_pdf(m.add("faxed-bol.pdf", "scanned-pdf", pages=1))
    scanned_bols(m, rng)
    phone_photos(m, rng)
    capture_pages(m, rng)
    for megabytes in (5, 20, 50):
        sized_document(m, rng, megabytes)
    hostile(m)
    m.write()


if __name__ == "__main__":
    main()
