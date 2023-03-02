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

import locale
import shutil
import socket
from timeit import default_timer as timer
from typing import TypeAlias

host = socket.gethostname()
locale.setlocale(locale.LC_ALL, "en_US.UTF-8")

# Type aliases
HealthStatus: TypeAlias = dict[str, str | int | int | int]
HealthStatusAndTime: TypeAlias = dict[str, str | int | int | int | float | float]
DiskUsage: TypeAlias = tuple[int, int, int]


class DiskUsageHealthCheck:
    """
    Class to check the disk usage of the system.
    """

    @staticmethod
    def compare_disk_usage() -> DiskUsage:
        """
        Get the total, used, and free disk space in gigabytes.

        Returns:
            Tuple[int, int, int]: A tuple containing the total, used, and free disk space in gigabytes.
        """
        total, used, free = shutil.disk_usage("/")
        total = total // (2**30)
        used = used // (2**30)
        free = free // (2**30)
        return total, used, free

    def check_disk_usage(self) -> HealthStatus:
        """
        Check the disk usage and return a dictionary indicating the status and disk usage information.

        Returns:
            HealthStatus: A dictionary containing the disk usage status and the total, used, and free disk space in gigabytes.
        """
        total, used, free = self.compare_disk_usage()
        if free < 5:
            return {"status": "Critical", "total": total, "used": used, "free": free}
        return (
            {"status": "Low", "total": total, "used": used, "free": free}
            if free < 10
            else {"status": "Online", "total": total, "used": used, "free": free}
        )

    def check_disk_usage_and_time(self) -> HealthStatusAndTime:
        """
        Check the disk usage and time taken to get the disk usage information and return a dictionary
        indicating the status, disk usage information, and time taken.

        Returns:
            HealthStatusAndTime: A dictionary containing the disk usage status, the total,
            used, and free disk space in gigabytes, and the time taken to get the disk usage information.
        """
        start = timer()
        total, used, free = self.compare_disk_usage()
        end = timer()
        if free < 5:
            return {
                "status": "Critical",
                "total": total,
                "used": used,
                "free": free,
                "time": end - start,
            }
        if free < 10:
            return {
                "status": "Low",
                "total": total,
                "used": used,
                "free": free,
                "time": end - start,
            }
        return (
            {
                "status": "Slow",
                "total": total,
                "used": used,
                "free": free,
                "time": end - start,
            }
            if end - start > 0.01
            else {
                "status": "Online",
                "total": total,
                "used": used,
                "free": free,
                "time": end - start,
            }
        )
