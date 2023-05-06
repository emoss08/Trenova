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

from timeit import default_timer as timer

from celery.exceptions import TaskRevokedError, TimeoutError

from .tasks import add


class CeleryHealthCheck:
    """
    The `CeleryHealthCheck` class is used to check the health of a Celery instance.
    """

    @staticmethod
    def check_celery() -> dict:
        """
        The `check_celery` method is used to check the health of a Celery instance.
        It returns a dictionary indicating the status and time taken to perform the check.
        The status will be either `'Working'` if it is Working properly, `'Offline'` if it
        is not Working properly, or `'slow'` if it is taking too long to respond.

        Returns:
            Dict[str, Union[str, float]]: The result of the Celery health check, including the status
            and time taken to perform the check.
        """
        start = timer()
        try:
            result = add.delay(3, 5)
            result.get(timeout=3)
            end = timer()
            result_time = end - start
            if result.result != 8:
                return {"status": "Offline", "time": result_time}
        except TimeoutError:
            end = timer()
            result_time = end - start
            return {"status": "Offline", "time": result_time}
        except TaskRevokedError:
            end = timer()
            result_time = end - start
            return {"status": "Offline", "time": result_time}
        return {"status": "Working", "time": result_time}
