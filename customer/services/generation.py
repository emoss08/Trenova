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

from customer.models import Customer


class CustomerGenerationService:
    """Customer Generation Service

    Generate a unique code for the customer.
    """

    @staticmethod
    def customer_code(*, instance: Customer) -> str:
        """Generate a unique code for the customer

        Args:
            instance (Customer): Customer instance

        Returns:
            str: Customer code
        """
        code = instance.name[:3].upper()
        new_code = f"{code}{Customer.objects.count() + 1:04d}"

        return new_code if Customer.objects.filter(code=code).exists() else code
