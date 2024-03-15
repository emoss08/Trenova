# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2024 Trenova                                                                       -
#                                                                                                  -
#  This file is part of Trenova.                                                                   -
#                                                                                                  -
#  The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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

import textwrap
import uuid
from typing import final

from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _

from utils.models import ChoiceField, GenericModel


class InvoiceControl(GenericModel):
    """Stores the Invoice Control information for a related :model: `organization.Organization`
    model.

    The Invoice Control model stores the invoice control information for a related
    :model: `organization.Organization` model. The invoice control information includes
    the invoice number prefix, invoice due after days, invoice terms, invoice footer,
    and invoice logo.
    """

    @final
    class DateFormatChoices(models.TextChoices):
        """Invoice Date Format Choices

        The Invoice Date Format Choices class is a TextChoices class that defines the
        invoice date format choices.

        Attributes:
            MM_DD_YYYY (str): MM/DD/YYYY
            DD_MM_YYYY (str): DD/MM/YYYY
        """

        MM_DD_YYYY = "%m/%d/%Y", _("MM/DD/YYYY")
        DD_MM_YYYY = "%d/%m/%Y", _("DD/MM/YYYY")
        YYYY_DD_MM = "%Y/%d/%m", _("YYYY/DD/MM")
        YYYY_MM_DD = "%Y/%m/%d", _("YYYY/MM/DD")

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
        help_text=_("Define invoice logo width. (PX)"),
        default=0,
    )
    show_invoice_due_date = models.BooleanField(
        _("Show Invoice Due Date"),
        default=True,
        help_text=_("Show the invoice due date on the invoice."),
    )
    invoice_date_format = ChoiceField(
        _("Invoice Date Format"),
        choices=DateFormatChoices.choices,
        default=DateFormatChoices.MM_DD_YYYY,
        help_text=_("Define the invoice date format."),
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
        db_table = "invoice_control"

    def __str__(self) -> str:
        """
        Returns the string representation of the InvoiceControl model.

        Returns:
            String representation of the InvoiceControl model. For example,
            `Trenova`
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
