"""Writes the synthetic documents the Docling renderer's golden tests convert.

Every company, person, address and number below is invented. Nothing here is,
or is derived from, a customer's file. The six documents cover what the
renderer has to get right: a native PDF with tables and a printed page header,
a faxed scan, a phone photo, and the three non-PDF formats uploads arrive in.

    python3 -I tools/docling-fixtures/generate.py <output-dir>

The output is deterministic for a given Pillow, reportlab, python-docx and
openpyxl version (noise is seeded), so regenerating only changes the files
when the generator changes.
"""

from __future__ import annotations

import random
import sys
from email.message import EmailMessage
from email.utils import format_datetime
from datetime import datetime, timezone
from pathlib import Path

from PIL import Image, ImageDraw, ImageFilter, ImageFont, ImageOps
from reportlab.lib import colors
from reportlab.lib.pagesizes import letter
from reportlab.lib.styles import getSampleStyleSheet
from reportlab.lib.units import inch
from reportlab.platypus import Paragraph, SimpleDocTemplate, Spacer, Table, TableStyle

FONT_DIR = Path("/usr/share/fonts/truetype/dejavu")
SANS = str(FONT_DIR / "DejaVuSans.ttf")
SANS_BOLD = str(FONT_DIR / "DejaVuSans-Bold.ttf")
MONO = str(FONT_DIR / "DejaVuSansMono.ttf")

LOAD_NUMBER = "RC-48213"


def rate_confirmation_pdf(path: Path) -> None:
    styles = getSampleStyleSheet()

    def header_footer(canvas, doc):
        canvas.saveState()
        canvas.setFont("Helvetica-Bold", 9)
        canvas.drawString(0.75 * inch, 10.55 * inch, "Northwind Freight Brokerage LLC")
        canvas.drawRightString(7.75 * inch, 10.55 * inch, f"Load # {LOAD_NUMBER}")
        canvas.setFont("Helvetica", 8)
        canvas.drawString(
            0.75 * inch, 0.5 * inch, "MC 900123 | 1200 Harbor Way, Oakland, CA 94607"
        )
        canvas.drawRightString(7.75 * inch, 0.5 * inch, f"Page {doc.page} of 2")
        canvas.restoreState()

    grid = TableStyle(
        [
            ("GRID", (0, 0), (-1, -1), 0.5, colors.black),
            ("BACKGROUND", (0, 0), (-1, 0), colors.lightgrey),
            ("FONTNAME", (0, 0), (-1, 0), "Helvetica-Bold"),
            ("FONTSIZE", (0, 0), (-1, -1), 8),
            ("VALIGN", (0, 0), (-1, -1), "TOP"),
        ]
    )

    story = [
        Paragraph("Carrier Rate Confirmation", styles["Title"]),
        Paragraph(
            "Carrier: Blue Ridge Transport Inc. (MC 711204, DOT 2290431)<br/>"
            "Dispatcher: Dana Whitfield, (555) 014-2290<br/>"
            "Equipment: 53' Dry Van | Weight: 38,450 lbs | Commodity: Packaged snacks",
            styles["Normal"],
        ),
        Spacer(1, 12),
        Paragraph("Stops", styles["Heading2"]),
        Table(
            [
                ["#", "Type", "Facility", "Address", "Date", "Window", "Ref"],
                [
                    "1",
                    "Pickup",
                    "Harvest Foods DC",
                    "4410 Industrial Pkwy, Fresno, CA 93725",
                    "10/21/2026",
                    "08:00-12:00",
                    "PU 77120",
                ],
                [
                    "2",
                    "Delivery",
                    "Valley Grocers #14",
                    "88 Commerce Dr, Reno, NV 89502",
                    "10/22/2026",
                    "06:00-10:00",
                    "PO 55318",
                ],
            ],
            colWidths=[0.3 * inch, 0.7 * inch, 1.3 * inch, 2.2 * inch, 0.8 * inch, 0.9 * inch, 0.8 * inch],
            style=grid,
        ),
        Spacer(1, 12),
        Paragraph("Charges", styles["Heading2"]),
        Table(
            [
                ["Description", "Qty", "Rate", "Amount"],
                ["Line haul", "1", "$1,850.00", "$1,850.00"],
                ["Fuel surcharge", "1", "$212.75", "$212.75"],
                ["Detention (after 2 hrs)", "0", "$50.00/hr", "$0.00"],
                ["Total", "", "", "$2,062.75"],
            ],
            colWidths=[2.6 * inch, 0.6 * inch, 1.2 * inch, 1.2 * inch],
            style=grid,
        ),
        Spacer(1, 18),
        Paragraph("Terms and conditions", styles["Heading2"]),
    ]
    terms = (
        "Carrier agrees to transport the shipment described above under the broker-carrier "
        "agreement on file. Payment terms are net 30 days from receipt of a signed bill of "
        "lading and this confirmation. Quick pay is available at 2% with a 48-hour turnaround. "
        "Carrier must call in at pickup and delivery and report any delay longer than one hour. "
        "Double brokering is prohibited and voids payment."
    )
    for index in range(6):
        story.append(Paragraph(f"{index + 1}. {terms}", styles["Normal"]))
        story.append(Spacer(1, 6))
    story += [
        Spacer(1, 18),
        Paragraph("Accepted by carrier: ______________________   Date: __________", styles["Normal"]),
    ]

    SimpleDocTemplate(
        str(path),
        pagesize=letter,
        topMargin=0.9 * inch,
        bottomMargin=0.8 * inch,
        leftMargin=0.75 * inch,
        rightMargin=0.75 * inch,
        title="Rate Confirmation",
        author="Northwind Freight Brokerage LLC",
    ).build(story, onFirstPage=header_footer, onLaterPages=header_footer)


