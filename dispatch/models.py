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

from django.conf import settings
from django.core.exceptions import ValidationError
from django.db import models
from django.db.models.aggregates import Max
from django.urls import reverse
from django.utils import timezone
from django.utils.translation import gettext_lazy as _
from djmoney.models.fields import MoneyField

from integration.models import IntegrationChoices
from organization.models import Organization
from utils.models import ChoiceField, GenericModel, RatingMethodChoices

User = settings.AUTH_USER_MODEL


class DispatchControl(GenericModel):
    """
    Class: DispatchControl

    Stores dispatch control information for a related :model:organization.Organization.

    The DispatchControl model stores dispatch control information for a related organization. It is used to store
        information such as the record
    service incident control, grace period, deadhead target, driver assign, trailer continuity, distance method,
        duplicate trailer check, regulatory check, prevention of orders on hold, and the generation of routes.

    Attributes:
        id (UUIDField): Primary key and default value is a randomly generated UUID. Editable and unique.
        organization (OneToOneField): ForeignKey to the related organization model with a CASCADE on delete. Has a
            verbose name of "Organization" and related names of "dispatch_control" and "dispatch_controls".
        record_service_incident (ChoiceField): ChoiceField that selects the record service incident control from the
            available choices (Never, Pickup, Delivery, Pickup and Delivery, All except shipper). Default value is
            "Never".
        grace_period (PositiveIntegerField): Positive integer field that stores the grace period for the service
            incident in minutes. Default value is 0.
        deadhead_target (DecimalField): Decimal field that stores the deadhead target mileage for the company. Default
            value is 0.00.
        driver_assign (BooleanField): Boolean field that enforces driver assign to orders for the company. Default
            value is True.
        trailer_continuity (BooleanField): Boolean field that enforces trailer continuity for the company. Default
            value is False.
        distance_method (ChoiceField): ChoiceField that selects the distance method from the available choices
            (Google, Monta). Default value is "Monta".
        dupe_trailer_check (BooleanField): Boolean field that enforces the duplicate trailer check for the company.
            Default value is False.
        regulatory_check (BooleanField): Boolean field that enforces the regulatory check for the company. Default
            value is False.
        prev_orders_on_hold (BooleanField): Boolean field that prevents dispatch of orders on hold for the company.
            Default value is False.
        generate_routes (BooleanField): Boolean field that indicates whether routes should be generated for the
            company. Default value is False.

    Methods:
        meta: Meta class for the DispatchControl model.
        __str__(self) -> str:
            Returns the string representation of the DispatchControl model.
        get_absolute_url(self) -> str:
            Returns the URL for this object's detail view.

    Examples:
    >>> dispatch_control = DispatchControl.objects.update(
        ...    record_service_incident=DispatchControl.ServiceIncidentControlChoices.NEVER,
        ...    grace_period=0,
        ...    deadhead_target=0.00,
        ...    driver_assign=True,
        ...    trailer_continuity=False,
        ...    distance_method=DispatchControl.DistanceMethodChoices.MONTA,
        ... )
    """

    @final
    class ServiceIncidentControlChoices(models.TextChoices):
        """
        Service Incident Control Choices
        """

        NEVER = "Never", _("Never")
        PICKUP = "Pickup", _("Pickup")
        DELIVERY = "Delivery", _("Delivery")
        PICKUP_DELIVERY = "Pickup and Delivery", _("Pickup and Delivery")
        ALL_EX_SHIPPER = "All except shipper", _("All except shipper")

    @final
    class DistanceMethodChoices(models.TextChoices):
        """
        Distance method choices for Order model
        """

        GOOGLE = "Google", _("Google")
        MONTA = "Monta", _("Monta")

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    organization = models.OneToOneField(
        Organization,
        on_delete=models.CASCADE,
        verbose_name=_("Organization"),
        related_name="dispatch_control",
        related_query_name="dispatch_controls",
    )
    record_service_incident = ChoiceField(
        _("Record Service Incident"),
        choices=ServiceIncidentControlChoices.choices,
        default=ServiceIncidentControlChoices.NEVER,
    )
    grace_period = models.PositiveIntegerField(
        _("Grace Period"),
        default=0,
        help_text=_("Grace period for the service incident in minutes."),
    )
    deadhead_target = models.DecimalField(
        _("Deadhead Target"),
        max_digits=5,
        decimal_places=2,
        default=0.00,
        help_text=_("Deadhead Mileage target for the company."),
    )
    driver_assign = models.BooleanField(
        _("Enforce Driver Assign"),
        default=True,
        help_text=_("Enforce driver assign to orders for the company."),
    )
    trailer_continuity = models.BooleanField(
        _("Enforce Trailer Continuity"),
        default=False,
        help_text=_("Enforce trailer continuity for the company."),
    )
    distance_method = ChoiceField(
        _("Distance Method"),
        choices=DistanceMethodChoices.choices,
        default=DistanceMethodChoices.MONTA,
        help_text=_("Distance method for the company."),
    )
    dupe_trailer_check = models.BooleanField(
        _("Enforce Duplicate Trailer Check"),
        default=False,
        help_text=_("Enforce duplicate trailer check for the company."),
    )
    regulatory_check = models.BooleanField(
        _("Enforce Regulatory Check"),
        default=False,
        help_text=_("Enforce regulatory check for the company."),
    )
    prev_orders_on_hold = models.BooleanField(
        _("Prevent Orders On Hold"),
        default=False,
        help_text=_("Prevent dispatch of orders on hold for the company."),
    )
    generate_routes = models.BooleanField(
        _("Generate Routes"),
        default=False,
        help_text=_("Generate routes for the company."),
    )
    driver_time_away_restriction = models.BooleanField(
        _("Enforce Driver Time Away"),
        default=True,
        help_text=_("Disallow assignments if the driver is on Time Away"),
    )
    tractor_worker_fleet_constraint = models.BooleanField(
        _("Enforce Tractor and Worker Fleet Continuity "),
        default=False,
        help_text=_("Enforce Worker and Tractor must be in the same fleet to be assigned to a dispatch."),
    )
    class Meta:
        """
        Metaclass for DispatchControl
        """

        verbose_name = _("Dispatch Control")
        verbose_name_plural = _("Dispatch Controls")
        ordering = ["organization"]
        db_table = "dispatch_control"

    def __str__(self) -> str:
        """Dispatch control string representation

        Returns:
            str: Dispatch control string representation
        """
        return textwrap.wrap(self.organization.name, 50)[0]

    def clean(self) -> None:
        """Dispatch control clean method

        Returns:
            None

        Raises:
            ValidationError: If the dispatch control is not valid.
        """
        super().clean()

        if self.distance_method == self.DistanceMethodChoices.GOOGLE and all(
            integration.integration_type != IntegrationChoices.GOOGLE_MAPS
            for integration in self.organization.integrations.all()
        ):
            raise ValidationError(
                {
                    "distance_method": _(
                        "Google Maps integration is not configured for the organization."
                        " Please configure the integration before selecting Google as "
                        "the distance method."
                    ),
                },
                code="invalid",
            )

    def get_absolute_url(self) -> str:
        """Dispatch control absolute URL

        Returns:
            str: Dispatch control absolute URL
        """
        return reverse("dispatch-control-detail", kwargs={"pk": self.pk})


