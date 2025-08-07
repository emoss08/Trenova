#
# Copyright 2023-2025 Eric Moss
# Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
# Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md#
import logging
from pathlib import Path
from typing import List, Optional

from pdf2image import convert_from_path
from PIL import Image

logger = logging.getLogger(__name__)


class PDFProcessor:
    """Handle PDF to image conversion for document quality assessment"""

    def __init__(self, dpi: int = 300):
        self.dpi = dpi

    def process_pdf(
        self, pdf_path: Path, output_dir: Optional[Path] = None
    ) -> List[Path]:
        """Convert PDF pages to images and save them"""
        if not pdf_path.exists():
            raise FileNotFoundError(f"PDF file not found: {pdf_path}")

        # Create output directory if needed
        if output_dir is None:
            output_dir = Path("temp_pdf_images")
        output_dir.mkdir(exist_ok=True, parents=True)

        try:
            # Convert PDF to images
            images = convert_from_path(pdf_path, dpi=self.dpi)
            image_paths = []

            for i, image in enumerate(images):
                # Save each page as an image
                image_path = output_dir / f"{pdf_path.stem}_page_{i + 1}.png"
                image.save(image_path, "PNG")
                image_paths.append(image_path)
                logger.info(f"Converted page {i + 1} of {pdf_path.name}")

            return image_paths

        except Exception as e:
            logger.error(f"Error processing PDF {pdf_path}: {str(e)}")
            raise

    def extract_page_images(self, pdf_path: Path) -> List[Image.Image]:
        """Extract pages as PIL Image objects without saving to disk"""
        if not pdf_path.exists():
            raise FileNotFoundError(f"PDF file not found: {pdf_path}")

        try:
            return convert_from_path(pdf_path, dpi=self.dpi)
        except Exception as e:
            logger.error(f"Error extracting images from PDF {pdf_path}: {str(e)}")
            raise
