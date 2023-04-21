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
import typing
import os
import tempfile
import zipfile
from time import sleep

from django.core.management import call_command
from django.core.management.base import BaseCommand, CommandParser
from urllib.request import urlretrieve
from rich.progress import Progress

from django.conf import settings
from backend.settings import BASE_DIR, INSTALLED_APPS
from plugin.models import Plugin
import importlib
from django import setup


class Command(BaseCommand):
    help = "Installs a plugin from a GitHub URL"

    def add_arguments(self, parser: CommandParser):
        parser.add_argument(
            "github_url", type=str, help="GitHub URL of the plugin to install"
        )

    def handle(self, *args: typing.Any, **options: typing.Any) -> None:
        github_url = options["github_url"]

        temp_dir = tempfile.mkdtemp()
        with Progress() as progress:
            # Step 1: Downloading plugin
            download_task = progress.add_task("[cyan]Downloading plugin...", total=100)
            progress.start_task(download_task)
            zip_file_path, _ = urlretrieve(
                github_url, os.path.join(temp_dir, os.path.basename(github_url))
            )
            progress.update(download_task, completed=100)

            # Step 2: Unzipping plugin
            progress.update(download_task, description="[cyan]Unzipping plugin...")
            with zipfile.ZipFile(zip_file_path, "r") as zip_ref:
                zip_ref.extractall(BASE_DIR)

            plugin_name = os.path.splitext(os.path.basename(zip_file_path))[0]

            # Check if the plugin already exists in INSTALLED_APPS
            if plugin_name in INSTALLED_APPS:
                progress.stop()
                self.stdout.write(
                    self.style.ERROR(f"{plugin_name} is already in INSTALLED_APPS.")
                )
                return

            # Step 4: Adding plugin to settings.py
            progress.update(
                download_task, description="[cyan]Adding plugin to settings..."
            )
            settings_file_path = os.path.join(
                BASE_DIR, "backend", "settings.py"
            )  # Replace 'my_project' with your project's name
            with open(settings_file_path, "a") as settings_file:
                settings_file.write(f"\nINSTALLED_APPS.append('{plugin_name}')\n")

            # Reload settings.py module to recognize the new plugin
            settings_module = importlib.import_module("backend.settings")
            importlib.reload(settings_module)

            # Call Django's setup() function
            setup()

            Plugin.objects.update_or_create(name=plugin_name, github_url=github_url)

            # Step 5: Applying migrations
            progress.update(
                download_task, description="[cyan]Applying migrations for plugin..."
            )

            call_command("makemigrations", plugin_name)
            call_command("migrate", plugin_name)

            progress.update(
                download_task,
                description="[cyan]Plugin successfully installed!",
                completed=100,
            )
