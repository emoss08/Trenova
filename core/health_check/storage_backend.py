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

from django.core.files.base import ContentFile
from django.core.files.storage import default_storage


class FileStorageHealthCheck:
    """
    Class to check the health of file storage in Django.
    """

    @staticmethod
    def check_file_storage() -> dict:
        """
        Check the health of the file storage by writing and reading a file.

        Returns:
            str: A string indicating the health of the file storage.
        """
        test_file_content = "test_content"
        test_file_name = "test_file.txt"

        start = timer()
        try:
            # Write a test file
            default_storage.save(test_file_name, ContentFile(test_file_content))

            # Read the test file
            file = default_storage.open(test_file_name, "r")
            content = file.read()
            file.close()

            # Check the contents of the test file
            if content != test_file_content:
                end = timer()
                return {"status": "Corrupted", "time": end - start}
        except Exception:
            end = timer()
            return {"status": "Offline", "time": end - start}
        finally:
            # Delete the test file
            if default_storage.exists(test_file_name):
                default_storage.delete(test_file_name)

        end = timer()
        return {"status": "Online", "time": end - start}
