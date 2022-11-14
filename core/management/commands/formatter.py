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

import subprocess
from typing import Any

from django.core.management.base import BaseCommand, CommandError


class Command(BaseCommand):
    """
    Command to format Python code using Black, isort, and pyupgrade.
    """

    help = "Format Python code using Black & isort"

    def handle(self, *args: Any, **options: Any) -> None:
        """
        Handle command.
        """
        try:
            import black
            import isort
        except ImportError as error:
            raise CommandError(
                "Please install black, isort, and pyupgrade to use this command."
            ) from error

        try:
            self.stdout.write(
                self.style.NOTICE("Formatting Python code using black...")
            )
            subprocess.run(["black", "--target-version=py311", "."], check=True)
        except subprocess.CalledProcessError as error:
            raise CommandError("Black failed.") from error

        try:
            self.stdout.write(
                self.style.NOTICE("Formatting Python code using isort...")
            )
            subprocess.run(["isort", "--profile", "black", "."])
            self.stdout.write(self.style.SUCCESS("Formatting complete."))
        except subprocess.CalledProcessError as error:
            raise CommandError("isort failed.") from error
