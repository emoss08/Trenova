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

import logging
import re

from rich.logging import RichHandler
from rich.text import Text


class CustomRichHandler(RichHandler):
    def level_styles(self, levelno):
        if levelno >= logging.CRITICAL:
            return "bold red"
        if levelno >= logging.ERROR:
            return "bold red"
        if levelno >= logging.WARNING:
            return "bold yellow"
        if levelno >= logging.INFO:
            return "bold blue"
        return "green" if levelno >= logging.DEBUG else "dim"

    def method_style(self, method):
        if method == "GET":
            return "green"
        elif method == "POST":
            return "yellow"
        elif method in ["PUT", "PATCH"]:
            return "cyan"
        elif method == "DELETE":
            return "red"
        return "white"

    def emit(self, record):
        message = self.format(record)
        level_name = record.levelname
        level_text = f"{level_name}: "
        message = message.replace(level_name, "", 1).strip()

        http_pattern = re.compile(
            r"(HTTP) (\w+) (/.+?) (\d+) \[(.*?)] => Outcome: (\w+)"
        )
        match = http_pattern.search(message)

        text = Text()
        if match:
            self.show_http_message(match, text)
        else:
            text.append(level_text, style=self.level_styles(record.levelno))
            text.append(message, style="white")

        self.console.print(text)

    def show_http_message(self, match, text):
        method = match[2]
        path = match[3]
        status = int(match[4])
        handler_name = match[5]
        outcome = match[6]

        text.append(method, style=self.method_style(method))
        text.append(f" {path}: ", style="bold blue")
        text.append("\n    => Matched: ", style="bold white")
        text.append(f"{method} {path} ", style="bold orange")
        text.append(f"[{handler_name}]", style="bold yellow")
        text.append(f"\n    => Outcome: ", style="bold white")
        if status >= 200 and status < 400:
            text.append(f"{outcome}.", style="bold green")
            text.append("\n    => ", style="bold white")
            text.append("Response Succeeded.", style="bold green")
        else:
            text.append(f"{outcome}.", style="bold red")
            text.append("\n    => ", style="bold white")
            text.append("Response Failed.", style="bold red")
