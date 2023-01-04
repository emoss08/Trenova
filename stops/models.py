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

from __future__ import annotations

import textwrap
import uuid
from typing import Any

from django.conf import settings
from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _

from dispatch.models import DispatchControl
from stops.services.create_service_incident import CreateServiceIncident
from stops.validation import StopValidation
from utils.models import ChoiceField, GenericModel, StatusChoices, StopChoices

User = settings.AUTH_USER_MODEL


class QualifierCode(GenericModel):
    """
    Stores Qualifier Code information that can be used in stop notes.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    code = models.CharField(
        _("Code"),
        max_length=255,
        unique=True,
        help_text=_("Code of the Qualifier Code"),
    )
    description = models.CharField(
        _("Description"),
        max_length=100,
        help_text=_("Description of the Qualifier Code"),
    )

    class Meta:
        """
        Qualifier Code Metaclass
        """

        verbose_name = _("Qualifier Code")
        verbose_name_plural = _("Qualifier Codes")
        ordering: list[str] = ["code"]

    def __str__(self) -> str:
        """Qualifier Code String Representation

        Returns:
            str: Code of the Qualifier
        """
        return textwrap.wrap(self.code, 50)[0]

    def get_absolute_url(self) -> str:
        """Qualifier Code Absolute URL

        Returns:
            str: Qualifier Code Absolute URL
        """
        return reverse("order:qualifier-code-detail", kwargs={"pk": self.pk})


class Stop(GenericModel):
    """
    Stores movement information related to a :model:`movements.Movement`.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    status = ChoiceField(
        choices=StatusChoices.choices,
        default=StatusChoices.NEW,
        help_text=_("The status of the stop."),
    )
    sequence = models.PositiveIntegerField(
        _("Sequence"),
        default=1,
        help_text=_("The sequence of the stop in the movement."),
    )
    movement = models.ForeignKey(
        "movements.Movement",
        on_delete=models.CASCADE,
        related_name="stops",
        related_query_name="stop",
        verbose_name=_("Movement"),
        help_text=_("The movement that the stop belongs to."),
    )
    location = models.ForeignKey(
        "location.Location",
        on_delete=models.PROTECT,
        related_name="stops",
        related_query_name="stop",
        verbose_name=_("Location"),
        help_text=_("The location of the stop."),
    )
    pieces = models.PositiveIntegerField(
        _("Pieces"),
        default=0,
        null=True,
        blank=True,
        help_text=_("Pieces"),
    )
    weight = models.PositiveIntegerField(
        _("Weight"),
        default=0,
        null=True,
        blank=True,
        help_text=_("Weight"),
    )
    address_line = models.CharField(
        _("Stop Address"),
        max_length=255,
        help_text=_("Stop Address"),
    )
    appointment_time = models.DateTimeField(
        _("Stop Appointment Time"),
        help_text=_("The time the equipment is expected to arrive at the stop."),
    )
    arrival_time = models.DateTimeField(
        _("Stop Arrival Time"),
        null=True,
        blank=True,
        help_text=_("The time the equipment actually arrived at the stop."),
    )
    departure_time = models.DateTimeField(
        _("Stop Departure Time"),
        null=True,
        blank=True,
        help_text=_("The time the equipment actually departed from the stop."),
    )
    stop_type = ChoiceField(
        choices=StopChoices.choices,
        help_text=_("The type of stop."),
    )

    class Meta:
        """
        Metaclass for the Stop model
        """

        verbose_name = _("Stop")
        verbose_name_plural = _("Stops")
        ordering: list[str] = ["movement", "sequence"]

    def __str__(self) -> str:
        """String representation of the Stop

        Returns:
            str: String representation of the Stop
        """
        return textwrap.shorten(
            f"{self.movement} - {self.sequence}({self.location})",
            width=50,
            placeholder="...",
        )

    def clean(self) -> None:
        """Stop clean Method

        Returns:
            None

        Raises:
            ValidationError: If the stop is not valid.

        """

        StopValidation(
            stop=self,
            stop_object=Stop,
        ).validate()

    def save(self, **kwargs: Any) -> None:
        """Save the stop object

        Args:
            **kwargs (Any): Keyword Arguments

        Returns:
            None
        """
        self.full_clean()

        if self.arrival_time and not self.departure_time:
            self.status = StatusChoices.IN_PROGRESS
        elif self.arrival_time and self.departure_time:
            self.status = StatusChoices.COMPLETED

        # TODO: THIS LOOKS WEIRD TO ME NOW. I MAY CHANGE THIS
        CreateServiceIncident(
            stop=self,
            dc_object=DispatchControl,
            si_object=ServiceIncident,
        ).create()

        super().save(**kwargs)

    def get_absolute_url(self) -> str:
        """Get the absolute url for the stop

        Returns:
            str: The absolute url for the stop
        """
        return reverse("stop:stops-detail", kwargs={"pk": self.pk})


