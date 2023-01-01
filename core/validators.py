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

from typing import Any, Optional

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
        less_than: Optional[bool],
        greater_than: Optional[bool],
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
                        f"Size should be greater than {self.width} x {self.height} pixels."
                    ]
                )

        if self.greater_than:
            if self.width is not None:
                error = True
            if self.height is not None:
                error = True
            if error:
                raise ValidationError(
                    [f"Size should be less than {self.width} x {self.height} pixels."]
                )

    def __eq__(self, other: object) -> bool:
        """Compare two validators. Inverse of __ne__.

        Args:
            other (object): Validator to compare to.

        Returns:
            bool: True if validators are equal, False otherwise.
        """
        if not isinstance(other, ImageSizeValidator):
            return NotImplemented
        return (
            self.width == other.width
            and self.height == other.height
            and self.less_than == other.less_than
            and self.greater_than == other.greater_than
        )

    def __ne__(self, other: object) -> bool:
        """Compare two validators. Inverse of __eq__.

        Args:
            other (object): Validator to compare to.

        Returns:
            bool: True if validators are not equal, False otherwise.
        """
        return not (self == other)
