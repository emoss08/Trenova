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
from django.utils.timezone import now
from rich.console import Console
from rich.logging import RichHandler
from rich.text import Text


class RocketStyleLoggingHandler(RichHandler):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.console = Console()

    def format_message(self, record):
        timestamp = now().strftime("%Y-%m-%d %H:%M:%S")

        # Extract information based on whether record.args is a dictionary
        if isinstance(record.args, dict):
            # Dictionary expected from logger.info extra parameter
            method = record.args.get("method", "UNKNOWN")
            path = record.args.get("path", "UNKNOWN")
            status_code = record.args.get("status", "UNKNOWN")
            handler_name = record.args.get("handler", "UNKNOWN")
            duration = record.args.get("time_taken", "UNKNOWN")
            client = record.args.get("client", "UNKNOWN")
            remote_addr = client.split(":")[0] if ":" in client else client
            size = record.args.get("size", "UNKNOWN")
        else:
            # Default values if record.args is not a dictionary
            method = (
                path
            ) = status_code = handler_name = duration = remote_addr = size = "UNKNOWN"

        outcome = (
            "Success"
            if str(status_code).startswith("2") or str(status_code).startswith("3")
            else "Failure"
        )

        text = Text.assemble(
            (timestamp, "bold dim"),
            " ",
            (method, "bold blue"),
            " ",
            (path, "white"),
            "\n",
            ("=> Matched: ", "bold"),
            (f"{method} {path}", "bold yellow"),
            " ",
            (f"({handler_name})", "bold yellow"),
            "\n",
            ("=> Outcome: ", "bold"),
            (outcome, "green" if outcome == "Success" else "red"),
            "\n",
            ("=> Status: ", "bold"),
            (str(status_code), "green" if outcome == "Success" else "red"),
            "\n",
            ("=> Duration: ", "bold"),
            (f"{duration}s", "white"),
            "\n",
            ("=> Remote Address: ", "bold"),
            (remote_addr, "white"),
            "\n",
            ("=> Size: ", "bold"),
            (f"{size} bytes", "white"),
            style="white",
        )
        return text

    def emit(self, record):
        message = self.format_message(record)
        self.console.print(message)
