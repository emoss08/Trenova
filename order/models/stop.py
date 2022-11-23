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

# THIS FILE IS A FUCKING NIGHTMARE BUT PYTHON & FUCKING DJANGO!

from __future__ import annotations

from typing import Any

from django.conf import settings
from django.core.exceptions import ValidationError
from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _

from order.models.choices import StatusChoices, StopChoices
from order.models.service_incident import ServiceIncident
from utils.models import ChoiceField, GenericModel

User = settings.AUTH_USER_MODEL


class Stop(GenericModel):
    """
    Stores movement information related to a :model:`order.Movement`.
    """

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
        "order.Movement",
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
        return f"{self.movement} - {self.sequence} - {self.location}"

    def clean(self) -> None:
        """Stop clean Method

        Returns:
            None

        Raises:
            ValidationError: If the stop is not valid.

        """
        if self.pk:
            if self.status == StatusChoices.NEW:
                old_status = Stop.objects.get(pk=self.pk).status

                if old_status in [StatusChoices.IN_PROGRESS, StatusChoices.COMPLETED]:
                    raise ValidationError(
                        {
                            "status": ValidationError(
                                _(
                                    "Cannot change status to new if the status was"
                                    " previously in progress or completed."
                                ),
                                code="invalid",
                            )
                        }
                    )

            if self.sequence > 1:
                previous_stop = self.movement.stops.filter(
                    sequence=self.sequence - 1
                ).first()

                if (
                        previous_stop
                        and self.appointment_time < previous_stop.appointment_time
                ):
                    raise ValidationError(
                        {
                            "appointment_time": ValidationError(
                                _("Appointment time must be after previous stop."),
                                code="invalid",
                            )
                        }
                    )

                if previous_stop and previous_stop.status != StatusChoices.COMPLETED:
                    if self.status in [
                        StatusChoices.IN_PROGRESS,
                        StatusChoices.COMPLETED,
                    ]:
                        raise ValidationError(
                            {
                                "status": ValidationError(
                                    _(
                                        "Cannot change status to in progress or completed if"
                                        " previous stop is not completed."
                                    ),
                                    code="invalid",
                                )
                            }
                        )

                if self.sequence < self.movement.stops.count():
                    next_stop = self.movement.stops.filter(
                        sequence__exact=self.sequence + 1
                    ).first()

                    if next_stop and self.appointment_time > next_stop.appointment_time:
                        raise ValidationError(
                            {
                                "appointment_time": ValidationError(
                                    _("Appointment time must be before next stop."),
                                    code="invalid",
                                )
                            }
                        )

                    # If the next stop is in progress or completed, the current stop cannot be available
                    if (
                            next_stop
                            and self.status != StatusChoices.COMPLETED
                            and next_stop.status
                            in [
                        StatusChoices.COMPLETED,
                        StatusChoices.IN_PROGRESS,
                    ]
                    ):
                        raise ValidationError(
                            {
                                "status": ValidationError(
                                    _(
                                        "Previous stop must be completed before this stop can"
                                        " be in progress or completed."
                                    ),
                                    code="invalid",
                                )
                            }
                        )

                    if not self.movement.primary_worker and not self.movement.equipment:
                        if self.status in [
                            StatusChoices.IN_PROGRESS,
                            StatusChoices.COMPLETED,
                        ]:
                            raise ValidationError(
                                {
                                    "status": ValidationError(
                                        _(
                                            "Cannot change status to in progress or completed if"
                                            " there is no equipment or primary worker."
                                        ),
                                        code="invalid",
                                    )
                                }
                            )

                        if self.arrival_time or self.departure_time:
                            raise ValidationError(
                                {
                                    "arrival_time": ValidationError(
                                        _(
                                            "Must assign worker or equipment to movement before"
                                            " setting arrival or departure time."
                                        ),
                                        code="invalid",
                                    )
                                }
                            )

                        if self.departure_time and not self.arrival_time:
                            raise ValidationError(
                                {
                                    "departure_time": ValidationError(
                                        _(
                                            "Must set arrival time before setting departure time."
                                        ),
                                        code="invalid",
                                    )
                                }
                            )

                        if self.departure_time < self.arrival_time:
                            raise ValidationError(
                                {
                                    "departure_time": ValidationError(
                                        _("Departure time must be after arrival time."),
                                        code="invalid",
                                    )
                                }
                            )

    def save(self, *args: Any, **kwargs: Any) -> None:
        """Stop save method

        Args:
            *args (Any): Arguments
            **kwargs (Any): Keyword Arguments

        Returns:
            None
        """

        self.full_clean()

        # If the status changes to in progress, change the movement status associated to this stop to in progress.
        if self.status == StatusChoices.IN_PROGRESS:
            self.movement.status = StatusChoices.IN_PROGRESS
            self.movement.save()

        # if the last stop is completed, change the movement status to complete.
        if self.status == StatusChoices.COMPLETED:
            if (
                    self.movement.stops.filter(status=StatusChoices.COMPLETED).count()
                    == self.movement.stops.count()
            ):
                self.movement.status = StatusChoices.COMPLETED
                self.movement.save()

        # If the arrival time is set, change the status to in progress.
        if self.arrival_time:
            self.status = StatusChoices.IN_PROGRESS

            # If the arrival time of the stop is after the appointment time, create a service incident.
            if self.arrival_time > self.appointment_time:
                ServiceIncident.objects.create(
                    organization=self.movement.order.organization,
                    movement=self.movement,
                    stop=self,
                    delay_time=self.arrival_time - self.appointment_time,
                )

        # If the stop arrival and departure time are set, change the status to complete.
        if self.arrival_time and self.departure_time:
            self.status = StatusChoices.COMPLETED

        super().save(*args, **kwargs)

    def get_absolute_url(self) -> str:
        """Get the absolute url for the Stop

        Returns:
            str: Absolute url for the Stop
        """
        return reverse("stop-detail", kwargs={"pk": self.pk})
