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

from __future__ import annotations

import textwrap
import uuid
from typing import Any

from django.conf import settings
from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _

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
        ordering = ["code"]
        db_table = "qualifier_code"
        constraints = [
            models.UniqueConstraint(
                fields=["code", "organization"],
                name="unique_qualifier_code_organization",
            )
        ]

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
        null=True,
        blank=True,
    )
    pieces = models.PositiveIntegerField(
        _("Pieces"),
        help_text=_("Total Piece Count of the Order"),
        default=0,
    )
    weight = models.DecimalField(
        _("Weight"),
        max_digits=10,
        decimal_places=2,
        help_text=_("Total Weight of the Order"),
        default=0,
    )
    address_line = models.CharField(
        _("Stop Address"),
        max_length=255,
        help_text=_("Stop Address"),
        blank=True,
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
        ordering = ["movement", "sequence"]
        db_table = "stop"

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
        super().clean()
        from stops.validation import StopValidation

        StopValidation(instance=self)

    def save(self, *args: Any, **kwargs: Any) -> None:
        """Save the stop object

        Args:
            **kwargs (Any): Keyword Arguments

        Returns:
            None
        """

        self.update_status_based_on_times()
        super().save(*args, **kwargs)

        # If the location code is entered and not the address_line then autofill address_line
        # with the location combination (address_line_1, address_line_2, city, state & zip_code)
        if self.location and not self.address_line:
            self.address_line = self.location.get_address_combination

    def update_status_based_on_times(self) -> None:
        if self.arrival_time is not None and self.departure_time is not None:
            self.status = StatusChoices.COMPLETED
        elif self.arrival_time is not None:
            self.status = StatusChoices.IN_PROGRESS
        else:
            self.status = StatusChoices.NEW

    def get_absolute_url(self) -> str:
        """Get the absolute url for the stop

        Returns:
            str: The absolute url for the stop
        """
        return reverse("stops-detail", kwargs={"pk": self.pk})


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
        db_table = "stop_comment"

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
        db_table = "service_incident"

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
