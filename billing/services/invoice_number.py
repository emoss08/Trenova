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

    def __init__(self, *, instance: models.BillingQueue):
        self.instance = instance
        self.invoice_number()

    def invoice_number(self) -> str:
        """Generate a unique invoice number

        Returns:
            str: Invoice number
        """
        if not self.instance.invoice_number:
            if latest_invoice := models.BillingQueue.objects.order_by(
                "invoice_number"
            ).last():
                latest_invoice_number = int(
                    latest_invoice.invoice_number.split(
                        self.instance.organization.invoice_control.invoice_number_prefix
                    )[-1]
                )
                self.instance.invoice_number = "{}{:05d}".format(
                    self.instance.organization.invoice_control.invoice_number_prefix,
                    latest_invoice_number + 1,
                )
            else:
                self.instance.invoice_number = f"{self.instance.organization.invoice_control.invoice_number_prefix}00001"

        return self.instance.invoice_number
