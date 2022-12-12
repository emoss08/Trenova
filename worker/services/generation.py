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

from worker.models import Worker

class WorkerGenerationService:
    """Worker Generation Service

    Generate a unique code for the worker.
    """

    @staticmethod
    def generate_worker_code(instance: Worker) -> str:
        """Generate a unique code for the worker

        Args:
            instance (Worker): The worker instance.

        Returns:
            str: Worker code
        """
        code = f"{instance.first_name[0]}{instance.last_name[:5]}".upper()
        new_code = f"{code}{Worker.objects.count() + 1:04d}"

        # Check if the code already exists in the database
        if Worker.objects.filter(code=code).exists():
            return new_code
        else:
            return code
