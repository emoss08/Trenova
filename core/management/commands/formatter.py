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

from django.core.management.base import BaseCommand, CommandError


class Command(BaseCommand):
    """
    Command to format Python code using Black, isort, and pyupgrade.
    """

    help = "Format Python code using Black, isort"

    def handle(self, *args, **options) -> None:
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

        # Black
        try:
            subprocess.run(["black", "."], check=True)
        except subprocess.CalledProcessError as error:
            raise CommandError("Black failed.") from error

        # isort
        try:
            subprocess.run(["isort", ".", "--profile", "django"])
        except subprocess.CalledProcessError as error:
            raise CommandError("isort failed.") from error

        # send a git commit
        try:
            subprocess.run(["git", "add", "."])
            subprocess.run(["git", "commit", "-a", "-m", "Monta auto-formatting"])
            subprocess.run(["git", "push"])
        except subprocess.CalledProcessError as error:
            raise CommandError("git commit failed.") from error
