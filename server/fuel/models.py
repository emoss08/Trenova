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

import textwrap
import uuid
from typing import final

from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _

from utils.models import ChoiceField, GenericModel


class FuelVendor(GenericModel):
    """
    Stores Fuel Vendor details related to an :model:`organization.Organization`
    """

    @final
    class CommunicationTypeChoices(models.TextChoices):
        """
        Communication Type Choices
        """

        FTP = "ftp", _("File Transfer Protocol")
        SFTP = "sftp", _("Secure File Transfer Protocol")
        LOCAL = "local", _("Local File System")
        HTTPS = "https", _("Hypertext Transfer Protocol Secure")

    @final
    class CommunicationModeChoices(models.TextChoices):
        """
        Communication Mode Choices
        """

        PUSH = "push", _("Push")
        PULL = "pull", _("Pull")
        PUSH_PULL = "push_pull", _("Push and Pull")

    id = models.UUIDField(
        _("ID"),
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        help_text=_("Unique identifier for the vendor"),
    )
    name = models.CharField(
        _("Name"),
        max_length=255,
        help_text=_("Name of the vendor"),
    )
    account_code = models.CharField(
        _("Account Code"),
        max_length=255,
        help_text=_("Account Code for the vendor"),
    )
    sub_account_code = models.CharField(
        _("Sub Account Code"),
        max_length=255,
        help_text=_("Sub Account Code for the vendor"),
    )
    communication_type = ChoiceField(
        _("Communication Type"),
        choices=CommunicationTypeChoices.choices,
        default=CommunicationTypeChoices.FTP,
        help_text=_("Communication type used to connect to the vendor"),
    )
    login = models.CharField(
        _("Login"),
        max_length=255,
        help_text=_("Login for the vendor"),
        blank=True,
    )
    password = models.CharField(
        _("Password"),
        max_length=255,
        help_text=_("Password for the vendor"),
        blank=True,
    )
    port = models.PositiveIntegerField(
        _("Port"),
        help_text=_("Port for the vendor"),
        blank=True,
        null=True,
    )
    server_address = models.CharField(
        _("Server Address"),
        max_length=255,
        help_text=_("Server address for the vendor"),
        blank=True,
    )
    directory = models.CharField(
        _("Directory"),
        max_length=255,
        help_text=_("Directory for the vendor"),
        blank=True,
    )
    proxy_server = models.CharField(
        _("Proxy Server"),
        max_length=255,
        help_text=_("Proxy server for the vendor"),
        blank=True,
    )
    email_address = models.EmailField(
        _("Email Address"),
        help_text=_("Email address for the vendor"),
        blank=True,
    )
    communication_mode = ChoiceField(
        _("Communication Mode"),
        choices=CommunicationModeChoices.choices,
        default=CommunicationModeChoices.PUSH,
        help_text=_("Communication mode used to connect to the vendor"),
        blank=True,
    )

    class Meta:
        """
        Metaclass for Vendor
        """

        verbose_name = _("Fuel Vendor")
        verbose_name_plural = _("Fuel Vendors")
        db_table = "fuel_vendor"

    def __str__(self) -> str:
        """String representation of the Vendor model

        Returns:
            str: String representation of the Vendor model
        """

        return textwrap.shorten(self.login, width=50, placeholder="...")

    def get_absolute_url(self) -> str:
        """Get absolute URL of the Vendor model

        Returns:
            str: Absolute URL of the Vendor model
        """

        return reverse("vendor-detail", kwargs={"pk": self.pk})


class FuelVendorFuelDetail(GenericModel):
    """
    Stores Fuel Vendor details related to an :model:`organization.Organization`
    """

    @final
    class ApVoucherChoices(models.TextChoices):
        """
        AP Voucher Choices
        """

        NONE = "none", _("None")
        REGULAR = "regular", _("Regular")
        MANUAL = "manual", _("Manual")

    id = models.UUIDField(
        _("ID"),
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        help_text=_("Unique identifier for the vendor"),
    )
    fuel_vendor = models.OneToOneField(
        FuelVendor,
        on_delete=models.CASCADE,
        related_name="fuel_vendor",
        help_text=_("Fuel Vendor"),
    )
    create_ap_voucher = ChoiceField(
        _("Create AP Voucher"),
        choices=ApVoucherChoices.choices,
        default=ApVoucherChoices.NONE,
        help_text=_("Create AP Voucher"),
    )
    ap_division_code = models.ForeignKey(
        "accounting.DivisionCode",
        on_delete=models.CASCADE,
        related_name="ap_division_code",
        help_text=_("AP Division Code"),
    )

    class Meta:
        """
        Metaclass for Vendor
        """

        verbose_name = _("Fuel Vendor Fuel Detail")
        verbose_name_plural = _("Fuel Vendor Fuel Details")
        db_table = "fuel_vendor_fuel_detail"

    def __str__(self) -> str:
        """String representation of the Vendor model

        Returns:
            str: String representation of the Vendor model
        """

        return textwrap.shorten(self.fuel_vendor.name, width=50, placeholder="...")

    def get_absolute_url(self) -> str:
        """Get absolute URL of the Vendor model

        Returns:
            str: Absolute URL of the Vendor model
        """

        return reverse("vendor-detail", kwargs={"pk": self.pk})