class DelayCode(GenericModel):
    """
    Class: DelayCode

    A model to store delay codes for a service incident.

    The DelayCode model stores codes and descriptions for a delay that occurs during a service incident. The fault of
        the delay can be recorded as either the fault of the carrier or driver.

    Attributes:
        code (CharField): The primary key, unique, and four character code for the delay. Help text is "Delay code for
            the service incident."
        description (CharField): A 100-character description for the delay code. Help text is "Description for the
            delay code."
        f_carrier_or_driver (BooleanField): A boolean value indicating if the fault of the delay is the carrier or
            driver. Default value is False.
        Help text is "Fault is carrier or driver."

    Class Attributes:
        Meta (class): A metaclass for the DelayCode model with verbose name "Delay Code" and verbose name plural
            "Delay Codes".
        The ordering is based on the code attribute.

    Methods:
        str(self) -> str:
            Returns the string representation of the DelayCode instance, which is the first 50 characters of the
            code attribute.
        get_absolute_url(self) -> str:
            Returns the URL for the DelayCode instance's detail view.

    References:
        https://docs.djangoproject.com/en/4.2/ref/models/instances/#

    Examples:
        >>> delay_code = DelayCode.objects.get(code="0001")
        >>> delay_code.code
        "0001"
    """

    code = models.CharField(
        _("Delay Code"),
        max_length=4,
        primary_key=True,
        help_text=_("Delay code for the service incident."),
    )
    description = models.CharField(
        _("Description"),
        max_length=100,
        help_text=_("Description for the delay code."),
    )
    f_carrier_or_driver = models.BooleanField(
        _("Fault of Carrier or Driver"),
        default=False,
        help_text=_("Fault is carrier or driver."),
    )

    class Meta:
        """
        Metaclass for DelayCode
        """

        verbose_name = _("Delay Code")
        verbose_name_plural = _("Delay Codes")
        ordering: list[str] = ["code"]
        db_table = "delay_code"
        constraints = [
            models.UniqueConstraint(
                fields=["code", "organization"],
                name="unique_delay_code_organization",
            )
        ]

    def __str__(self) -> str:
        """Delay code string representation

        Returns:
            str: Delay code string representation
        """
        return textwrap.wrap(self.code, 50)[0]

    def get_absolute_url(self) -> str:
        """Delay code absolute URL

        Returns:
            str: Delay code absolute URL
        """
        return reverse("delay-codes-detail", kwargs={"pk": self.pk})


