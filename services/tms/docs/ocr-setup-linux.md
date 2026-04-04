# OCR Setup on Linux

The document intelligence pipeline in TMS uses the `tesseract` binary for OCR on scanned PDFs and image uploads.

## Install

Ubuntu / Debian:

```bash
sudo apt update
sudo apt install -y tesseract-ocr tesseract-ocr-eng
```

Fedora:

```bash
sudo dnf install -y tesseract tesseract-langpack-eng
```

Arch:

```bash
sudo pacman -S --noconfirm tesseract tesseract-data-eng
```

If you need more languages, install the matching language packs and set the configured OCR language accordingly.

## Verify

Check that the binary is available:

```bash
which tesseract
tesseract --version
tesseract --list-langs
```

The default TMS configuration expects:

- command: `tesseract`
- language: `eng`

## Runtime expectation

TMS document intelligence uses:

- native extraction first for PDFs and office/text files
- OCR fallback for scanned PDFs and images

If `tesseract` is missing, scanned/image extraction will move to `Failed` instead of looping forever.

## Local test

After installing `tesseract`, upload a scanned PDF or image document and verify:

1. The document reaches `Indexed`.
2. Extracted text is visible in the document intelligence dialog.
3. Resource-scoped document search can find text from the scanned file.

## Optional config override

If `tesseract` is not on `PATH`, point TMS at the binary explicitly in config/env for document intelligence OCR command.
