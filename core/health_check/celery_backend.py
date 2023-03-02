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
        The status will be either `'Working'` if it is Working properly, `'Offline'` if it is not Working properly,
        or `'slow'` if it is taking too long to respond.

        Returns:
            Dict[str, Union[str, float]]: The result of the Celery health check, including the status and time taken to perform the check.
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
