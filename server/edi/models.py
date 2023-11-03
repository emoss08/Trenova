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

from django.core.exceptions import ValidationError
from django.core.validators import EmailValidator
from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _
from encrypted_model_fields.fields import EncryptedCharField

from utils.models import ChoiceField, GenericModel


@final
class DjangoFieldChoices(models.TextChoices):
    """
    Choices for the Django Field type
    """

    AUTO_FIELD = "AutoField", _("Auto Field")
    BLANK_CHOICE_DASH = "BLANK_CHOICE_DASH", _("Blank Choice Dash")
    BIG_AUTO_FIELD = "BigAutoField", _("Big Auto Field")
    BIG_INTEGER_FIELD = "BigIntegerField", _("Big Integer Field")
    BINARY_FIELD = "BinaryField", _("Binary Field")
    BOOLEAN_FIELD = "BooleanField", _("Boolean Field")
    CHAR_FIELD = "CharField", _("Char Field")
    COMMA_SEPARATED_INTEGER_FIELD = "CommaSeparatedIntegerField", _(
        "Comma Separated Integer Field"
    )
    DATE_FIELD = "DateField", _("Date Field")
    DATETIME_FIELD = "DateTimeField", _("Date Time Field")
    DECIMAL_FIELD = "DecimalField", _("Decimal Field")
    DURATION_FIELD = "DurationField", _("Duration Field")
    EMAIL_FIELD = "EmailField", _("Email Field")
    EMPTY = "Empty", _("Empty")
    FIELD = "Field", _("Field")
    FILE_PATH_FIELD = "FilePathField", _("File Path Field")
    FLOAT_FIELD = "FloatField", _("Float Field")
    GENERIC_IP_ADDRESS_FIELD = "GenericIPAddressField", _("Generic IP Address Field")
    IP_ADDRESS_FIELD = "IPAddressField", _("IP Address Field")
    INTEGER_FIELD = "IntegerField", _("Integer Field")
    NOT_PROVIDED = "NOT_PROVIDED", _("Not Provided")
    NULL_BOOLEAN_FIELD = "NullBooleanField", _("Null Boolean Field")
    POSITIVE_BIG_INTEGER_FIELD = "PositiveBigIntegerField", _(
        "Positive Big Integer Field"
    )
    POSITIVE_INTEGER_FIELD = "PositiveIntegerField", _("Positive Integer Field")
    POSITIVE_SMALL_INTEGER_FIELD = "PositiveSmallIntegerField", _(
        "Positive Small Integer Field"
    )
    SLUG_FIELD = "SlugField", _("Slug Field")
    SMALL_AUTO_FIELD = "SmallAutoField", _("Small Auto Field")
    SMALL_INTEGER_FIELD = "SmallIntegerField", _("Small Integer Field")
    TEXT_FIELD = "TextField", _("Text Field")
    TIME_FIELD = "TimeField", _("Time Field")
    URL_FIELD = "URLField", _("URL Field")
    UUID_FIELD = "UUIDField", _("UUID Field")


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
        blank=True,
        help_text=_(
            "The format of the data in this field (e.g. 'MMDDYYYY' for a date field)"
        ),
    )
    data_type = ChoiceField(
        _("Data Type"),
        choices=DjangoFieldChoices.choices,
        default=DjangoFieldChoices.CHAR_FIELD,
        help_text=_("The data type of the data in this field"),
    )
    validation_regex = models.CharField(
        _("Validation Regex"),
        max_length=255,
        blank=True,
        help_text=_(
            "A regular expression that can be used to validate the data in this field"
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

    def __str__(self) -> str:
        """EDI Segment Field as string

        Returns:
            str: String representation of the EDI Segment Field
        """
        return textwrap.shorten(
            f"{self.edi_segment.code} - {self.model_field}",
            width=50,
            placeholder="...",
        )

    def clean(self) -> None:
        """Validates the EDI Segment Field

        Raises:
            ValidationError: If the data type does not match the model field
        """
        from edi.helpers import validate_data_type

        super().clean()

        match, internal_type = validate_data_type(
            data_type=self.data_type, model_field=self.model_field
        )

        if not match:
            raise ValidationError(
                {
                    "data_type": _(
                        f"You selected {self.data_type} but the model field {self.model_field} is of type {internal_type}"
                    )
                }
            )


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
        max_length=60,
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
        """
        Choices for the EDI Format field
        """

        X12 = "X12", _("X12")
        EDIFACT = "EDIFACT", _("EDIFACT")

    @final
    class EDIDirectionChoices(models.TextChoices):
        """
        Choices for the EDI Direction field
        """

        OUTBOUND = "O", _("Outbound")
        INBOUND = "I", _("Inbound")

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
    direction = ChoiceField(
        _("Direction"),
        choices=EDIDirectionChoices.choices,
        default=EDIDirectionChoices.OUTBOUND,
        help_text=_("Whether this billing profile is for inbound or outbound EDI"),
    )
    edi_ship_only = models.BooleanField(
        _("Create Only for EDI Shipments"),
        default=False,
        help_text=_(
            "Whether or not to create EDI invoices only for shipments that were created via EDI"
        ),
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
    edi_comm_profile = models.ForeignKey(
        "EDICommProfile",
        related_name="edi_billing_profiles",
        related_query_name="edi_billing_profile",
        null=True,
        blank=True,
        on_delete=models.SET_NULL,
        help_text=_("The communication profile to use for this billing profile"),
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
    notes = models.TextField(
        _("Notes"),
        blank=True,
        help_text=_("Notes about this EDI billing profile"),
    )

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


class EDILocationMapping(GenericModel):
    """
    Stores edi location mapping information related to :model:`edi.EDIBillingProfile`
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
        verbose_name=_("ID"),
    )
    edi_billing_profile = models.ForeignKey(
        EDIBillingProfile,
        on_delete=models.CASCADE,
        related_name="edi_location",
        verbose_name=_("EDI Billing Profile"),
        help_text=_("The EDI Billing Profile this location belongs to"),
    )
    location = models.ForeignKey(
        "location.Location",
        on_delete=models.RESTRICT,
        related_name="location",
        related_query_name="locations",
        verbose_name=_("Location"),
        help_text=_("Internal Location code."),
    )
    partner_edi_code = models.CharField(
        max_length=50,
        verbose_name=_("Partner EDI Code"),
        help_text=_("Partner EDI Code."),
    )

    class Meta:
        """
        Metaclass for EDILocationMapping model
        """

        verbose_name = _("EDI Location Mapping")
        verbose_name_plural = _("EDI Location Mappings")
        db_table = "edi_mp_location"

    def __str__(self) -> str:
        """EDI Location string representation

        Returns:
            str: String representation of the EDI Location Mapping
        """
        return textwrap.shorten(
            f"{self.location} - {self.partner_edi_code}", width=50, placeholder="..."
        )

    def get_absolute_url(self) -> str:
        """EDI Location Absolute URL

        Returns:
            str: Absolute URL of the EDI Location
        """
        return reverse("edi-location-mapping-details", kwargs={"pk": self.pk})


class EDIBillToMapping(GenericModel):
    """
    Stores edi bill to mapping information related to :model:`edi.EDIBillingProfile`
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
        verbose_name=_("ID"),
    )
    edi_billing_profile = models.ForeignKey(
        EDIBillingProfile,
        on_delete=models.CASCADE,
        related_name="edi_bill_to",
        verbose_name=_("EDI Billing Profile"),
        help_text=_("The EDI Billing Profile this bill-to belongs to"),
    )
    customer = models.ForeignKey(
        "customer.Customer",
        on_delete=models.RESTRICT,
        related_name="customer",
        related_query_name="customers",
        verbose_name=_("Customer"),
        help_text=_("Internal Customer code."),
    )
    partner_edi_code = models.CharField(
        max_length=50,
        verbose_name=_("Partner EDI Code"),
        help_text=_("Partner EDI Code."),
    )

    class Meta:
        """
        Metaclass for EDIBillToMapping model
        """

        verbose_name = _("EDI Bill To Mapping")
        verbose_name_plural = _("EDI Bill To Mappings")
        db_table = "edi_mp_bill_to"

    def __str__(self) -> str:
        """EDI Bill To string representation

        Returns:
            str: String representation of the EDI Bill To Mapping
        """
        return textwrap.shorten(
            f"{self.customer} - {self.partner_edi_code}", width=50, placeholder="..."
        )

    def get_absolute_url(self) -> str:
        """EDI Bill To Absolute URL

        Returns:
            str: Absolute URL of the EDI Bill To
        """
        return reverse("edi-bill-to-mapping-details", kwargs={"pk": self.pk})


class EDICommodityMapping(GenericModel):
    """
    Stores edi commodity to mapping information related to :model:`edi.EDIBillingProfile`
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
        verbose_name=_("ID"),
    )
    edi_billing_profile = models.ForeignKey(
        EDIBillingProfile,
        on_delete=models.CASCADE,
        related_name="edi_commodity",
        verbose_name=_("EDI Billing Profile"),
        help_text=_("The EDI Billing Profile this location belongs to"),
    )
    commodity = models.ForeignKey(
        "commodities.Commodity",
        on_delete=models.RESTRICT,
        related_name="commodity",
        related_query_name="commodities",
        verbose_name=_("Commodity"),
        help_text=_("Internal Commodity code."),
    )
    partner_edi_code = models.CharField(
        max_length=50,
        verbose_name=_("Partner EDI Code"),
        help_text=_("Partner EDI Code."),
    )

    class Meta:
        """
        Metaclass for EDIBillToMapping model
        """

        verbose_name = _("EDI Commodity Mapping")
        verbose_name_plural = _("EDI Commodity Mappings")
        db_table = "edi_mp_commodity_to"

    def __str__(self) -> str:
        """EDI Commodity Mapping string representation

        Returns:
            str: String representation of the EDI Commodity Mapping
        """
        return textwrap.shorten(
            f"{self.commodity} - {self.partner_edi_code}", width=50, placeholder="..."
        )

    def get_absolute_url(self) -> str:
        """EDI Commodity Mapping Absolute URL

        Returns:
            str: Absolute URL of the EDI Commodity Mapping
        """
        return reverse("edi-commodity-mapping-details", kwargs={"pk": self.pk})


class EDIChargeCodeMapping(GenericModel):
    """
    Stores edi charge code to mapping information related to :model:`edi.EDIBillingProfile`
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
        verbose_name=_("ID"),
    )
    edi_billing_profile = models.ForeignKey(
        EDIBillingProfile,
        on_delete=models.CASCADE,
        related_name="edi_charge_code",
        verbose_name=_("EDI Billing Profile"),
        help_text=_("The EDI Billing Profile this location belongs to"),
    )
    accessorial_charge = models.ForeignKey(
        "billing.AccessorialCharge",
        on_delete=models.RESTRICT,
        related_name="accessorial_charge",
        related_query_name="accessorial_charges",
        verbose_name=_("Accessorial Charge Code"),
        help_text=_("Internal Accessorial Charge code."),
    )
    partner_edi_code = models.CharField(
        max_length=50,
        verbose_name=_("Partner EDI Code"),
        help_text=_("Partner EDI Code."),
    )

    class Meta:
        """
        Metaclass for EDIChargeCodeMapping model
        """

        verbose_name = _("EDI Charge Code Mapping")
        verbose_name_plural = _("EDI Charge Code Mappings")
        db_table = "edi_mp_charge_code_to"

    def __str__(self) -> str:
        """EDI Charge Code Mapping string representation

        Returns:
            str: String representation of the EDI Charge Code Mapping
        """
        return textwrap.shorten(
            f"{self.accessorial_charge} - {self.partner_edi_code}",
            width=50,
            placeholder="...",
        )

    def get_absolute_url(self) -> str:
        """EDI Charge Code Mapping Absolute URL

        Returns:
            str: Absolute URL of the EDI Charge Code Mapping
        """
        return reverse("edi-charge-code-mapping-details", kwargs={"pk": self.pk})


class EDIBillingValidation(GenericModel):
    """
    Stores edi billing validation information related to :model:`edi.EDIBillingProfile`
    """

    @final
    class TestPassChoices(models.TextChoices):
        """
        Choices for the Test Pass field
        """

        ALL_RULES = "ALL_RULES", _("All Rules are Satisfied")
        ANY_RULE = "ANY_RULE", _("Any One Rule Is Satisfied")

    @final
    class RuleTypeChoices(models.TextChoices):
        """
        Choices for the Rule Type field
        """

        FIELD = "FIELD", _("Field")
        SEGMENT = "SEGMENT", _("Segment")
        COMPOSITE = "COMPOSITE", _("Composite")
        LOOP = "LOOP", _("Loop")

    @final
    class TableNameChoices(models.TextChoices):
        """
        Choices for the Table Name field
        """

        EDI_BILL = "EDI_BILL", _("EDI Bill")
        SHIPMENT = "SHIPMENT", _("Shipment")
        STOP = "STOP", _("Stop")

    @final
    class OperatorChoices(models.TextChoices):
        """
        Choices for the Operator field
        """

        EQUALS = "EQUALS", _("Equals")
        LESS_THAN_OR_EQUAL_TO = "LESS_THAN_OR_EQUAL_TO", _("Less Than or Equal To")
        LESS_THEN = "LESS_THEN", _("Less Than")
        LESS_THAN_OR_GREATER_THAN = "LESS_THAN_OR_GREATER_THAN", _(
            "Less Than or Greater Than"
        )
        GREATER_THAN_OR_EQUAL_TO = "GREATER_THAN_OR_EQUAL_TO", _(
            "Greater Than or Equal To"
        )
        GREATER_THAN = "GREATER_THAN", _("Greater Than")
        IS_NULL = "IS_NULL", _("Is Null")
        IS_NOT_NULL = "IS_NOT_NULL", _("Is Not Null")
        ENDS_WITH = "ENDS_WITH", _("Ends With")
        STARTS_WITH = "STARTS_WITH", _("Starts With")
        CONTAINS = "CONTAINS", _("Contains")
        DOES_NOT_CONTAIN = "DOES_NOT_CONTAIN", _("Does Not Contain")
        HAS_LENGTH_OF = "HAS_LENGTH_OF", _("Has Length Of")
        CONFORMS_TO_REGEX = "CONFORMS_TO_REGEX", _("Conforms to Regex")
        WITHIN = "WITHIN", _("Within")
        HAS_NOT_MET = "HAS_NOT_MET", _("Has Not Met")

    @final
    class RuleActionChoices(models.TextChoices):
        """
        Choices for the Rule Action field
        """

        WARNING = "WARNING", _("Warning")
        ERROR = "ERROR", _("Error")
        STOP = "STOP", _("Stop")

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
        verbose_name=_("ID"),
    )
    edi_billing_profile = models.ForeignKey(
        EDIBillingProfile,
        on_delete=models.CASCADE,
        related_name="edi_billing_validation",
        verbose_name=_("EDI Billing Profile"),
        help_text=_("The EDI Billing Profile this validation belongs to"),
    )
    test_passes_when = ChoiceField(
        _("Test Passes When"),
        choices=TestPassChoices.choices,
        help_text=_("The condition for a test to pass"),
    )
    rule_type = ChoiceField(
        _("Rule Type"),
        choices=RuleTypeChoices.choices,
        help_text=_("The type of rule to use"),
    )
    table_name = ChoiceField(
        _("Table Name"),
        choices=TableNameChoices.choices,
        help_text=_("The table to use for this rule"),
    )
    operator = ChoiceField(
        _("Operator"),
        choices=OperatorChoices.choices,
        help_text=_("The operator to use for this rule"),
    )
    value = models.CharField(
        _("Value"),
        max_length=255,
        help_text=_("The value to compare using the operator"),
    )
    description = models.CharField(
        _("Rule Description"),
        max_length=255,
        help_text=_("A descriptive name for this rule"),
    )
    data_type = models.CharField(
        _("Data Type"),
        max_length=50,
        choices=[("INTEGER", "Integer"), ("STRING", "String"), ("FLOAT", "Float")],
        default="STRING",
    )
    complex_rules = models.JSONField(
        _("Complex Rules"),
        null=True,
        blank=True,
        help_text=_("JSON data for complex rule configurations"),
    )
    rule_action = ChoiceField(
        _("Rule Action"),
        choices=RuleActionChoices.choices,
        help_text=_("The action to take if this rule is not satisfied"),
    )
    error_message = models.CharField(
        _("Error Message"),
        max_length=255,
        help_text=_("The error message to display if this rule is not satisfied"),
    )

    class Meta:
        """
        Metaclass for EDIBillingValidation model
        """

        verbose_name = _("EDI Billing Validation")
        verbose_name_plural = _("EDI Billing Validations")
        db_table = "edi_billing_validation"

    def __str__(self) -> str:
        """EDI Billing Validation string representation

        Returns:
            str: String representation of the EDI Billing Validation
        """
        return textwrap.shorten(
            f"{self.description} - {self.rule_action}",
            width=50,
            placeholder="...",
        )

    def get_absolute_url(self) -> str:
        """EDI Billing Validation Absolute URL

        Returns:
            str: Absolute URL of the EDI Billing Validation
        """
        return reverse("edi-billing-validation-details", kwargs={"pk": self.pk})


class EDINotification(GenericModel):
    """
    Stores edi notification information related to :model:`edi.EDIBillingProfile`
    """

    @final
    class NotificationTypeChoices(models.TextChoices):
        """
        Choices for the Notification Type field
        """

        ERROR_REPORT = "ERROR_REPORT", _("Error Report")
        FUNCTIONAL_ACK = "FUNCTIONAL_ACK", _("Functional Acknowledgement")
        AUTO_RESOLUTION = "AUTO_RESOLUTION", _("Auto Resolution")
        HISTORY_SNAPSHOT = "HISTORY_SNAPSHOT", _("History Snapshot")

    @final
    class NotificationFormatChoices(models.TextChoices):
        """
        Choices for the Notification Format field
        """

        EMAIL = "EMAIL", _("Email")
        SMS = "SMS", _("SMS")
        WEBHOOK = "WEBHOOK", _("Webhook")
        PUSH_NOTIFICATION = "PUSH_NOTIFICATION", _("Push Notification")

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
        verbose_name=_("ID"),
    )
    edi_billing_profile = models.ForeignKey(
        EDIBillingProfile,
        on_delete=models.CASCADE,
        related_name="edi_notification",
        verbose_name=_("EDI Billing Profile"),
        help_text=_("The EDI Billing Profile this validation belongs to"),
    )
    notification_type = ChoiceField(
        _("Notification Type"),
        choices=NotificationTypeChoices.choices,
        help_text=_("The type of notification to send"),
    )
    notification_format = ChoiceField(
        _("Notification Format"),
        choices=NotificationFormatChoices.choices,
        help_text=_("The format of the notification"),
    )
    parameters = models.JSONField(
        _("Parameters"),
        null=True,
        blank=True,
        help_text=_("JSON data for notification parameters"),
    )
    recipients = models.CharField(
        _("Recipients"),
        max_length=255,
        help_text=_("Comma separated list of recipients"),
    )

    class Meta:
        """
        Metaclass for EDINotification model
        """

        verbose_name = _("EDI Notification")
        verbose_name_plural = _("EDI Notifications")
        db_table = "edi_notification"

    def __str__(self) -> str:
        """EDI Notification string representation

        Returns:
            str: String representation of the EDI Notification
        """
        return textwrap.shorten(
            f"{self.notification_type} - {self.notification_format}",
            width=50,
            placeholder="...",
        )

    def clean(self) -> None:
        """EDINotification clean method

        Returns:
            None: This function does not return anything.
        """
        super().clean()

        # Split the recipients by comma to get individual emails
        emails = [
            email.strip() for email in self.recipients.split(",") if email.strip()
        ]

        # Use Django's EmailValidator to validate each email
        validator = EmailValidator()
        for email in emails:
            try:
                validator(email)
            except ValidationError as exc:
                raise ValidationError(
                    {
                        "recipients": _(
                            f"{email} is not a valid email address. Please Try again."
                        )
                    },
                    code="invalid",
                ) from exc

    def get_absolute_url(self) -> str:
        """EDI Notification Absolute URL

        Returns:
            str: Absolute URL of the EDI Notification
        """
        return reverse("edi-notification-details", kwargs={"pk": self.pk})


