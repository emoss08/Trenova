"""Draws the installer's banner and side bitmaps from the web app's logo.

Windows Installer shows 24-bit bitmaps: a 493x58 banner across the top of
most pages and a 493x312 image behind the welcome and finish pages, whose
text sits on its right, so only the left panel is drawn on. Run it again
when the logo changes:

    python3 native/capture/installer/assets/make-bitmaps.py
"""

from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

HERE = Path(__file__).resolve().parent
LOGO = HERE.parents[3] / "client" / "apps" / "web" / "public" / "logo.ico"
FONTS = Path("/usr/share/fonts/truetype")

# The design system's ink (the primary colour) and card, as sRGB.
INK = (29, 34, 44)
CARD = (255, 255, 255)
INK_MUTED = (172, 178, 192)


def font(name: str, size: int) -> ImageFont.FreeTypeFont:
    for candidate in (FONTS / "dejavu" / name, FONTS / "liberation" / name):
        if candidate.exists():
            return ImageFont.truetype(str(candidate), size)
    return ImageFont.load_default(size)


def logo(size: int) -> Image.Image:
    with Image.open(LOGO) as ico:
        largest = max(ico.info.get("sizes", {(256, 256)}))
        ico.size = largest
        mark = ico.convert("RGBA")
    return mark.resize((size, size), Image.Resampling.LANCZOS)


def banner() -> Image.Image:
    image = Image.new("RGB", (493, 58), CARD)
    mark = logo(38)
    image.paste(mark, (493 - 38 - 12, 10), mark)
    return image


def side() -> Image.Image:
    image = Image.new("RGB", (493, 312), CARD)
    draw = ImageDraw.Draw(image)
    draw.rectangle((0, 0, 164, 312), fill=INK)
    mark = logo(72)
    image.paste(mark, ((164 - 72) // 2, 78), mark)
    for text, typeface, size, colour, y in (
        ("Trenova", "DejaVuSans-Bold.ttf", 17, CARD, 166),
        ("Capture", "DejaVuSans-Bold.ttf", 17, CARD, 188),
        ("Scan and print", "DejaVuSans.ttf", 11, INK_MUTED, 222),
        ("into Trenova", "DejaVuSans.ttf", 11, INK_MUTED, 238),
    ):
        face = font(typeface, size)
        width = draw.textlength(text, font=face)
        draw.text(((164 - width) / 2, y), text, font=face, fill=colour)
    return image


if __name__ == "__main__":
    banner().save(HERE / "banner.bmp")
    side().save(HERE / "dialog.bmp")
