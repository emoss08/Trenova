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

import requests
from django.core.management.base import BaseCommand
from rich.console import Console
from rich.progress import Progress

from location import models


class Command(BaseCommand):
    help = "Preloads the states from CountriesNow API"

    def handle(self, *args: typing.Any, **options: typing.Any) -> None:
        r = requests.post(
            "https://countriesnow.space/api/v0.1/countries/states",
            json={"country": "United States"},
        )
        data = r.json()
        states = data["data"]["states"]

        existing_states = set(
            models.States.objects.values_list("abbreviation", flat=True)
        )
        new_states = [
            state for state in states if state["state_code"] not in existing_states
        ]

        state_objects = [
            models.States(
                name=state["name"],
                abbreviation=state["state_code"],
                country_name=data["data"]["name"],
                country_iso3=data["data"]["iso3"],
            )
            for state in new_states
        ]

        console = Console()
        with Progress() as progress:
            task = progress.add_task("[green]Loading States...", total=len(new_states))

            for _ in new_states:
                progress.advance(task)

        if state_objects:
            models.States.objects.bulk_create(state_objects)
            console.print(f"{len(state_objects)} new states loaded.", style="green")
        else:
            console.print("No new states to load.", style="yellow")

        self.stdout.write(self.style.SUCCESS("States preloading completed."))
