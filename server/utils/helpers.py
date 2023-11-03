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

import io
import time
from functools import wraps

from dateutil.parser import parse
from django.db import models
from PIL import Image

from organization.models import BusinessUnit


def get_or_create_business_unit(*, bs_name: str) -> BusinessUnit:
    business_unit: BusinessUnit
    created: bool

    business_unit, created = BusinessUnit.objects.get_or_create(name=bs_name)
    return business_unit


def get_pk_value(*, instance):
    pk_field = instance._meta.pk.name
    pk = getattr(instance, pk_field, None)

    # Check to make sure that we got an pk not a model object.
    if isinstance(pk, models.Model):
        pk = get_pk_value(instance=pk)
    return pk


def convert_to_date(date_str: str) -> str:
    """Convert an ISO 8601 string to a date string."""
    try:
        return parse(date_str).date().isoformat()
    except ValueError:
        return date_str


def optimize_image(img: Image.Image, size: tuple[int, int]) -> io.BytesIO:
    """Optimizes an image by resizing it and converting it to WEBP format.

    Args:
        img (Image.Image): An opened PIL Image instance.
        size (tuple[int, int]): A tuple containing the width and height to resize the image to.

    Returns:
        io.BytesIO: BytesIO object containing the optimized image.
    """

    # Resize the image.
    img = img.resize(size, resample=Image.Resampling.BICUBIC)

    # Convert the image to RGB.
    img = img.convert("RGB")

    # Create a buffer to store the optimized image.
    output = io.BytesIO()

    # Save the image in WEBP format.
    img.save(output, "webp")

    # Seek to the beginning of the file.
    output.seek(0)

    return output


def calc_time(func):
    @wraps(func)
    def timeit_wrapper(*args, **kwargs):
        start_time = time.perf_counter()
        result = func(*args, **kwargs)
        end_time = time.perf_counter()
        total_time = end_time - start_time
        print(f"Function {func.__name__}{args} {kwargs} Took {total_time:.4f} seconds")
        return result

    return timeit_wrapper
