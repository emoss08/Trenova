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
from typing import Any, final

from django.conf import settings
from django.core.exceptions import ValidationError
from django.core.validators import MaxValueValidator, MinValueValidator
from django.db import models
from django.urls import reverse
from django.utils.functional import cached_property
from django.utils.timezone import datetime
from django.utils.translation import gettext_lazy as _
from localflavor.us.models import USStateField, USZipCodeField

from dispatch.validators.regulatory import validate_worker_regulatory_information
from organization.models import Depot
from utils.models import ChoiceField, GenericModel

User = settings.AUTH_USER_MODEL


class Worker(GenericModel):
    """
    Stores the equipment information that can be used later to
    assign an order to a movement.
    """

    @final
    class WorkerType(models.TextChoices):
        """
        The type of worker.
        """

        EMPLOYEE = "EMPLOYEE", _("Employee")
        CONTRACTOR = "CONTRACTOR", _("Contractor")

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    code = models.CharField(
        _("Code"),
        max_length=10,
        unique=True,
        blank=True,
        editable=False,
        help_text=_(
            "The code of the worker. This field is required and must be unique."
        ),
    )
    is_active = models.BooleanField(
        _("Active"),
        default=True,
        help_text=_(
            "Designates whether this worker should be treated as active. "
            "Unselect this instead of deleting workers."
        ),
    )
    worker_type = ChoiceField(
        _("Worker type"),
        choices=WorkerType.choices,
        default=WorkerType.EMPLOYEE,
        help_text=_("The type of worker."),
    )
    first_name = models.CharField(
        _("First name"),
        max_length=255,
        help_text=_("The first name of the worker."),
    )
    last_name = models.CharField(
        _("Last name"),
        max_length=255,
        help_text=_("The last name of the worker."),
    )
    address_line_1 = models.CharField(
        _("Address line 1"),
        max_length=255,
        help_text=_("The address line 1 of the worker."),
    )
    address_line_2 = models.CharField(
        _("Address line 2"),
        max_length=255,
        blank=True,
        help_text=_("The address line 2 of the worker."),
    )
    city = models.CharField(
        _("City"),
        max_length=255,
        help_text=_("The city of the worker."),
    )
    state = USStateField(
        _("State"),
        help_text=_("The state of the worker."),
    )
    zip_code = USZipCodeField(
        _("zip code"),
        help_text=_("The zip code of the worker."),
    )
    depot = models.ForeignKey(
        Depot,
        on_delete=models.CASCADE,
        null=True,
        blank=True,
        related_name="worker",
        related_query_name="workers",
        verbose_name=_("Depot"),
        help_text=_("The depot of the worker."),
    )
    manager = models.ForeignKey(
        User,
        on_delete=models.CASCADE,
        null=True,
        blank=True,
        related_name="worker",
        related_query_name="workers",
        verbose_name=_("Manager"),
        help_text=_("The manager of the worker."),
    )
    entered_by = models.ForeignKey(
        User,
        on_delete=models.CASCADE,
        null=True,
        blank=True,
        related_name="worker_entered",
        related_query_name="workers_entered",
        verbose_name=_("Entered by"),
        help_text=_("The user who entered the worker."),
    )

    class Meta:
        """
        Metaclass for Worker.
        """

        verbose_name = _("worker")
        verbose_name_plural = _("workers")
        ordering: list[str] = ["code"]

    def __str__(self) -> str:
        """Worker string representation

        Returns:
            str: Worker string representation
        """

        return textwrap.wrap(f"{self.first_name} {self.last_name}", 50)[0]

    def save(self, **kwargs) -> None:
        """Worker save method

        Returns:
            None
        """

        self.full_clean()
        super().save(**kwargs)

    def get_absolute_url(self) -> str:
        """Worker absolute url

        Returns:
            str: Worker absolute url
        """

        return reverse("worker:detail", kwargs={"pk": self.pk})

    @cached_property
    def get_full_name(self) -> str:
        """Worker full name

        Returns:
            str: Worker full name
        """

        return f"{self.first_name} {self.last_name}"

    @cached_property
    def get_full_address(self) -> str:
        """Worker full address

        Returns:
            str: Worker full address
        """

        return f"{self.address_line_1} {self.address_line_2} {self.city} {self.state} {self.zip_code}"