class StopComment(GenericModel):
    """
    Stores comment  information related to a :model:`stop.Stop`.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    stop = models.ForeignKey(
        Stop,
        on_delete=models.CASCADE,
        related_name="comments",
        verbose_name=_("Stop"),
    )
    comment_type = models.ForeignKey(
        "dispatch.CommentType",
        on_delete=models.PROTECT,
        related_name="stop_comments",
        related_query_name="stop_comment",
        verbose_name=_("Comment Type"),
        help_text=_("The type of comment."),
    )
    qualifier_code = models.ForeignKey(
        QualifierCode,
        on_delete=models.PROTECT,
        related_name="stop_comments",
        related_query_name="stop_comment",
        verbose_name=_("Qualifier Code"),
        help_text=_("Qualifier code for the comment."),
    )
    comment = models.TextField(
        _("Comment"),
        help_text=_("Comment text."),
    )
    entered_by = models.ForeignKey(
        User,
        on_delete=models.PROTECT,
        related_name="stop_comments",
        related_query_name="stop_comment",
        verbose_name=_("Entered By"),
        help_text=_("User who entered the comment."),
    )

    class Meta:
        """
        Metaclass for Stop Comment class.
        """

        verbose_name = _("Stop Comment")
        verbose_name_plural = _("Stop Comments")

    def __str__(self) -> str:
        """String representation for stop comment

        Returns:
            str: return string representation for stop comment.
        """
        return textwrap.shorten(
            f"{self.stop}, {self.comment_type}({self.qualifier_code})",
            width=50,
            placeholder="...",
        )

    def get_absolute_url(self) -> str:
        """Get the absolute url for the StopComment

        Returns:
            str: Absolute url for the StopComment
        """
        return reverse("stop:stop-comment-detail", kwargs={"pk": self.pk})


class ServiceIncident(GenericModel):
    """
    Stores Service Incident information related to a
    :model:`order.Order` and :model:`stop.Stop`.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    movement = models.ForeignKey(
        "movements.Movement",
        on_delete=models.CASCADE,
        related_name="service_incidents",
        related_query_name="service_incident",
        verbose_name=_("Movement"),
    )
    stop = models.ForeignKey(
        Stop,
        on_delete=models.CASCADE,
        related_name="service_incidents",
        related_query_name="service_incident",
        verbose_name=_("Stop"),
    )
    delay_code = models.ForeignKey(
        "dispatch.DelayCode",
        on_delete=models.PROTECT,
        related_name="service_incidents",
        related_query_name="service_incident",
        verbose_name=_("Delay Code"),
        blank=True,
        null=True,
    )
    delay_reason = models.CharField(
        _("Delay Reason"),
        max_length=100,
        blank=True,
    )
    delay_time = models.DurationField(
        _("Delay Time"),
        null=True,
        blank=True,
    )

    class Meta:
        """
        ServiceIncident Metaclass
        """

        verbose_name = _("Service Incident")
        verbose_name_plural = _("Service Incidents")

    def __str__(self) -> str:
        """String representation of the ServiceIncident

        Returns:
            str: String representation of the ServiceIncident
        """
        return textwrap.shorten(
            f"{self.movement}, {self.stop}({self.delay_code})",
            width=50,
            placeholder="...",
        )

    def get_absolute_url(self) -> str:
        """Get the absolute url for the ServiceIncident

        Returns:
            str: Absolute url for the ServiceIncident
        """
        return reverse("stop:service-incident-detail", kwargs={"pk": self.pk})