class FleetCode(GenericModel):
    """
    Class: FleetCode

    Model for storing fleet codes for service incidents.

    A FleetCode instance represents a code used to identify a fleet of vehicles for service incidents.
    This allows for tracking and reporting on specific fleets of vehicles, including their revenue goals,
    deadhead goals, and mileage goals.

    Attributes:
        code (CharField): Fleet code for the service incident.
            Has a max length of 4 characters, is the primary key and unique, with help text of "Fleet code for the
            service incident.".
        description (CharField): Description for the fleet code.
            Has a max length of 100 characters and help text of "Description for the fleet code.".
        is_active (BooleanField): Whether the fleet code is active.
            Has a default value of True and help text of "Is the fleet code active.".
        revenue_goal (DecimalField): Revenue goal for the fleet code.
            Has a maximum of 10 digits, 2 decimal places, a default value of 0.00, and help text of "Revenue goal for
            the fleet code.".
        deadhead_goal (DecimalField): Deadhead goal for the fleet code.
            Has a maximum of 10 digits, 2 decimal places, a default value of 0.00, and help text of "Deadhead goal for
            the fleet code.".
        mileage_goal (DecimalField): Mileage goal for the fleet code.
            Has a maximum of 10 digits, 2 decimal places, a default value of 0.00, and help text of "Mileage goal for
            the fleet code.".

    Methods:
        __str__(self) -> str:
            Returns a string representation of the fleet code, wrapped to a maximum of 50 characters.

        get_absolute_url(self) -> str:
            Returns the URL for this object's detail view.

    Examples:
        >>> fleet_code = FleetCode.objects.create(
        ...     code="FLEET",
        ...     description="Fleet Code",
        ...     revenue_goal=1000.00,
        ...     deadhead_goal=100.00,
        ...     mileage_goal=1000.00,
        ... )
        >>> fleet_code.code
        "FLEET"
        >>> fleet_code.description
        "Fleet Code"
    """

    code = models.CharField(
        _("Fleet Code"),
        max_length=4,
        primary_key=True,
        help_text=_("Fleet code for the service incident."),
    )
    description = models.CharField(
        _("Description"),
        max_length=100,
        help_text=_("Description for the fleet code."),
    )
    is_active = models.BooleanField(
        _("Is Active"),
        default=True,
        help_text=_("Is the fleet code active."),
    )
    revenue_goal = models.DecimalField(
        _("Revenue Goal"),
        max_digits=10,
        decimal_places=2,
        default=0.00,
        help_text=_("Revenue goal for the fleet code."),
    )
    deadhead_goal = models.DecimalField(
        _("Deadhead Goal"),
        max_digits=10,
        decimal_places=2,
        default=0.00,
        help_text=_("Deadhead goal for the fleet code."),
    )
    mileage_goal = models.DecimalField(
        _("Mileage Goal"),
        max_digits=10,
        decimal_places=2,
        default=0.00,
        help_text=_("Mileage goal for the fleet code."),
    )
    manager = models.ForeignKey(
        User,
        on_delete=models.CASCADE,
        related_name="fleet_code_manager",
        help_text=_("Manager for the fleet code."),
        null=True,
        blank=True,
    )

    class Meta:
        """
        Metaclass for FleetCode
        """

        verbose_name = _("Fleet Code")
        verbose_name_plural = _("Fleet Codes")
        ordering: list[str] = ["code"]
        db_table = "fleet_code"
        constraints = [
            models.UniqueConstraint(
                fields=["code", "organization"],
                name="unique_fleet_code_organization",
            )
        ]

    def __str__(self) -> str:
        """
        Return a string representation of the FleetCode instance.

        Returns:
            str: A string representation of the FleetCode instance, wrapped to a maximum of 4 characters.
        """
        return textwrap.wrap(self.code, 4)[0]

    def get_absolute_url(self) -> str:
        """
        Return the absolute URL for the FleetCode instance's detail view.

        Returns:
            str: The absolute URL for the FleetCode instance's detail view.
        """
        return reverse("fleet-codes-detail", kwargs={"pk": self.pk})


