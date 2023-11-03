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

import logging
import secrets

from django.core.exceptions import ValidationError
from django.core.files.base import ContentFile
from PIL import Image, UnidentifiedImageError

from accounts import models
from utils.helpers import optimize_image

log = logging.getLogger(__name__)


def generate_key() -> str:
    """Generates a random key.

    Returns:
        str: A random key.
    """
    return secrets.token_hex(20)


def generate_thumbnail(
    *, user_profile: models.UserProfile, size: tuple[int, int]
) -> None:
    """Generates a thumbnail for a user profile picture.

    This function opens the user's profile picture, resizes it into a thumbnail using
    bicubic resampling, converts the colorspace to RGB, and saves it as a WebP format.

    The thumbnail is saved in-memory before being added to the user's profile.
    If the user doesn't have a profile picture, their thumbnail is set to `None`.

    In case an error is encountered during opening or resizing of the image,
    it is logged and the same is raised again. If the error is about the image being unidentified,
    a validation error message is raised along with the original error.

    Args:
        user_profile (models.UserProfile): A UserProfile instance for which to generate
        a thumbnail.
        size (tuple[int, int]): A tuple containing the width and height of the thumbnail.

    Returns:
        None: This function does not return anything.

    Raises:
        UnidentifiedImageError: If the given image is unidentified.
        ValidationError: If image uploading has an issue due to it being invalid.
        Exception: If an unexpected error is raised during the thumbnail generation process.
    """

    # If the user doesn't have a profile picture, don't generate a thumbnail.
    if not user_profile.profile_picture:
        user_profile.thumbnail = None
        return

    try:
        # Open the image.
        img = Image.open(user_profile.profile_picture)

        # Optimize the image.
        optimized_img = optimize_image(img, size)

        # Save the thumbnail to the user's profile.
        user_profile.thumbnail = ContentFile(
            optimized_img.getvalue(),
            f"{user_profile.user.username}_thumbnail.webp",
        )

        # Close the opened resources.
        img.close()
        optimized_img.close()

    except* UnidentifiedImageError as exc:
        log.error(
            f"Uploaded image for {user_profile.user.username} is invalid. Exception: {exc}"
        )
        raise ValidationError(
            {"profile_picture": "The image is invalid. Please try again."},
            code="invalid",
        ) from exc
    except* Exception as exc:
        log.exception(f"Failed to generate thumbnail for {user_profile.user.username}.")
        raise exc
