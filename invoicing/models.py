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

import textwrap
import uuid

from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _

from utils.models import GenericModel


class InvoiceControl(GenericModel):
    """Stores the Invoice Control information for a related :model: `organization.Organization`
    model.

    The Invoice Control model stores the invoice control information for a related
    :model: `organization.Organization` model. The invoice control information includes
    the invoice number prefix, invoice due after days, invoice terms, invoice footer,
    and invoice logo.

    Attributes:
        id (UUIDField): Primary key and default value is a randomly generated UUID.
            Editable and unique.
        organization (OneToOneField): ForeignKey to the related organization model
            with a CASCADE on delete. Has a verbose name of "Organization" and
            related names of "billing_control".
        invoice_number_prefix (CharField): CharField with a max length of 10.
            Has a verbose name of "Invoice Number Prefix", help text of "Define a prefix for
            invoice numbers.", and a default value of "INV-".
        invoice_due_after_days (PositiveIntegerField): PositiveIntegerField with a default
            value of 30. Has a verbose name of "Invoice Due After Days", and help text of
            "Define the number of days after invoice date that an invoice is due."
        invoice_terms (TextField): TextField with a default value of "" and blank set to True.
            Has a verbose name of "Invoice Terms", and help text of "Define invoice terms."
        invoice_footer (TextField): TextField with a default value of "" and blank set to True.
            Has a verbose name of "Invoice Footer", and help text of "Define invoice footer."
        invoice_logo (ImageField): ImageField with a default value of None and blank set to True.
            Has a verbose name of "Invoice Logo", and help text of "Define invoice logo."
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    organization = models.OneToOneField(
        "organization.Organization",
        on_delete=models.CASCADE,
        verbose_name=_("Organization"),
        related_name="invoice_control",
    )
    invoice_number_prefix = models.CharField(
        _("Invoice Number Prefix"),
        max_length=10,
        help_text=_("Define a prefix for invoice numbers."),
        default="INV-",
    )
    credit_memo_number_prefix = models.CharField(
        _("Credit Note Number Prefix"),
        max_length=10,
        help_text=_("Define a prefix for credit note numbers."),
        default="CN-",
    )
    invoice_due_after_days = models.PositiveIntegerField(
        _("Invoice Due After Days"),
        default=30,
        help_text=_(
            "Define the number of days after invoice date that an invoice is due."
        ),
    )
    invoice_terms = models.TextField(
        _("Invoice Terms"),
        help_text=_("Define invoice terms."),
        default="",
        blank=True,
    )
    invoice_footer = models.TextField(
        _("Invoice Footer"),
        help_text=_("Define invoice footer."),
        default="",
        blank=True,
    )
    invoice_logo = models.ImageField(
        _("Invoice Logo"),
        upload_to="invoice_logo",
        help_text=_("Define invoice logo."),
        blank=True,
    )
    invoice_logo_width = models.PositiveIntegerField(
        _("Invoice Logo Width"),
        default=0,
        help_text=_("Define invoice logo width. (PX)"),
    )
    show_invoice_due_date = models.BooleanField(
        _("Show Invoice Due Date"),
        default=True,
        help_text=_("Show the invoice due date on the invoice."),
    )
    show_amount_due = models.BooleanField(
        _("Show Amount Due"),
        default=True,
        help_text=_("Show the amount due on the invoice."),
    )
    attach_pdf = models.BooleanField(
        _("Attach PDF"),
        default=True,
        help_text=_("Attach the invoice PDF to the invoice email."),
    )

    class Meta:
        """
        Metaclass for the InvoiceControl model.
        """

        verbose_name = _("Invoice Control")
        verbose_name_plural = _("Invoice Controls")

    def __str__(self) -> str:
        """
        Returns the string representation of the InvoiceControl model.

        Returns:
            String representation of the InvoiceControl model. For example,
            `Monta`
        """
        return textwrap.shorten(
            f"Invoice Control: {self.organization.name}", width=30, placeholder="..."
        )

    def get_absolute_url(self) -> str:
        """
        Returns the absolute url for the InvoiceControl model.

        Returns:
            Absolute url for the InvoiceControl object. For example,
            `/invoice-control/edd1e612-cdd4-43d9-b3f3-bc099872088b/`
        """
        return reverse("invoice-control-detail", kwargs={"pk": self.pk})
