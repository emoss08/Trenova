# -*- coding: utf-8 -*-
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

from typing import Any, Callable

from django.core.exceptions import ValidationError
from django.utils.deconstruct import deconstructible


@deconstructible
class MinimumSizeValidator:
    """Validate image dimensions

    Args:
        width (int): Width of the image.
        height (int): Height of the image.
    """

    def __init__(self, width: int, height: int) -> None:
        self.width = width
        self.height = height

    def __call__(self, image: Any) -> None:
        """Validator function to validate image dimensions

        Args:
            image (Any): Image to validate dimensions of.

        Returns:
            None

        Raises:
            ValidationError: If image dimension are too big for the field being validated.
        """
        error = False
        if self.width is not None and image.width < self.width:
            error = True
        if self.height is not None and image.height < self.height:
            error = True
        if error:
            raise ValidationError(
                [f"Size should be at least {self.width} x {self.height} pixels."]
            )

    def __eq__(self, other: Callable) -> bool:
        """Compare two validators. Inverse of __ne__.

        Args:
            other (Callable): Validator to compare to.

        Returns:
            bool: True if validators are equal, False otherwise.
        """
        return (
                isinstance(other, self.__class__)
                and self.width == other.width
                and self.height == other.height
        )

    def __ne__(self, other: Callable) -> bool:
        """Compare two validators. Inverse of __eq__.

        Args:
            other (Callable): Validator to compare to.

        Returns:
            bool: True if validators are not equal, False otherwise.
        """
        return not self.__eq__(other)
