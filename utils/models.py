# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2023 MONTA                                                                         -
#                                                                                                  -
#  This file is part of Monta.                                                                     -
#                                                                                                  -
#  Monta is free software: you can redistribute it and/or modify                                   -
#  it under the terms of the GNU General Public License as published by                            -
#  the Free Software Foundation, either version 3 of the License, or                               -
#  (at your option) any later version.                                                             -
#                                                                                                  -
#  Monta is distributed in the hope that it will be useful,                                        -
#  but WITHOUT ANY WARRANTY; without even the implied warranty of                                  -
#  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the                                   -
#  GNU General Public License for more details.                                                    -
#                                                                                                  -
#  You should have received a copy of the GNU General Public License                               -
#  along with Monta.  If not, see <https://www.gnu.org/licenses/>.                                 -
# --------------------------------------------------------------------------------------------------

import base64
from typing import Any, final
import cryptography.fernet

from django.core import checks
from django.core.checks import CheckMessage, Error
from django.db import models
from django.conf import settings
from django.core.exceptions import ImproperlyConfigured

from django.db.models import CharField
from django.utils.translation import gettext_lazy as _
from django_extensions.db.models import TimeStampedModel

from organization.models import Organization


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

    def save(self, **kwargs: Any) -> None:
        """Save the model instance

        Args:
            **kwargs (Any):

        Returns:
            None
        """

        self.full_clean()
        super().save(**kwargs)


class ChoiceField(CharField):
    """
    A CharField that lets you use Django choices and provides a nice
    representation in the admin.
    """

    description = _("Choice Field")

    def __init__(
        self, *args: Any, db_collation: str | None = None, **kwargs: Any
    ) -> None:
        super().__init__(*args, **kwargs)
        self.db_collation = db_collation
        if self.choices:
            self.max_length = max(len(choice[0]) for choice in self.choices)

    def check(self, **kwargs: Any) -> list[CheckMessage | CheckMessage]:
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

    def _validate_choices_attribute(self, **kwargs: Any) -> list[Error] | list:
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


def get_crypter() -> cryptography.fernet.MultiFernet:
    """
    Returns a MultiFernet object initialized with the encryption keys defined in the `FIELD_ENCRYPTION_KEY` setting.

    Raises:
        ImproperlyConfigured: If `FIELD_ENCRYPTION_KEY` is not defined or is defined incorrectly.

    Returns:
        cryptography.fernet.MultiFernet: A MultiFernet object initialized with the encryption keys.
    """
    configured_keys = getattr(settings, "FIELD_ENCRYPTION_KEY", None)

    if configured_keys is None:
        raise ImproperlyConfigured("FIELD_ENCRYPTION_KEY must be defined in settings")

    try:
        if isinstance(configured_keys, (tuple, list)):
            keys = [
                cryptography.fernet.Fernet(key.encode("utf-8"))
                for key in configured_keys
            ]
        else:
            keys = [
                cryptography.fernet.Fernet(configured_keys.encode("utf-8")),
            ]
    except Exception as e:
        raise ImproperlyConfigured(
            f"FIELD_ENCRYPTION_KEY defined incorrectly: {str(e)}"
        )

    if len(keys) == 0:
        raise ImproperlyConfigured("No keys defined in setting FIELD_ENCRYPTION_KEY")

    return cryptography.fernet.MultiFernet(keys)


CRYPTER: cryptography.fernet.MultiFernet = get_crypter()


class EncryptedCharField(models.CharField):
    """
    A custom Django CharField that encrypts and decrypts its value using the encryption keys defined in the
    `FIELD_ENCRYPTION_KEY` setting.

    Attributes:
        description (str): A description of the field for use in Django's admin interface.
    """

    description = "Encrypted CharField"

    def get_prep_value(self, value: str) -> str:
        """
        Encrypts the given value using the configured encryption keys.

        Args:
            value (str): The value to encrypt.

        Returns:
            str: The encrypted value, encoded as a base64-encoded string.
        """
        value: str = super().get_prep_value(value)
        if value is not None:
            value_bytes: bytes = value.encode("utf-8")
            encrypted_bytes: bytes = CRYPTER.encrypt(value_bytes)
            value: str = base64.b64encode(encrypted_bytes).decode("utf-8")  # type: ignore
        return value

    def from_db_value(self, value: str, expression: object, connection: object) -> str:
        """
        Decrypts the given value using the configured encryption keys.

        Args:
            value (str): The value to decrypt.
            expression (object): The query expression used to fetch the value.
            connection (object): The database connection used to fetch the value.

        Returns:
            str: The decrypted value.
        """
        if value is not None:
            value_bytes: bytes = base64.b64decode(value.encode("utf-8"))
            decrypted_bytes: bytes = CRYPTER.decrypt(value_bytes)
            value: str = decrypted_bytes.decode("utf-8")  # type: ignore
        return super().from_db_value(value, expression, connection)