class EDICommProfile(GenericModel):
    """
    Stores edi communication profile information related to :model:`edi.EDIBillingProfile`
    """

    @final
    class ProtocolChoices(models.TextChoices):
        """
        Choices for the Protocol field
        """

        FTP = "FTP", _("FTP")
        HTTP = "HTTP", _("HTTP")
        AS2 = "AS2", _("AS2")
        SFTP = "SFTP", _("SFTP")

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
        verbose_name=_("ID"),
    )
    name = models.CharField(
        verbose_name=_("Name"),
        max_length=100,
        help_text=_("A descriptive name for this communication profile"),
    )
    is_active = models.BooleanField(
        verbose_name=_("Is Active"),
        default=False,
        help_text=_("Whether or not this communication profile is active"),
    )
    protocol = ChoiceField(
        _("Protocol"),
        choices=ProtocolChoices.choices,
        help_text=_("The protocol to use for this communication profile"),
    )
    server_url = models.CharField(
        verbose_name=_("Server URL"),
        max_length=255,
        help_text=_("The URL of the server to use for this communication profile"),
    )
    port = models.PositiveIntegerField(
        verbose_name=_("Port"),
        help_text=_("The port to use for this communication profile"),
    )
    username = models.CharField(
        verbose_name=_("Username"),
        max_length=100,
        help_text=_("The username to use for this communication profile"),
    )
    password = EncryptedCharField(
        verbose_name=_("Password"),
        max_length=100,
        help_text=_("The password to use for this communication profile"),
    )
    is_secure = models.BooleanField(
        verbose_name=_("Is Secure"),
        default=False,
        help_text=_("Whether or not this communication profile is secure"),
    )
    inbound_folder = models.CharField(
        verbose_name=_("Inbound Folder"),
        max_length=255,
        help_text=_("The folder to use for inbound EDI files"),
    )
    outbound_folder = models.CharField(
        verbose_name=_("Outbound Folder"),
        max_length=255,
        help_text=_("The folder to use for outbound EDI files"),
    )
    ack_folder = models.CharField(
        verbose_name=_("ACK Folder"),
        max_length=255,
        help_text=_("The folder to use for EDI acknowledgments"),
        blank=True,
    )
    retry_count = models.PositiveIntegerField(
        verbose_name=_("Retry Count"),
        help_text=_("The number of times to retry sending EDI files"),
        default=3,
        blank=True,
        null=True,
    )
    retry_interval = models.PositiveIntegerField(
        verbose_name=_("Retry Interval"),
        help_text=_("The number of seconds to wait between retries"),
        default=5,
        blank=True,
        null=True,
    )
    timeout = models.PositiveIntegerField(
        verbose_name=_("Timeout"),
        help_text=_("The timeout period for the connection in seconds"),
        default=120,
    )
    protocol_specific_settings = models.JSONField(
        verbose_name=_("Protocol Specific Settings"),
        blank=True,
        null=True,
        help_text=_("JSON dict for protocol-specific settings"),
    )
    ssl_certificate = models.FileField(
        verbose_name=_("SSL Certificate"),
        blank=True,
        null=True,
        upload_to="ssl_certificates/",
        help_text=_("SSL Certificate for secure connections"),
    )
    is_locked = models.BooleanField(
        verbose_name=_("Is Locked"),
        default=False,
        help_text=_(
            "Whether or not this profile is currently being used, preventing concurrent writes"
        ),
    )

    class Meta:
        """
        Metaclass for EDICommProfile model
        """

        verbose_name = _("EDI Communication Profile")
        verbose_name_plural = _("EDI Communication Profiles")
        db_table = "edi_comm_profile"

    def __str__(self) -> str:
        """EDI Communication Profile string representation

        Returns:
            str: String representation of the EDI Communication Profile
        """
        return textwrap.shorten(
            f"{self.name} - {self.protocol}",
            width=50,
            placeholder="...",
        )

    def get_absolute_url(self) -> str:
        """EDI Communication Profile Absolute URL

        Returns:
            str: Absolute URL of the EDI Communication Profile
        """
        return reverse("edi-comm-profile-details", kwargs={"pk": self.pk})

    def clean(self) -> None:
        """EDI Communication Profile clean method

        Returns:
            None: This function does not return anything.
        """
        super().clean()

        if self.protocol == "FTP" and self.is_secure:
            raise ValidationError(
                {"is_secure": _("FTP protocol cannot be secure. Please Try again.")},
                code="invalid",
            )
        if self.protocol == "HTTP" and self.is_secure:
            raise ValidationError(
                {"is_secure": _("HTTP protocol cannot be secure. Please Try again.")},
                code="invalid",
            )
        if self.protocol == "AS2" and not self.is_secure:
            raise ValidationError(
                {"is_secure": _("AS2 protocol must be secure. Please Try again.")},
                code="invalid",
            )
        if self.protocol == "SFTP" and not self.is_secure:
            raise ValidationError(
                {"is_secure": _("SFTP protocol must be secure. Please Try again.")},
                code="invalid",
            )
        if self.protocol == "AS2" and self.port != 443:
            raise ValidationError(
                {"port": _("AS2 protocol must use port 443. Please Try again.")},
                code="invalid",
            )
        if self.protocol == "SFTP" and self.port != 22:
            raise ValidationError(
                {"port": _("SFTP protocol must use port 22. Please Try again.")},
                code="invalid",
            )
        if self.protocol == "FTP" and self.port != 21:
            raise ValidationError(
                {"port": _("FTP protocol must use port 21. Please Try again.")},
                code="invalid",
            )
        if self.protocol == "HTTP" and self.port != 80:
            raise ValidationError(
                {"port": _("HTTP protocol must use port 80. Please Try again.")},
                code="invalid",
            )
        if self.protocol == "HTTP" and self.is_secure:
            raise ValidationError(
                {"is_secure": _("HTTP protocol cannot be secure. Please Try again.")},
                code="invalid",
            )
