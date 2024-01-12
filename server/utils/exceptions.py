# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2024 Trenova                                                                       -
#                                                                                                  -
#  This file is part of Trenova.                                                                   -
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


class TrenovaCoreException(Exception):
    """
    Base class for all exceptions to be raised by Trenova.
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


class TokenException(TrenovaCoreException):
    """
    Exception for token errors
    """

    def __init__(self, message: str) -> None:
        """This is the constructor for the AuthenticationError class.

        Args:
            message (str): The error message

        Returns:
            None
        """
        super().__init__(message, 401)


class InvalidTokenException(TokenException):
    """
    Exception for invalid token errors
    """


class DjangoCommandException(Exception):
    """
    Exception for Django Commands
    """
