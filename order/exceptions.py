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

from utils.exceptions import MontaCoreException


class RatingException(MontaCoreException):
    """
    Base class for all rating exceptions to be raised by Monta.
    """

    def __init__(self, message: str, status_code: int) -> None:
        """This is the constructor for the RatingException class.

        Args:
            message (str): The error message
            status_code (int): The status code of the error

        Returns:
            None
        """
        super().__init__(message, status_code)


class SequenceException(MontaCoreException):
    """
    Base class for sequencing exceptions to be raised by Monta.
    """

    def __init__(self, message: str, status_code: int) -> None:
        """This is the constructor for the SequenceException class.

        Args:
            message (str): The error message
            status_code (int): The status code of the error

        Returns:
            None
        """
        super().__init__(message, status_code)