class WorkerProfile(GenericModel):
    """
    Stores the worker profile information related to the :model:`worker.Worker`.
    """

    @final
    class WorkerSexChoices(models.TextChoices):
        """
        Worker Sex/Gender Choices
        """

        MALE = "male", _("Male")
        FEMALE = "female", _("Female")
        NON_BINARY = "non-binary", _("Non-binary")
        OTHER = "other", _("Other")

    @final
    class EndorsementChoices(models.TextChoices):
        """
        Worker Endorsement Choices
        """

        NONE = "N", _("None")
        HAZMAT = "H", _("Hazmat")
        TANKER = "T", _("Tanker")
        X = "X", _("Tanker and Hazmat")

    worker = models.OneToOneField(
        Worker,
        on_delete=models.CASCADE,
        primary_key=True,
        related_name="profile",
        related_query_name="profiles",
        verbose_name=_("Worker"),
        help_text=_("The worker of the profile."),
    )
    race = models.CharField(
        _("Race/Ethnicity"),
        max_length=100,
        blank=True,
        help_text=_("Race/Ethnicity"),
    )
    sex = ChoiceField(
        _("Sex/Gender"),
        choices=WorkerSexChoices.choices,
        blank=True,
        help_text=_("Sex/Gender of the worker."),
    )
    date_of_birth = models.DateField(
        _("Date of Birth"),
        blank=True,
        null=True,
        help_text=_("Date of Birth of the worker."),
    )
    license_number = models.CharField(
        _("License Number"),
        max_length=20,
        help_text=_("License Number."),
        blank=True,
    )
    license_state = USStateField(
        _("License State"),
        help_text=_("License State."),
        null=True,
        blank=True,
    )
    license_expiration_date = models.DateField(
        _("License Expiration Date"),
        help_text=_("License Expiration Date."),
        null=True,
        blank=True,
    )
    endorsements = ChoiceField(
        _("Endorsements"),
        choices=EndorsementChoices.choices,
        default=EndorsementChoices.NONE,
        help_text=_("Endorsements."),
        blank=True,
    )
    hazmat_expiration_date = models.DateField(
        _("Hazmat Expiration Date"),
        blank=True,
        null=True,
        help_text=_("Hazmat Endorsement Expiration Date."),
    )
    hm_126_expiration_date = models.DateField(
        _("HM-126 Expiration Date"),
        blank=True,
        null=True,
        help_text=_("HM126GF Training Expiration Date."),
    )
    hire_date = models.DateField(
        _("Hire Date"),
        blank=True,
        null=True,
        help_text=_("Date of hire."),
        default=datetime.today,
    )
    termination_date = models.DateField(
        _("Termination Date"),
        blank=True,
        null=True,
        help_text=_("Date of termination."),
    )
    review_date = models.DateField(
        _("Review Date"),
        blank=True,
        null=True,
        help_text=_("Next Review Date."),
    )
    physical_due_date = models.DateField(
        _("Physical Due Date"),
        blank=True,
        null=True,
        help_text=_("Next Physical Due Date."),
    )
    mvr_due_date = models.DateField(
        _("MVR Due Date"),
        blank=True,
        null=True,
        help_text=_("Next MVR Due Date."),
    )
    medical_cert_date = models.DateField(
        _("Medical Cert Date"),
        blank=True,
        null=True,
        help_text=_("Medical Certification Expiration Date."),
    )

    class Meta:
        """
        Metaclass for WorkerProfile.
        """

        verbose_name = _("Worker profile")
        verbose_name_plural = _("Worker profiles")
        ordering: list[str] = ["worker"]

    def __str__(self) -> str:
        """Worker Profile string representation

        Returns:
            str: Worker Profile string representation
        """

        return textwrap.wrap(
            f"{self.worker.first_name} {self.worker.last_name} Profile", 50
        )[0]

    def clean(self) -> None:
        """Worker Profile clean method

        Returns:
            NOne

        Raises:
            ValidationError: If the worker profile is not valid.
        """

        super().clean()

        if (
            self.endorsements
            in [
                WorkerProfile.EndorsementChoices.X,
                WorkerProfile.EndorsementChoices.HAZMAT,
            ]
            and not self.hazmat_expiration_date
        ):
            raise ValidationError(
                {
                    "hazmat_expiration_date": _(
                        "Hazmat expiration date is required for this endorsement."
                    ),
                },
            )

        # validate worker regulatory information
        validate_worker_regulatory_information(self)

    def save(self, **kwargs) -> None:
        """Worker Profile save method

        Returns:
            None
        """

        self.full_clean()
        super().save(**kwargs)

    def get_absolute_url(self) -> str:
        """Worker Profile absolute url

        Returns:
            str: Worker Profile absolute url
        """

        return reverse("worker:profile-detail", kwargs={"pk": self.pk})


