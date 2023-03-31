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

from typing import Any, Union

from django.core.exceptions import ImproperlyConfigured, ValidationError
from django.utils.deconstruct import deconstructible


@deconstructible
class ImageSizeValidator:
    """Validate image dimensions

    Args:
        width (int): Width of the image.
        height (int): Height of the image.
        less_than (bool): If True, image dimensions must be less than width and height.
        greater_than (bool): If True, image dimensions must be greater than width and height.
    """

    def __init__(
        self,
        width: int,
        height: int,
        less_than: bool | None,
        greater_than: bool | None,
    ) -> None:
        self.width = width
        self.height = height
        self.less_than = less_than
        self.greater_than = greater_than

    def __call__(self, image: Any) -> None:
        """Validator function to validate image dimensions

        Args:
            image: Image to validate dimensions of.

        Returns:
            None

        Raises:
            ValidationError: If image dimension are too big for the field being validated.
        """
        error = False

        if self.greater_than and self.less_than:
            raise ImproperlyConfigured(
                f"{self.__class__.__name__} cannot be used with both "
                "greater_than and less_than set to True."
            )

        if self.less_than:
            if self.width is not None:
                error = True
            if self.height is not None:
                error = True
            if error:
                raise ValidationError(
                    [
                        f"Size should be greater than {self.width} x {self.height} pixels. Please Try Again."
                    ]
                )

        if self.greater_than:
            if self.width is not None:
                error = True
            if self.height is not None:
                error = True
            if error:
                raise ValidationError(
                    [
                        f"Size should be less than {self.width} x {self.height} pixels. Please try again."
                    ]
                )

    def __eq__(self, other: object) -> bool:
        """Compare two validators. Inverse of __ne__.

        Args:
            other (object): Validator to compare to.

        Returns:
            bool: True if validators are equal, False otherwise.
        """
        return (
            (  # type: ignore
                self.width == other.width
                and self.height == other.height
                and self.less_than == other.less_than
                and self.greater_than == other.greater_than
            )
            if isinstance(other, ImageSizeValidator)
            else NotImplemented
        )

    def __ne__(self, other: object) -> bool:
        """Compare two validators. Inverse of __eq__.

        Args:
            other (object): Validator to compare to.

        Returns:
            bool: True if validators are not equal, False otherwise.
        """
        return not (self == other)
