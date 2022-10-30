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


class MontaCoreException(Exception):
    """
    Base class for all exceptions to be raised by Monta.
    """

    def __init__(self, message: str, status_code: int) -> None:
        """This is the constructor for the AuthenticationError class.

        Args:
            message (str): The error message
            status_code (int): The status code of the error

        Returns:
            None
        """
        self.message: str = message
        self.status_code: int = status_code
        super().__init__(self.message, self.status_code)

    def __str__(self) -> str:
        """String representation of the exception.

        Returns:
            str: The string representation of the exception
        """
        return f"{self.message} (Status Code: {self.status_code})"