class CommentType(GenericModel):
    """
    Class: CommentType

    Model for storing different types of comments.

    A CommentType instance represents a type of comment that can be associated with a comment.
    This allows for categorization and grouping of comments based on their type.

    Attributes:
        id (UUIDField): Primary key and default value is a randomly generated UUID.
            Editable and unique.
        name (CharField): Name of the comment type.
            Has a max length of 255 characters and help text of "Comment type name".
        description (TextField): Description of the comment type.
            Has a max length of 255 characters and help text of "Comment type description".

    Methods:
        __str__(self) -> str:
            Returns a string representation of the comment type, wrapped to a maximum of 50 characters.

        get_absolute_url(self) -> str:
            Returns the URL for this object's detail view.

    Typical Usage Example:
        >>> comment_type = CommentType.objects.create(
        ...     name="Test Comment Type",
        ...     description="Test Comment Type Description",
        ... )
        >>> comment_type
        <CommentType: Test Comment Type>
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    name = models.CharField(
        _("Name"),
        max_length=255,
        help_text=_("Comment type name"),
    )
    description = models.TextField(
        _("Description"),
        max_length=255,
        help_text=_("Comment type description"),
    )

    class Meta:
        """
        Metaclass for the Rate model.

        The Meta class defines some options for the Rate model.
        """

        verbose_name = _("Comment Type")
        verbose_name_plural = _("Comment Types")
        ordering = ["organization"]
        db_table = "comment_type"

    def __str__(self) -> str:
        """
        Return a string representation of the CommentType instance.

        Returns:
            str: A string representation of the CommentType instance, wrapped to a maximum of 50 characters.
        """
        return textwrap.wrap(self.name, 50)[0]

    def get_absolute_url(self) -> str:
        """
        Return the absolute URL for the CommentType instance's detail view.

        Returns:
            str: The absolute URL for the CommentType instance's detail view.
        """
        return reverse("comment-types-detail", kwargs={"pk": self.pk})


class Rate(GenericModel):  # type:ignore
    """
    Class: Rate

    Django model representing a Rate. This model stores information about the rates for a related customer,
    commodity, order type, and equipment type.

    Attributes:
        id (UUIDField): Primary key and default value is a randomly generated UUID. Not editable and unique.
        rate_number (CharField): A unique identifier for the rate, with max length of 10 characters.
        customer (ForeignKey): A foreign key to the customer model, with a related name of "rates".
        effective_date (DateField): The date when the rate becomes effective.
        expiration_date (DateField): The date when the rate expires.
        commodity (ForeignKey): A foreign key to the commodity model, with a related name of "rates".
        order_type (ForeignKey): A foreign key to the order type model, with a related name of "rates".
        equipment_type (ForeignKey): A foreign key to the equipment type model, with a related name of "rates".
        comments (TextField): Comments about the rate.

    Methods:
        str(self) -> str:
            Returns the string representation of the Rate instance, which is the first 10 characters of the rate_number
            field.

        get_absolute_url(self) -> str:
            Returns the absolute URL for the detail view of this Rate instance.

        set_rate_number_before_create(self) -> None:
            Sets the rate_number field with the result of the generate_rate_number method before the instance is
            created.

        generate_rate_number() -> str:
            Returns a new rate number that has not been used before, generated by incrementing the count of all previous
            Rate instances.

        Class Meta:
            verbose_name (str): "Rate".
            verbose_name_plural (str): "Rates".
            ordering (list): Orders the Rate instances by the rate_number field.

    Typical Usage:
    >>> rate = Rate.objects.create(
        ...     customer=customer,
        ...     effective_date=timezone.now(),
        ...     expiration_date=timezone.now() + timedelta(days=30),
        ...     commodity=commodity,
        ...     order_type=order_type,
        ...     equipment_type=equipment_type,
        ... )
        >>> rate
        <Rate: R00001>
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    rate_number = models.CharField(
        _("Rate Number"),
        max_length=10,
        editable=False,
        help_text=_("Rate Number for Rate"),
    )
    customer = models.ForeignKey(
        "customer.Customer",
        on_delete=models.SET_NULL,
        verbose_name=_("Customer"),
        related_name="rates",
        null=True,
        blank=True,
        help_text=_("Customer for Rate"),
    )
    effective_date = models.DateField(
        _("Effective Date"),
        help_text=_("Effective Date for Rate"),
        default=timezone.now,
    )
    expiration_date = models.DateField(
        _("Expiration Date"),
        help_text=_("Expiration Date for Rate"),
        default=timezone.now,
    )
    commodity = models.ForeignKey(
        "commodities.Commodity",
        on_delete=models.SET_NULL,
        verbose_name=_("Commodity"),
        related_name="rates",
        null=True,
        blank=True,
        help_text=_("Commodity for Rate"),
    )
    order_type = models.ForeignKey(
        "order.OrderType",
        on_delete=models.SET_NULL,
        verbose_name=_("Order Type"),
        related_name="rates",
        null=True,
        blank=True,
    )
    equipment_type = models.ForeignKey(
        "equipment.EquipmentType",
        on_delete=models.SET_NULL,
        verbose_name=_("Equipment Type"),
        related_name="rates",
        null=True,
        blank=True,
    )
    comments = models.TextField(
        _("Comments"),
        max_length=255,
        blank=True,
        help_text=_("Comments for Rate"),
    )

    class Meta:
        """
        Metaclass for the Rate model.

        The Meta class defines some options for the Rate model.
        """

        verbose_name = _("Rate")
        verbose_name_plural = _("Rates")
        ordering = ["rate_number"]
        db_table = "rate"
        constraints = [
            models.UniqueConstraint(
                fields=["rate_number", "organization"],
                name="unique_rate_number_organization",
            )
        ]

    def __str__(self) -> str:
        """
        Return the string representation of a Rate instance.

        Returns:
            str: The first 10 characters of the rate_number field.
        """
        return textwrap.wrap(self.rate_number, 10)[0]

    def get_absolute_url(self) -> str:
        """
        Return the absolute URL for the detail view of a Rate instance.

        Returns:
            str: The absolute URL for the detail view of the Rate instance.
        """
        return reverse("rates-detail", kwargs={"pk": self.pk})

    def clean(self) -> None:
        """
        Clean the Rate instance.

        Returns:
            None: None
        """
        if self.expiration_date < self.effective_date:
            raise ValidationError(
                {
                    "expiration_date": _(
                        "Expiration Date must be after Effective Date. Please correct and try again."
                    )
                }
            )

    @staticmethod
    def generate_rate_number() -> str:
        """
        Generate a unique rate number for a Rate instance.

        This method generates a unique rate number by finding the highest rate number and
        incrementing it by 1.

        Returns:
            str: A unique rate number for a Rate instance, formatted as "R{count:05d}".
        """

        if rate_number := Rate.objects.aggregate(Max("rate_number"))[
            "rate_number__max"
        ]:
            count = int(rate_number[1:]) + 1
        else:
            count = 1

        return f"R{count:05d}"


