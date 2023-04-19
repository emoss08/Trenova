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

from typing import Any, final, Union, Optional
import secrets
import string
from django.core import checks
from django.core.checks import CheckMessage, Error
from django.db import models
from django.db.models import CharField
from django.utils.translation import gettext_lazy as _
from django_extensions.db.models import TimeStampedModel
from pendulum import DateTime

from organization.models import Organization
import pendulum


def generate_random_string(length: int = 10) -> str:
    return "".join(
        secrets.choice(string.ascii_letters + string.digits) for _ in range(length)
    )


@final
class StatusChoices(models.TextChoices):
    """
    Status Choices for Order, Stop & Movement Statuses.
    """

    NEW = "N", _("New")
    IN_PROGRESS = "P", _("In Progress")
    COMPLETED = "C", _("Completed")
    HOLD = "H", _("Hold")
    BILLED = "B", _("Billed")
    VOIDED = "V", _("Voided")


@final
class RatingMethodChoices(models.TextChoices):
    """
    Rating Method choices for Order Model
    """

    FLAT = "F", _("Flat Fee")
    PER_MILE = "PM", _("Per Mile")
    PER_STOP = "PS", _("Per Stop")
    POUNDS = "PP", _("Per Pound")


@final
class StopChoices(models.TextChoices):
    """
    Status Choices for the Stop Model
    """

    PICKUP = "P", _("Pickup")
    SPLIT_PICKUP = "SP", _("Split Pickup")
    SPLIT_DROP = "SD", _("Split Drop Off")
    DELIVERY = "D", _("Delivery")
    DROP_OFF = "DO", _("Drop Off")


class GenericModel(TimeStampedModel):
    """
    Generic Model Fields
    """

    organization = models.ForeignKey(
        Organization,
        on_delete=models.CASCADE,
        related_name="%(class)ss",
        related_query_name="%(class)s",
        verbose_name=_("Organization"),
        help_text=_("Organization"),
    )

    class Meta:
        abstract = True

    def save(self, *args: Any, **kwargs: Any) -> None:
        """Save the model instance

        Args:
            *args (Any): Arguments
            **kwargs (Any): Keyword Arguments

        Returns:
            None
        """

        self.full_clean()
        super().save(*args, **kwargs)


class ChoiceField(CharField):
    """
    A CharField that lets you use Django choices and provides a nice
    representation in the admin.
    """

    description = _("Choice Field")

    def __init__(
        self, *args: Any, db_collation: Union[str, None] = None, **kwargs: Any
    ) -> None:
        super().__init__(*args, **kwargs)
        self.db_collation = db_collation
        if self.choices:
            self.max_length = max(len(choice[0]) for choice in self.choices)

    def check(self, **kwargs: Any) -> list[Union[CheckMessage, CheckMessage]]:
        """Check the field for errors.

        Check the fields for errors and return a list of Error objects.

        Args:
            **kwargs (Any): Keyword Arguments

        Returns:
            list[CheckMessage | CheckMessage]: List of Error objects
        """
        return [
            *super().check(**kwargs),
            *self._validate_choices_attribute(**kwargs),
        ]

    def _validate_choices_attribute(self, **kwargs: Any) -> Union[list[Error], list]:
        """Validate the choices attribute for the field.

        Validate the choices attribute is set in the field, if not return a list of
        Error objects.

        Args:
            **kwargs (Any): Keyword Arguments

        Returns:
            list{Error} | list: List of Error objects or an empty list
        """
        if self.choices is None:
            return [
                checks.Error(
                    "ChoiceField must define a `choice` attribute.",
                    hint="Add a `choice` attribute to the ChoiceField.",
                    obj=self,
                    id="fields.E120",
                )
            ]
        return []


class PendulumDateTimeField(models.DateTimeField):
    def to_python(self, value: Any) -> Optional[pendulum.DateTime]:
        if isinstance(value, pendulum.DateTime):
            return value
        return super().to_python(value)

    def from_db_value(
        self, value: Optional[Any], expression: Any, connection: Any
    ) -> Optional[pendulum.DateTime]:
        return pendulum.instance(value) if value is not None else value

    def get_prep_value(self, value: Optional[pendulum.DateTime]) -> Optional[Any]:
        if value is not None:
            value = pendulum.instance(value).naive()
        return super().get_prep_value(value)
