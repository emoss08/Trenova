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

from utils.models import GenericModel, ChoiceField
from django.utils.translation import gettext_lazy as _


class EDISegmentField(GenericModel):
    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
        verbose_name=_("ID"),
    )
    edi_segment = models.ForeignKey(
        "EDISegment",
        on_delete=models.CASCADE,
        related_name="fields",
        verbose_name=_("EDI Segment"),
        help_text=_("The EDI segment this field belongs to"),
    )
    model_field = models.CharField(
        _("Model Field"),
        max_length=255,
        help_text=_(
            "The name of the field on the BillingQueue model that this field maps to"
        ),
    )
    format = models.CharField(
        _("Format"),
        max_length=30,
        null=True,
        blank=True,
        help_text=_(
            "The format of the data in this field (e.g. 'MMDDYYYY' for a date field)"
        ),
    )
    position = models.PositiveIntegerField(
        _("Position"),
        help_text=_(
            "The position of this field in the EDI segment (e.g. 1 for the first field)"
        ),
    )

    class Meta:
        """
        Meta options for the EDI Segment Field model
        """

        db_table = "edi_segment_field"
        verbose_name = _("EDI Segment Field")
        verbose_name_plural = _("EDI Segment Fields")
        ordering = ["position"]


class EDISegment(GenericModel):
    """Stores information related to :model:`edi.EDISegment`

    Defines reusable parsing configuration for EDI document segments. Matches
    segments by code and maps them to parser functions to extract data. The
    extracted data is mapped to invoice fields.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    code = models.CharField(
        _("Code"),
        max_length=10,
        help_text=_(
            "The segment code found in the EDI file for this segment (e.g. N1)"
        ),
    )
    name = models.CharField(
        _("Name"),
        max_length=30,
        help_text=_(
            "A human readable name that describes the segment (e.g. Supplier Name)"
        ),
    )
    parser = models.CharField(
        _("Parser"),
        max_length=100,
        help_text=_("Format string for the segment (e.g. 'N1*%s*%s*%s*%s')"),
    )
    sequence = models.PositiveSmallIntegerField(
        _("Sequence"),
        help_text=_(
            "The sequence in which the segment should appear in the EDI document"
        ),
    )
    is_required = models.BooleanField(
        _("Is Required"),
        default=False,
        help_text=_("Whether or not this segment is required in the EDI document"),
    )

    class Meta:
        """
        Meta options for the EDI Segment model
        """

        ordering = ["sequence"]
        verbose_name = _("EDI Segment")
        verbose_name_plural = _("EDI Segments")
        db_table = "edi_segment"

    def __str__(self) -> str:
        """EDI Segment as string

        Returns:
            str: String representation of the EDI Segment
        """
        return textwrap.shorten(
            f"{self.code} - {self.name}", width=50, placeholder="..."
        )

    def get_absolute_url(self) -> str:
        """EDI Segment Absolute URL

        Returns:
            str: Absolute URL of the EDI Segment
        """
        return reverse("edi-segment-details", kwargs={"pk": self.pk})


class EDIBillingProfile(GenericModel):
    """Stores information related to :model:`edi.EDIBillingProfile`

    Contains configuration for generating EDI invoices for a customer. References
    the customer and defines EDI-specific fields like envelope IDs, formats,
    acknowledgments, processing settings, etc.
    """

    @final
    class EDIFormatChoices(models.TextChoices):
        X12 = "X12", _("X12")
        EDIFACT = "EDIFACT", _("EDIFACT")

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    customer = models.ForeignKey(
        "customer.Customer",
        verbose_name=_("Customer"),
        related_name="edi_billing_profiles",
        related_query_name="edi_billing_profile",
        on_delete=models.RESTRICT,
        help_text=_("The customer this billing profile is for"),
    )
    edi_enabled = models.BooleanField(
        _("EDI Enabled"),
        default=False,
        help_text=_("Whether or not EDI is enabled for this customer"),
    )
    edi_format = ChoiceField(
        _("EDI Format"),
        help_text=_("The EDI format to use for this customer"),
        choices=EDIFormatChoices.choices,
    )
    destination = models.URLField(
        _("Destination"),
        max_length=255,
        help_text=_("The URL to send the EDI file to"),
        blank=True,
        null=True,
    )
    username = models.CharField(
        _("Username"),
        max_length=255,
        blank=True,
        help_text=_("Username for the destination"),
    )
    password = models.CharField(
        _("Password"),
        max_length=255,
        blank=True,
        help_text=_("Password for the destination"),
    )
    segments = models.ManyToManyField(
        EDISegment,
        verbose_name=_("Segments"),
        help_text=_("The segments to include in the EDI file"),
    )
    edi_isa_id = models.CharField(
        max_length=15,
        verbose_name=_("ISA ID"),
        help_text=_(
            "Interchange Sender ID, used in the ISA (Interchange Control Header) segment of the EDI file."
        ),
    )
    edi_gs_id = models.CharField(
        max_length=15,
        verbose_name=_("GS ID"),
        help_text=_(
            "Functional Group Sender ID, used in the GS (Functional Group Header) segment."
        ),
    )
    edi_version = models.CharField(
        max_length=4,
        verbose_name=_("EDI Version"),
        help_text=_(
            "Represents the version of the EDI standards you're using (e.g., 4010, 5010 for X12)"
        ),
    )
    edi_test_mode = models.BooleanField(
        default=False,
        verbose_name=_("Test Mode"),
        help_text=_("Whether the EDI document is for testing purposes."),
    )
    edi_functional_ack = models.BooleanField(
        verbose_name=_("Functional ACK"),
        help_text=_(
            "Indicates whether a functional acknowledgment (997 or 999) is expected in return."
        ),
    )
    edi_ta1_timeout = models.PositiveIntegerField(
        verbose_name=_("TA1 Timeout"),
        help_text=_(
            "Timeout in seconds for TA1 (Interchange Acknowledgment) response."
        ),
    )
    edi_997_ack = models.BooleanField(
        verbose_name=_("997 ACK"),
        help_text=_("Whether a 997 acknowledgment is expected in return."),
    )
    edi_gs3_receiver_id = models.CharField(
        max_length=15,
        verbose_name=_("GS03 Receiver ID"),
        help_text=_("Functional Group Receiver ID, used in the GS segment."),
    )
    edi_gs2_application_sender_id = models.CharField(
        max_length=15,
        verbose_name=_("GS02 Sender ID"),
        help_text=_("Application Sender ID, also used in the GS segment."),
    )
    edi_isa_authority = models.CharField(
        max_length=2,
        verbose_name=_("ISA Authority"),
        help_text=_("Authorization Information Qualifier, used in the ISA segment."),
    )
    edi_isa_security = models.CharField(
        max_length=10,
        verbose_name=_("ISA Security"),
        help_text=_("Security Information Qualifier, used in the ISA segment."),
    )

    edi_isa_security_info = models.CharField(
        max_length=10,
        verbose_name=_("ISA Security Info"),
        help_text=_("Security Information, used in the ISA segment."),
    )

    edi_isa_interchange_id_qualifier = models.CharField(
        max_length=2,
        verbose_name=_("ISA Interchange ID Qualifier"),
        help_text=_("Interchange ID Qualifier, used in the ISA segment."),
    )

    edi_gs_application_receiver_id = models.CharField(
        max_length=15,
        verbose_name=_("GS Application Receiver ID"),
        help_text=_("Application Receiver's Code, used in the GS segment."),
    )

    edi_gs_code = models.CharField(
        max_length=2,
        verbose_name=_("GS Code"),
        help_text=_("Functional Identifier Code, used in the GS segment."),
    )

    edi_isa_receiver_id = models.CharField(
        max_length=15,
        verbose_name=_("ISA Receiver ID"),
        help_text=_("Interchange Receiver ID, used in the ISA segment."),
    )
    processing_settings = models.JSONField(
        blank=True,
        null=True,
        verbose_name=_("Processing Settings"),
        help_text=_("Additional settings for processing the EDI document."),
    )
    validation_settings = models.JSONField(
        blank=True,
        null=True,
        verbose_name=_("Validation Settings"),
        help_text=_("JSON dict with data validation rules"),
    )
    # transmission_log = models.ForeignKey(
    #     "TransmissionLog",
    #     on_delete=models.SET_NULL,
    #     null=True,
    #     blank=True,
    #     verbose_name="Transmission Log",
    #     help_text="Log of transmission results",
    # )
    # history = models.ManyToManyField(
    #     "EDIDocument", verbose_name="Document History", help_text="Historical EDI documents"
    # )
    #

    class Meta:
        """
        Meta options for EDI Billing Profile
        """

        verbose_name = _("EDI Billing Profile")
        verbose_name_plural = _("EDI Billing Profiles")
        db_table = "edi_billing_profile"

    def __str__(self) -> str:
        """EDI Billing Profile as string

        Returns:
            str: String representation of the EDI Billing Profile
        """
        return textwrap.shorten(
            f"{self.customer} - {self.edi_format}", width=50, placeholder="..."
        )

    def get_absolute_url(self) -> str:
        """EDI Billing Profile Absolute URL

        Returns:
            str: Absolute URL of the EDI Billing Profile
        """
        return reverse("edi-billing-profile-details", kwargs={"pk": self.pk})