class RateTable(GenericModel):
    """
    Class: RateTable

    The `RateTable` model represents a table that stores the rate details for a specific origin and destination location
    and their respective rate method, rate amount and distance override.

    Attributes:
        id (UUIDField): A unique identifier for the rate table instance.
        rate (ForeignKey): A foreign key to the `Rate` model, representing the rate for the rate table.
        description (CharField): A description for the rate table.
        origin_location (ForeignKey): A foreign key to the `Location` model, representing the origin location for the
            rate table.
        destination_location (ForeignKey): A foreign key to the `Location` model, representing the destination location
            for the rate table.
        rate_method (ChoiceField): The rate method for the rate table, chosen from the `RatingMethodChoices` choices.
        rate_amount (PositiveIntegerField): The rate amount for the rate table.
        distance_override (PositiveIntegerField): The distance override for the rate table.

    Methods:
        meta (Meta): The Meta class defines some options for the RateTable model.
        __str__ (str): Return the string representation of a RateTable instance.
        get_absolute_url (str): Return the absolute URL for the detail view of a RateTable instance.

    Typical Usage:
        >>> rate_table = RateTable.objects.create(
        ...     rate=rate,
        ...     description="Rate Table 1",
        ...     origin_location=origin_location,
        ...     destination_location=destination_location,
        ...     rate_method=RatingMethodChoices.FLAT,
        ...     rate_amount=100,
        ...     distance_override=100,
        ... )
        >>> rate_table
        <RateTable: Rate Table 1>
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    rate = models.ForeignKey(
        Rate,
        on_delete=models.PROTECT,
        related_name="rate_tables",
        verbose_name=_("Rate"),
        help_text=_("Rate for Rate Table"),
    )
    description = models.CharField(
        _("Description"),
        max_length=100,
        help_text=_("Description for Rate Table"),
        blank=True,
    )
    origin_location = models.ForeignKey(
        "location.Location",
        on_delete=models.PROTECT,
        related_name="origin_rate_tables",
        verbose_name=_("Origin Location"),
        help_text=_("Origin Location for Rate Table"),
        blank=True,
        null=True,
    )
    destination_location = models.ForeignKey(
        "location.Location",
        on_delete=models.PROTECT,
        related_name="destination_rate_tables",
        verbose_name=_("Destination Location"),
        help_text=_("Destination Location for Rate Table"),
        blank=True,
        null=True,
    )
    rate_method = ChoiceField(
        _("Rate Method"),
        choices=RatingMethodChoices.choices,
        default=RatingMethodChoices.FLAT,
        help_text=_("Rate Method for Rate Table"),
    )
    rate_amount = models.PositiveIntegerField(
        _("Rate"),
        help_text=_("Rate for Rate Table"),
    )
    distance_override = models.PositiveIntegerField(
        _("Distance Override"),
        help_text=_("Distance Override for Rate Table"),
        blank=True,
        null=True,
    )

    class Meta:
        """
        Metaclass for the RateTable model.
        """

        verbose_name = _("Rate Table")
        verbose_name_plural = _("Rate Tables")
        ordering = ["rate", "origin_location", "destination_location"]
        db_table = "rate_table"

    def __str__(self) -> str:
        """
        Return the string representation of a RateTable instance.

        Returns:
            str: The description field of the RateTable instance.
        """
        return self.description

    def get_absolute_url(self) -> str:
        """
        Return the absolute URL for the detail view of a RateTable instance.

        Returns:
            str: The absolute URL for the detail view of the RateTable instance.
        """
        return reverse("rate-tables-detail", kwargs={"pk": self.pk})


class RateBillingTable(GenericModel):  # type:ignore
    """
    Class: RateBillingTable

    Django model representing a RateBillingTable. This model stores Billing Table information for a
    related :model:`rates.Rate`.

    Attributes:
        id (UUIDField): The primary key for the rate billing table instance.
        rate (ForeignKey): The rate associated with the rate billing table instance.
        charge_code (ForeignKey): The charge code associated with the rate billing table instance.
        description (CharField): The description for the rate billing table instance.
        units (PositiveIntegerField): The number of units for the rate billing table instance.
        charge_amount (MoneyField): The charge amount for the rate billing table instance.
        sub_total (MoneyField): The sub_total for the rate billing table instance.

    Methods:
        meta: Return the meta options for the RateBillingTable model.
        get_absolute_url: Return the absolute URL for the detail view of a RateBillingTable instance.
        __str__: Return the string representation of a RateBillingTable instance.

    Examples:
        >>> rate_billing_table = RateBillingTable.objects.create(
        ...        rate=rate,
        ...        charge_code=charge_code,
        ...        description="Rate Billing Table 1",
        ...        units=100,
        ...        charge_amount=100,
        ...    )
        >>> rate_billing_table
        <RateBillingTable: Rate Billing Table 1>
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    rate = models.ForeignKey(
        Rate,
        on_delete=models.PROTECT,
        related_name="rate_billing_tables",
        verbose_name=_("Rate"),
        help_text=_("Rate for Rate Billing Table"),
    )
    charge_code = models.ForeignKey(
        "billing.AccessorialCharge",
        on_delete=models.PROTECT,
        related_name="rate_billing_tables",
        verbose_name=_("Charge Code"),
        help_text=_("Charge Code for Rate Billing Table"),
    )
    description = models.CharField(
        _("Description"),
        max_length=100,
        help_text=_("Description for Rate Billing Table"),
        blank=True,
    )
    units = models.PositiveIntegerField(
        _("Units"),
        help_text=_("Units for Rate Billing Table"),
    )
    charge_amount = MoneyField(
        _("Charge Amount"),
        max_digits=19,
        decimal_places=4,
        default=0,
        default_currency="USD",
        help_text=_("Charge Amount for Rate Billing Table"),
    )
    sub_total = MoneyField(
        _("Total"),
        max_digits=19,
        decimal_places=4,
        default=0,
        default_currency="USD",
        help_text=_("Total for Rate Billing Table"),
    )

    class Meta:
        """
        Metaclass for the RateBillingTable model.
        """

        verbose_name = _("Rate Billing Table")
        verbose_name_plural = _("Rate Billing Tables")
        ordering = ["rate", "charge_code"]
        db_table = "rate_billing_table"

    def __str__(self) -> str:
        """
        Return the string representation of a RateBillingTable instance.

        The string representation of a RateBillingTable instance is the value of its description field.

        Returns:
            str: The description field of the RateBillingTable instance.
        """
        return self.description

    def get_absolute_url(self) -> str:
        """
        Return the absolute URL for the detail view of a RateBillingTable instance.

        Returns:
            str: The absolute URL for the detail view of the RateBillingTable instance.
        """
        return reverse("rate-billing-tables-detail", kwargs={"pk": self.pk})
