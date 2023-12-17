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

from colorfield.fields import ColorField
from django.core.validators import RegexValidator
from django.db import models
from django.db.models import Avg, DurationField, ExpressionWrapper, F
from django.db.models.functions import Lower
from django.urls import reverse
from django.utils.functional import cached_property
from django.utils.translation import gettext_lazy as _
from localflavor.us.models import USZipCodeField
from model_utils.models import TimeStampedModel

from organization.models import Depot
from utils.models import ChoiceField, GenericModel, PrimaryStatusChoices


class LocationCategory(GenericModel):
    """
    Stores location category information
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    name = models.CharField(
        _("Name"),
        max_length=100,
    )
    description = models.TextField(
        _("Description"),
        blank=True,
    )
    color = ColorField(
        _("Color"),
        blank=True,
        null=True,
        help_text=_("Color code for Location Category"),
    )

    class Meta:
        """
        Metaclass for Location Category
        """

        verbose_name = _("Location Category")
        verbose_name_plural = _("Location Categories")
        db_table = "location_category"
        db_table_comment = "Stores location category information."
        constraints = [
            models.UniqueConstraint(
                Lower("name"),
                "organization",
                name="unique_location_category_name_organization",
            )
        ]

    def __str__(self) -> str:
        """Location Category string representation

        Returns:
            str: Location Category name
        """
        return textwrap.shorten(
            f"{self.name}: {self.description}", width=50, placeholder="..."
        )

    def get_absolute_url(self) -> str:
        """Location Category absolute URL

        Returns:
            str: Location Category absolute URL
        """
        return reverse("location-category-detail", kwargs={"pk": self.pk})


class Location(GenericModel):
    """
    Stores location information for a related :model:`organization.Organization`.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    status = ChoiceField(
        _("Status"),
        choices=PrimaryStatusChoices.choices,
        help_text=_("Status of the Location."),
        default=PrimaryStatusChoices.ACTIVE,
    )
    code = models.CharField(  # TODO(WOLFRED): AUTO GENERATE THE CODE
        _("Code"),
        max_length=10,
        help_text=_("Unique Code for the Location."),
    )
    location_category = models.ForeignKey(
        LocationCategory,
        on_delete=models.RESTRICT,
        verbose_name=_("Location Category"),
        related_name="location",
        related_query_name="locations",
        help_text=_("The Location Category associated with the Location"),
        null=True,
        blank=True,
    )
    name = models.CharField(
        _("Name"),
        max_length=255,
        help_text=_("Location name"),
        db_index=True,
    )
    depot = models.ForeignKey(
        Depot,
        on_delete=models.PROTECT,
        verbose_name=_("Depot"),
        related_name="location",
        related_query_name="locations",
        help_text=_("The Depot associated with the Location"),
        null=True,
        blank=True,
    )
    description = models.TextField(
        _("Description"),
        blank=True,
        help_text=_("Location description"),
    )
    address_line_1 = models.CharField(
        _("Address Line 1"),
        max_length=255,
        help_text=_("Address Line 1"),
    )
    address_line_2 = models.CharField(
        _("Address Line 2"),
        max_length=255,
        help_text=_("Address Line 2"),
        blank=True,
    )
    city = models.CharField(
        _("City"),
        max_length=255,
        help_text=_("City"),
    )
    state = models.CharField(
        _("State"),
        max_length=5,
        help_text=_("State"),
    )
    zip_code = USZipCodeField(
        _("Zip Code"),
        help_text=_("Zip Code"),
    )
    longitude = models.FloatField(
        _("Longitude"),
        help_text=_("Longitude"),
        blank=True,
        null=True,
    )
    latitude = models.FloatField(
        _("Latitude"),
        help_text=_("Latitude"),
        blank=True,
        null=True,
    )
    place_id = models.CharField(
        _("Place ID"),
        max_length=255,
        help_text=_("Place ID"),
        blank=True,
    )
    is_geocoded = models.BooleanField(
        _("Is Geocoded"),
        default=False,
        help_text=_("Is the location geocoded?"),
    )

    class Meta:
        """
        Metaclass for Location Model
        """

        verbose_name = _("Location")
        verbose_name_plural = _("Locations")
        db_table = "location"
        db_table_comment = "Stores location information for a related organization."
        constraints = [
            models.UniqueConstraint(
                Lower("code"),
                "organization",
                name="unique_location_code_organization",
            )
        ]

    def __str__(self) -> str:
        """Location string representation

        Returns:
            str: Location ID
        """
        return textwrap.shorten(
            f"{self.code}: {self.name}", width=50, placeholder="..."
        )

    def get_absolute_url(self) -> str:
        """Location absolute URL

        Returns:
            str: Location absolute URL
        """
        return reverse("location_detail", kwargs={"pk": self.pk})

    @cached_property
    def get_address_combination(self) -> str:
        """Location address combination

        Returns:
            str: Location address combination
        """
        return f"{self.address_line_1}, {self.city}, {self.state} {self.zip_code}"

    def get_avg_wait_time(self) -> float:
        """
        Calculates the average wait time for this location.
        """
        return self.stops.aggregate(
            wait_time_avg=Avg(
                ExpressionWrapper(
                    F("departure_time") - F("arrival_time"),
                    output_field=DurationField(),
                )
            )
        )["wait_time_avg"]