class Sheet:
    """A white page drawn at a given DPI, in inches from the top left."""

    def __init__(self, width_in: float, height_in: float, dpi: int) -> None:
        self.dpi = dpi
        self.image = Image.new("L", (int(width_in * dpi), int(height_in * dpi)), 255)
        self.draw = ImageDraw.Draw(self.image)

    def font(self, size_pt: float, bold: bool = False, mono: bool = False) -> ImageFont.FreeTypeFont:
        face = MONO if mono else (SANS_BOLD if bold else SANS)
        return ImageFont.truetype(face, int(size_pt * self.dpi / 72))

    def text(self, x: float, y: float, value: str, size: float = 10, bold: bool = False) -> None:
        self.draw.text((x * self.dpi, y * self.dpi), value, fill=0, font=self.font(size, bold))

    def box(self, x: float, y: float, w: float, h: float, width: int = 2) -> None:
        d = self.dpi
        self.draw.rectangle((x * d, y * d, (x + w) * d, (y + h) * d), outline=0, width=width)

    def table(self, x: float, y: float, widths: list[float], rows: list[list[str]], row_h: float = 0.32) -> float:
        for r, row in enumerate(rows):
            cx = x
            for c, cell in enumerate(row):
                self.box(cx, y + r * row_h, widths[c], row_h)
                self.text(cx + 0.06, y + r * row_h + 0.07, cell, size=9, bold=r == 0)
                cx += widths[c]
        return y + len(rows) * row_h


