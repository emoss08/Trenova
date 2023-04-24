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

import sys
import threading

from rich.console import Console
from rich.traceback import Traceback

console = Console()


def rich_traceback_hook(exc_type, exc_value, exc_traceback):
    """
    Formats and prints a rich traceback to the console.

    Args:
        exc_type: The type of the exception.
        exc_value: The value of the exception.
        exc_traceback: The traceback object for the exception.

    Returns:
        None
    """
    tb = Traceback.from_exception(exc_type, exc_value, exc_traceback, show_locals=True)
    console.print(tb)


class RichTracebackThread(threading.Thread):
    """
    A subclass of `threading.Thread` that overrides the `run` method to handle exceptions with the rich traceback hook.
    """

    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)

    def run(self):
        """
        Overrides the `run` method of the `threading.Thread` class to handle exceptions with the rich traceback hook.

        Returns:
            None
        """
        try:
            super().run()
        except Exception:
            sys.excepthook(*sys.exc_info())


sys.excepthook = rich_traceback_hook
threading.Thread = RichTracebackThread
