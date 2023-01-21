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
from typing import Any

from django.conf import settings
from django.db import models
from django.urls import reverse
from django.utils.functional import cached_property
from django.utils.translation import gettext_lazy as _
from localflavor.us.models import USStateField, USZipCodeField

from organization.models import Depot
from utils.models import GenericModel

User = settings.AUTH_USER_MODEL


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
        unique=True,
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
        ordering: tuple[str, ...] = ("name",)

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
        unique=True,
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

    def __str__(self) -> str:
        """LocationContact string representation

        Returns:
            str: LocationContact name
        """
        return textwrap.wrap(self.name, 50)[0]

    def update_location_contact(self, **kwargs: Any) -> None:
        """Update LocationContact

        Args:
            **kwargs (Any): LocationContact attributes
        """
        for key, value in kwargs.items():
            setattr(self, key, value)
        self.save()

    def get_absolute_url(self) -> str:
        """LocationContact absolute URL

        Returns:
            str: LocationContact absolute URL
        """
        return reverse("location:locationcontact_detail", kwargs={"pk": self.pk})


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
        User,
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

    def __str__(self) -> str:
        """LocationComment string representation

        Returns:
            str: LocationComment name
        """
        return textwrap.wrap(self.comment, 50)[0]

    def update_location_comment(self, **kwargs: Any) -> None:
        """Update LocationComment

        Args:
            **kwargs (Any): LocationComment attributes
        """
        for key, value in kwargs.items():
            setattr(self, key, value)
        self.save()

    def get_absolute_url(self) -> str:
        """LocationComment absolute URL

        Returns:
            str: LocationComment absolute URL
        """
        return reverse("location:locationcomment_detail", kwargs={"pk": self.pk})