def faxed_bol_pdf(path: Path) -> None:
    sheet = Sheet(8.5, 11, 200)
    sheet.text(0.6, 0.35, "FAX  10/21/2026 14:02  FROM: HARVEST FOODS DC  559-555-0142  P.1/1", size=8)
    sheet.text(2.4, 0.8, "STRAIGHT BILL OF LADING", size=18, bold=True)
    sheet.text(5.6, 1.35, "BOL #: HF-2026-10433", size=11, bold=True)
    sheet.text(0.7, 1.35, "Ship date: 10/21/2026", size=11)
    sheet.box(0.6, 1.8, 3.6, 1.3)
    sheet.text(0.7, 1.85, "SHIP FROM", size=9, bold=True)
    sheet.text(0.7, 2.15, "Harvest Foods DC", size=10)
    sheet.text(0.7, 2.4, "4410 Industrial Pkwy", size=10)
    sheet.text(0.7, 2.65, "Fresno, CA 93725", size=10)
    sheet.box(4.3, 1.8, 3.6, 1.3)
    sheet.text(4.4, 1.85, "SHIP TO", size=9, bold=True)
    sheet.text(4.4, 2.15, "Valley Grocers #14", size=10)
    sheet.text(4.4, 2.4, "88 Commerce Dr", size=10)
    sheet.text(4.4, 2.65, "Reno, NV 89502", size=10)
    sheet.text(0.7, 3.35, "Carrier: Blue Ridge Transport Inc.   SCAC: BRTI   Trailer: 53118   Seal: 0049921", size=10)
    sheet.text(0.7, 3.65, f"Customer order / PO: 55318    Load: {LOAD_NUMBER}", size=10)
    bottom = sheet.table(
        0.6,
        4.1,
        [0.8, 0.9, 3.6, 1.1, 0.9],
        [
            ["Qty", "Type", "Description", "Weight", "Class"],
            ["22", "PLT", "Packaged snack foods, non-haz", "38,450", "65"],
            ["", "", "", "", ""],
            ["22", "", "TOTAL", "38,450", ""],
        ],
    )
    sheet.text(0.7, bottom + 0.4, "Shipper signature: ____________________     Driver signature: ____________________", size=10)
    sheet.text(0.7, bottom + 0.8, "Received in good order except as noted.", size=9)

    rng = random.Random(4213)
    img = sheet.image.rotate(0.8, resample=Image.BICUBIC, expand=False, fillcolor=255)
    img = img.resize((img.width, img.height // 2), Image.BILINEAR).resize((img.width, img.height), Image.NEAREST)
    pixels = img.load()
    for _ in range(9000):
        x, y = rng.randrange(img.width), rng.randrange(img.height)
        pixels[x, y] = 0 if pixels[x, y] > 128 else 255
    img = img.point(lambda value: 0 if value < 150 else 255, mode="1")
    img.save(path, "PDF", resolution=200)


def photo_pod_jpeg(path: Path) -> None:
    sheet = Sheet(8.5, 11, 120)
    sheet.text(2.3, 0.6, "PROOF OF DELIVERY", size=20, bold=True)
    sheet.text(0.7, 1.4, "Delivery receipt #: VG14-88213", size=12, bold=True)
    sheet.text(0.7, 1.8, f"Load: {LOAD_NUMBER}     PO: 55318     BOL: HF-2026-10433", size=11)
    sheet.text(0.7, 2.2, "Consignee: Valley Grocers #14, 88 Commerce Dr, Reno, NV 89502", size=11)
    sheet.text(0.7, 2.6, "Delivered: 10/22/2026  07:42", size=11)
    sheet.table(
        0.7,
        3.1,
        [1.0, 3.5, 1.4, 1.2],
        [
            ["Pallets", "Description", "Received", "Damaged"],
            ["22", "Packaged snack foods", "22", "0"],
        ],
        row_h=0.4,
    )
    sheet.text(0.7, 4.3, "Exceptions: none", size=11)
    sheet.text(0.7, 5.2, "Received by: M. Ortega", size=12)
    sheet.draw.line((2.0 * 120, 5.95 * 120, 4.6 * 120, 5.7 * 120), fill=0, width=3)
    sheet.draw.line((2.3 * 120, 5.6 * 120, 3.1 * 120, 6.05 * 120), fill=0, width=3)
    sheet.text(0.7, 5.75, "Signature:", size=12)

    page = sheet.image.convert("RGB")
    canvas = Image.new("RGB", (1500, 1900), (92, 84, 74))
    width, height = page.size
    coefficients = _perspective_coefficients(
        [(0, 0), (width, 0), (width, height), (0, height)],
        [(60, 40), (width - 30, 90), (width - 80, height - 10), (20, height - 60)],
    )
    warped = page.transform(page.size, Image.PERSPECTIVE, coefficients, Image.BICUBIC, fillcolor=(92, 84, 74))
    canvas.paste(warped, (230, 300))
    vertical = Image.linear_gradient("L").resize(canvas.size)
    horizontal = Image.linear_gradient("L").transpose(Image.Transpose.ROTATE_90).resize(canvas.size)
    shade = Image.blend(vertical, horizontal, 0.5)
    shade = ImageOps.invert(shade).point(lambda value: 150 + value * 105 // 255)
    canvas = Image.composite(canvas, Image.new("RGB", canvas.size, (0, 0, 0)), shade)
    canvas = canvas.filter(ImageFilter.GaussianBlur(1.1))
    canvas.save(path, "JPEG", quality=72)


def _perspective_coefficients(source, target):
    import numpy

    matrix = []
    for (x, y), (u, v) in zip(target, source):
        matrix.append([x, y, 1, 0, 0, 0, -u * x, -u * y])
        matrix.append([0, 0, 0, x, y, 1, -v * x, -v * y])
    a = numpy.array(matrix, dtype=float)
    b = numpy.array(source, dtype=float).reshape(8)
    return numpy.linalg.solve(a, b).tolist()


def rate_confirmation_docx(path: Path) -> None:
    from docx import Document
    from docx.shared import Pt

    document = Document()
    document.core_properties.author = "Northwind Freight Brokerage LLC"
    document.core_properties.created = datetime(2026, 10, 20, tzinfo=timezone.utc)
    header = document.sections[0].header.paragraphs[0]
    header.text = f"Northwind Freight Brokerage LLC    Load # {LOAD_NUMBER}-B"
    document.add_heading("Carrier Rate Confirmation", level=1)
    document.add_paragraph("Carrier: Granite State Haulers (MC 640118)")
    document.add_paragraph("Equipment: 48' Flatbed with tarps")
    document.add_heading("Stops", level=2)
    table = document.add_table(rows=1, cols=4)
    table.style = "Table Grid"
    for cell, label in zip(table.rows[0].cells, ["Type", "Location", "Date", "Window"]):
        cell.text = label
    for row in [
        ["Pickup", "Summit Steel, 12 Mill Rd, Nashua, NH 03060", "10/23/2026", "07:00-09:00"],
        ["Delivery", "Coastal Builders Yard, 5 Pier St, Portland, ME 04101", "10/23/2026", "13:00-16:00"],
    ]:
        cells = table.add_row().cells
        for cell, value in zip(cells, row):
            cell.text = value
    document.add_heading("Charges", level=2)
    for item in ["Line haul: $1,240.00", "Tarp fee: $75.00", "Total: $1,315.00"]:
        document.add_paragraph(item, style="List Bullet")
    note = document.add_paragraph("Special instructions: chains and binders required; call | 555-0199 on arrival.")
    note.runs[0].font.size = Pt(9)
    document.save(path)


def accessorials_xlsx(path: Path) -> None:
    from openpyxl import Workbook

    workbook = Workbook()
    workbook.properties.creator = "Northwind Freight Brokerage LLC"
    workbook.properties.created = datetime(2026, 10, 20)
    loads = workbook.active
    loads.title = "Tendered loads"
    loads.append(["Load", "Origin", "Destination", "Pickup", "Equipment", "Rate"])
    loads.append([LOAD_NUMBER, "Fresno, CA", "Reno, NV", "2026-10-21", "Dry Van", 2062.75])
    loads.append(["RC-48214", "Stockton, CA", "Boise, ID", "2026-10-22", "Reefer", 3110.00])
    loads.append(["RC-48215", "Modesto, CA", "Salt Lake City, UT", "2026-10-23", "Dry Van", 2780.50])
    charges = workbook.create_sheet("Accessorials")
    charges.append(["Code", "Description", "Amount"])
    charges.append(["DET", "Detention per hour after 2 free", 50])
    charges.append(["LUM", "Lumper, receipt required", 175])
    charges.append(["TONU", "Truck ordered not used", 150])
    workbook.save(path)


def tender_eml(path: Path) -> None:
    message = EmailMessage()
    message["From"] = "Dana Whitfield <dana@northwind-freight.example>"
    message["To"] = "dispatch@blueridge-transport.example"
    message["Subject"] = f"Load tender {LOAD_NUMBER} Fresno CA to Reno NV"
    message["Date"] = format_datetime(datetime(2026, 10, 20, 15, 30, tzinfo=timezone.utc))
    message["Message-ID"] = "<tender-rc48213@northwind-freight.example>"
    message.set_content(
        f"Hi team,\n\nPlease confirm load {LOAD_NUMBER}.\n\n"
        "Pickup: Harvest Foods DC, Fresno, CA on 10/21/2026 08:00-12:00\n"
        "Delivery: Valley Grocers #14, Reno, NV on 10/22/2026 06:00-10:00\n"
        "Rate: $2,062.75 all in\n\nThanks,\nDana\n"
    )
    message.add_alternative(
        f"<html><body><p>Hi team,</p><p>Please confirm load <b>{LOAD_NUMBER}</b>.</p>"
        "<table border='1'><tr><th>Stop</th><th>Location</th><th>Date</th><th>Window</th></tr>"
        "<tr><td>Pickup</td><td>Harvest Foods DC, Fresno, CA</td><td>10/21/2026</td><td>08:00-12:00</td></tr>"
        "<tr><td>Delivery</td><td>Valley Grocers #14, Reno, NV</td><td>10/22/2026</td><td>06:00-10:00</td></tr>"
        "</table><p>Rate: $2,062.75 all in</p><p>Thanks,<br>Dana</p></body></html>",
        subtype="html",
    )
    path.write_bytes(bytes(message))


def main() -> None:
    if len(sys.argv) != 2:
        raise SystemExit("usage: generate.py <output-dir>")
    out = Path(sys.argv[1])
    out.mkdir(parents=True, exist_ok=True)
    rate_confirmation_pdf(out / "rate-confirmation.pdf")
    faxed_bol_pdf(out / "faxed-bol.pdf")
    photo_pod_jpeg(out / "photo-pod.jpg")
    rate_confirmation_docx(out / "rate-confirmation.docx")
    accessorials_xlsx(out / "tendered-loads.xlsx")
    tender_eml(out / "load-tender.eml")


if __name__ == "__main__":
    main()
