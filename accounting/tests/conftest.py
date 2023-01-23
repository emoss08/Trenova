"""
COPYRIGHT 2022 MONTA

This file is part of Monta.

Monta is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

Monta is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with Monta.  If not, see <https://www.gnu.org/licenses/>.
"""

import os
import shutil
from pathlib import Path

import pytest

pytestmark = pytest.mark.django_db


def remove_media_directory(file_path: str) -> None:
    """Remove Media Directory after test tear down.

    Primary usage is when tests are performing file uplaods.
    This method deletes the media directory after the test.
    This is to prevent the media directory from filling up
    with test files.

    Args:
        file_path (str): path to directory in media folder.

    Returns:
        None
    """

    base_dir = Path(__file__).resolve().parent.parent
    media_dir = os.path.join(base_dir, f"media/{file_path}")

    if os.path.exists(media_dir):
        shutil.rmtree(media_dir, ignore_errors=True, onerror=None)


def remove_file(file_path: str) -> None:
    """Remove File after test tear down.

    Primary usage is when tests are performing file uplaods.
    This method deletes the file after the test.
    This is to prevent the media directory from filling up
    with test files.

    Args:
        file_path (str): path to file in media folder.

    Returns:
        None
    """

    base_dir = Path(__file__).resolve().parent.parent
    file = os.path.join(base_dir, f"media/{file_path}")

    if os.path.exists(file):
        os.remove(file)