class WorkerContact(GenericModel):
    """
    Store contact information related to the associated :model:`worker.Worker`
    Model.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    worker = models.ForeignKey(
        Worker,
        on_delete=models.CASCADE,
        related_name="contacts",
        related_query_name="contacts",
        verbose_name=_("Worker"),
        help_text=_("Related Worker."),
    )
    name = models.CharField(
        _("name"),
        max_length=255,
        help_text=_("Name of the contact."),
    )
    phone = models.PositiveIntegerField(
        _("phone"),
        null=True,
        blank=True,
        validators=[MinValueValidator(1000000000), MaxValueValidator(9999999999)],
        help_text=_("Phone number in the format 1234567890"),
    )
    email = models.EmailField(
        _("Email Address"),
        blank=True,
        max_length=255,
        help_text=_("Email address of the contact."),
    )
    relationship = models.CharField(
        _("Relationship"),
        max_length=255,
        blank=True,
        help_text=_("Relationship to the worker."),
    )
    is_primary = models.BooleanField(
        _("Primary"),
        default=False,
        help_text=_("Is this the primary contact?"),
    )
    mobile_phone = models.PositiveIntegerField(
        _("mobile phone"),
        blank=True,
        null=True,
        help_text=_("Mobile phone number."),
    )

    class Meta:
        """
        Metaclass for WorkerContact
        """

        verbose_name = _("worker contact")
        verbose_name_plural = _("worker contacts")
        ordering: list[str] = ["worker"]

    def __str__(self) -> str:
        """Worker Contact string representation

        Returns:
            str: Worker Contact string representation
        """
        return textwrap.wrap(self.name, 50)[0]

    def save(self, **kwargs: Any):
        """Worker Contact save method

        Returns:
            None
        """

        self.full_clean()
        super().save(**kwargs)

    def get_absolute_url(self) -> str:
        """Worker Contact absolute url

        Returns:
            str: Worker Contact absolute url
        """

        return reverse("worker:contact-detail", kwargs={"pk": self.pk})


class WorkerComment(GenericModel):
    """
    Store worker comments related to the associated :model:`worker.Worker`
    Model.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    worker = models.ForeignKey(
        Worker,
        on_delete=models.CASCADE,
        related_name="comments",
        related_query_name="comments",
        verbose_name=_("worker"),
        help_text=_("Related worker."),
    )
    comment_type = models.ForeignKey(
        "dispatch.CommentType",
        on_delete=models.CASCADE,
        related_name="comments",
        related_query_name="comments",
        verbose_name=_("Comment Type"),
        help_text=_("Related comment type."),
    )
    comment = models.TextField(
        _("Comment"),
        help_text=_("Comment"),
    )
    entered_by = models.ForeignKey(
        User,
        on_delete=models.CASCADE,
        related_name="worker_comments",
        related_query_name="worker_comments",
        verbose_name=_("Entered By"),
        help_text=_("User who entered the comment."),
    )

    class Meta:
        """
        Metaclass for WorkerComment
        """

        verbose_name = _("worker comment")
        verbose_name_plural = _("worker comments")
        ordering: list[str] = ["worker"]

    def __str__(self) -> str:
        """Worker Comment string representation

        Returns:
            str: Worker Comment string representation
        """

        return textwrap.wrap(self.comment, 50)[0]

    def save(self, **kwargs: Any):
        """Worker Comment save method

        Returns:
            None
        """

        self.full_clean()
        super().save(**kwargs)

    def get_absolute_url(self) -> str:
        """Worker Comment absolute url

        Returns:
            str: Worker Comment absolute url
        """

        return reverse("worker:comment-detail", kwargs={"pk": self.pk})
