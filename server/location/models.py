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
from typing import Any

from django.db import models
from django.db.models.functions import Lower
from django.urls import reverse
from django.utils.functional import cached_property
from django.utils.translation import gettext_lazy as _
from localflavor.us.models import USStateField, USZipCodeField

from organization.models import Depot
from utils.models import GenericModel


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

    class Meta:
        """
        Metaclass for Location Category
        """

        verbose_name = _("Location Category")
        verbose_name_plural = _("Location Categories")
        ordering = ("name",)
        db_table = "location_category"
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
        return textwrap.wrap(self.name, 50)[0]

    def get_absolute_url(self) -> str:
        """Location Category absolute URL

        Returns:
            str: Location Category absolute URL
        """
        return reverse("location:locationcategory_detail", kwargs={"pk": self.pk})

    def update_location_category(self, **kwargs: Any) -> None:
        """
        Updates the Location Category with the given kwargs
        """
        for key, value in kwargs.items():
            setattr(self, key, value)
        self.save()


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
    code = models.CharField(
        _("Code"),
        max_length=100,
    )
    location_category = models.ForeignKey(
        LocationCategory,
        on_delete=models.PROTECT,
        verbose_name=_("Category"),
        related_name="location",
        related_query_name="locations",
        help_text=_("Location category"),
        null=True,
        blank=True,
    )
    depot = models.ForeignKey(
        Depot,
        on_delete=models.PROTECT,
        verbose_name=_("Depot"),
        related_name="location",
        related_query_name="locations",
        help_text=_("Location depot"),
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
    state = USStateField(
        _("State"),
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
        ordering = ("code",)
        db_table = "location"
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
        return textwrap.wrap(
            f"{self.code}: {self.address_line_1}, {self.city}, {self.state}", 50
        )[0]

    def get_absolute_url(self) -> str:
        """Location absolute URL

        Returns:
            str: Location absolute URL
        """
        return reverse("location:location_detail", kwargs={"pk": self.pk})

    @cached_property
    def get_address_combination(self) -> str:
        """Location address combination

        Returns:
            str: Location address combination
        """
        return f"{self.address_line_1}, {self.city}, {self.state} {self.zip_code}"

    def update_location(self, **kwargs: Any) -> None:
        """
        Updates the location with the given kwargs
        """
        for key, value in kwargs.items():
            setattr(self, key, value)
        self.save()


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
    )
    email = models.EmailField(
        _("Email"),
        max_length=255,
        help_text=_("Email of the contact."),
        blank=True,
    )
    phone = models.PositiveIntegerField(
        _("Phone"),
        help_text=_("Phone of the contact."),
        null=True,
        blank=True,
    )
    fax = models.PositiveIntegerField(
        _("Fax"),
        help_text=_("Fax of the contact."),
        null=True,
        blank=True,
    )

    class Meta:
        """
        Meta Class for LocationContact Model
        """

        verbose_name = _("Location Contact")
        verbose_name_plural = _("Location Contacts")
        ordering: tuple[str] = ("name",)
        indexes: list[models.Index] = [
            models.Index(fields=["name"]),
        ]
        db_table = "location_contact"

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

    def update_location_contact(self, **kwargs: Any) -> None:
        """Update LocationContact

        Args:
            **kwargs (Any): LocationContact attributes
        """
        for key, value in kwargs.items():
            setattr(self, key, value)
        self.save()


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
        ordering: tuple[str] = ("location",)
        db_table = "location_comment"

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

    def update_location_comment(self, **kwargs: Any) -> None:
        """Update LocationComment

        Args:
            **kwargs (Any): LocationComment attributes
        """
        for key, value in kwargs.items():
            setattr(self, key, value)
        self.save()
