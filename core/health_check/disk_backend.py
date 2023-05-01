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

import locale
import shutil
import socket
from timeit import default_timer as timer

from utils.types import DiskUsage, HealthStatus, HealthStatusAndTime

host = socket.gethostname()
locale.setlocale(locale.LC_ALL, "en_US.UTF-8")


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