class LocationContact(GenericModel):
    """
    Stores location contact information related to :model:`location.Location`.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    location = models.ForeignKey(
        Location,
        on_delete=models.PROTECT,
        verbose_name=_("Location"),
        related_name="location_contacts",
        related_query_name="location_contact",
        help_text=_("Location"),
    )
    name = models.CharField(
        _("Name"),
        max_length=255,
        help_text=_("Name of the contact."),
        db_index=True,
    )
    email = models.EmailField(
        _("Email"),
        max_length=255,
        help_text=_("Email of the contact."),
        blank=True,
    )
    phone = models.CharField(
        _("Phone Number"),
        max_length=15,
        blank=True,
        help_text=_("Phone Number of the contact."),
        validators=[
            RegexValidator(
                regex=r"^\(?([0-9]{3})\)?[-. ]?([0-9]{3})[-. ]?([0-9]{4})$",
                message=_("Phone number must be in the format (xxx) xxx-xxxx"),
            )
        ],
    )
    fax = models.CharField(
        _("Fax Number"),
        max_length=15,
        blank=True,
        help_text=_("The Fax Number of the contact"),
        validators=[
            RegexValidator(
                regex=r"^\(?([0-9]{3})\)?[-. ]?([0-9]{3})[-. ]?([0-9]{4})$",
                message=_("Fax number must be in the format (xxx) xxx-xxxx"),
            )
        ],
    )

    class Meta:
        """
        Meta Class for LocationContact Model
        """

        verbose_name = _("Location Contact")
        verbose_name_plural = _("Location Contacts")
        db_table = "location_contact"
        db_table_comment = "Stores location contact information related to location."

    def __str__(self) -> str:
        """LocationContact string representation

        Returns:
            str: LocationContact name
        """
        return textwrap.wrap(self.name, 50)[0]

    def get_absolute_url(self) -> str:
        """LocationContact absolute URL

        Returns:
            str: LocationContact absolute URL
        """
        return reverse("location-contacts-detail", kwargs={"pk": self.pk})


class LocationComment(GenericModel):
    """
    Stores location contact information related to :model:`location.Location`.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    location = models.ForeignKey(
        Location,
        on_delete=models.CASCADE,
        related_name="location_comments",
        related_query_name="location_comment",
        verbose_name=_("Location"),
    )
    comment_type = models.ForeignKey(
        "dispatch.CommentType",
        on_delete=models.PROTECT,
        related_name="location_comments",
        related_query_name="location_comment",
        verbose_name=_("Comment Type"),
    )
    comment = models.TextField(
        _("Comment"),
        help_text=_("Comment"),
    )
    entered_by = models.ForeignKey(
        "accounts.User",
        on_delete=models.PROTECT,
        related_name="location_comments",
        related_query_name="location_comment",
        verbose_name=_("Entered By"),
    )

    class Meta:
        """
        Meta Class for LocationComment Model
        """

        verbose_name = _("Location Comment")
        verbose_name_plural = _("Location Comments")
        ordering = ("location",)
        db_table = "location_comment"
        db_table_comment = "Stores location comment information related to location."

    def __str__(self) -> str:
        """LocationComment string representation

        Returns:
            str: LocationComment name
        """
        return textwrap.wrap(self.comment, 50)[0]

    def get_absolute_url(self) -> str:
        """LocationComment absolute URL

        Returns:
            str: LocationComment absolute URL
        """
        return reverse("location-comments-detail", kwargs={"pk": self.pk})


class States(TimeStampedModel):
    """
    Stores US states and their abbreviations
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
        help_text=_("The name of the state."),
    )
    abbreviation = models.CharField(
        _("Abbreviation"),
        help_text=_("The abbreviation of the state."),
        max_length=5,
    )
    country_name = models.CharField(
        _("Country Name"),
        max_length=255,
        help_text=_("The name of the country."),
    )
    country_iso3 = models.CharField(
        _("Country ISO3"),
        max_length=3,
        help_text=_("The ISO3 of the country."),
    )

    class Meta:
        """
        Metaclass for the State model
        """

        verbose_name = _("State")
        verbose_name_plural = _("States")
        ordering = ["name"]
        db_table = "states"
        constraints = [
            models.UniqueConstraint(
                Lower("name"),
                "abbreviation",
                name="unique_name_abbreviation_state",
            )
        ]

    def __str__(self) -> str:
        """State string representation.

        Returns:
            str: String representation of the state.
        """

        return textwrap.shorten(
            f"{self.name} - {self.abbreviation}",
            width=50,
            placeholder="...",
        )

    def get_absolute_url(self) -> str:
        """State absolute URL

        Returns:
            str: The absolute url for the state.
        """
        return reverse("state-detail", kwargs={"pk": self.pk})
