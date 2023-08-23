# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2023 MONTA                                                                         -
#                                                                                                  -
#  This file is part of Monta.                                                                     -
#                                                                                                  -
#  The Monta software is licensed under the Business Source License 1.1. You are granted the right -
#  to copy, modify, and redistribute the software, but only for non-production use or with a total -
#  of less than three server instances. Starting from the Change Date (November 16, 2026), the     -
#  software will be made available under version 2 or later of the GNU General Public License.     -
#  If you use the software in violation of this license, your rights under the license will be     -
#  terminated automatically. The software is provided "as is," and the Licensor disclaims all      -
#  warranties and conditions. If you use this license's text or the "Business Source License" name -
#  and trademark, you must comply with the Licensor's covenants, which include specifying the      -
#  Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use     -
#  Grant, and not modifying the license in any other way.                                          -
# --------------------------------------------------------------------------------------------------

import os
from PIL import Image


def convert_png_to_webp(
    directory: str, lossless: bool = False, quality: int = 80, method: int = 4
) -> None:
    """Converts all PNG images in a given directory to the webp format.

    This function processes all PNG images in the provided directory and converts them into webp images.
    The new images are saved in the same directory with the same filename but with the .webp extension.
    After each conversion, a message is printed specifying the name of the image and the settings used for
    the conversion.

    Args:
        directory (str): The directory containing the png images to convert.

        lossless (bool, optional): If True, the images will be converted losslessly. Default is False.

        quality (int, optional): A value between 0 and 100 indicating the quality of the converted image.
        Higher values will result in larger file size and better quality. Default is 80.

        method (int, optional): A number between 0 and 6 specifying the compression method to use.
        Higher values will result in slower compression speed but better image quality and smaller file size.
        Default is 4.

    Returns:
        None: This function does not return anything.

    Note:
        This function requires the `PIL` library to be installed.
        If the specified directory does not exist, an appropriate message will be printed and the function will return
        immediately.
    """
    if not os.path.exists(directory):
        print(f"Directory {directory} does not exist!")
        return

    for filename in os.listdir(directory):
        if filename.endswith(".png"):
            filepath = os.path.join(directory, filename)
            with Image.open(filepath) as im:
                webp_filepath = f"{os.path.splitext(filepath)[0]}.webp"
                im.save(
                    webp_filepath,
                    "WEBP",
                    lossless=lossless,
                    quality=quality,
                    method=method,
                )
                print(
                    f"Converted {filename} to WEBP format with quality={quality} and method={method}."
                )


if __name__ == "__main__":
    import sys

    if len(sys.argv) < 2:
        print(
            "Usage: python script_name.py <directory_path> [lossless] [quality] [method]"
        )
        sys.exit(1)

    directory_path = sys.argv[1]
    quality = 80
    method = 4

    lossless = sys.argv[2].lower() == "true" if len(sys.argv) > 2 else False
    if len(sys.argv) > 3:
        quality = int(sys.argv[3])
    if len(sys.argv) > 4:
        method = int(sys.argv[4])

    convert_png_to_webp(
        directory_path, lossless=lossless, quality=quality, method=method
    )
