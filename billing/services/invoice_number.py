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

from billing import models


class InvoiceNumberService:
    """Invoice Number Service

    Generate a unique invoice
    """

    @staticmethod
    def invoice_number(instance: models.BillingQueue) -> str:
        """Generate a unique invoice number

        Args:
            instance (models.BillingQueue): BillingQueue instance

        Returns:
            str: Invoice number
        """
        if not instance.invoice_number:
            if latest_invoice := models.BillingQueue.objects.order_by(
                "invoice_number"
            ).last():
                latest_invoice_number = int(
                    latest_invoice.invoice_number.split(
                        instance.organization.scac_code
                    )[-1]
                )
                instance.invoice_number = "{0}{1:05d}".format(
                    instance.organization.scac_code, latest_invoice_number + 1
                )
            else:
                instance.invoice_number = f"{instance.organization.scac_code}00001"

            return instance.invoice_number
