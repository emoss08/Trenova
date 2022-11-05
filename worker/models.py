# -*- coding: utf-8 -*-
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
from typing import Any, final

from django.conf import settings
from django.core.exceptions import ValidationError
from django.core.validators import MaxValueValidator, MinValueValidator
from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _
from localflavor.us.models import USStateField  # type: ignore

from control_file.models import CommentType
from core.models import GenericModel
from dispatch.models import DispatchControl
from dispatch.validators.regulatory import \
    validate_worker_regulatory_information
from organization.models import Depot

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

    code = models.CharField(
        _("code"),
        max_length=10,
        unique=True,
        null=True,
        blank=True,
        editable=False,
        help_text=_(
            "The code of the worker. This field is required and must be unique."
        ),
    )
    is_active = models.BooleanField(
        _("active"),
        default=True,
        help_text=_(
            "Designates whether this worker should be treated as active. "
            "Unselect this instead of deleting workers."
        ),
    )
    worker_type = models.CharField(
        _("worker type"),
        max_length=10,
        choices=WorkerType.choices,
        default=WorkerType.EMPLOYEE,
        help_text=_("The type of worker."),
    )
    first_name = models.CharField(
        _("first name"),
        max_length=255,
        help_text=_("The first name of the worker."),
    )
    last_name = models.CharField(
        _("last name"),
        max_length=255,
        help_text=_("The last name of the worker."),
    )
    address_line_1 = models.CharField(
        _("address line 1"),
        max_length=255,
        help_text=_("The address line 1 of the worker."),
    )
    address_line_2 = models.CharField(
        _("address line 2"),
        max_length=255,
        blank=True,
        null=True,
        help_text=_("The address line 2 of the worker."),
    )
    city = models.CharField(
        _("city"),
        max_length=255,
        help_text=_("The city of the worker."),
    )
    state = USStateField(
        _("state"),
        help_text=_("The state of the worker."),
    )
    zip_code = models.PositiveIntegerField(
        _("zip code"),
        validators=[
            MinValueValidator(10000),
            MaxValueValidator(99999),
        ],
        help_text=_("The zip code of the worker."),
    )
    depot = models.ForeignKey(
        Depot,
        on_delete=models.CASCADE,
        null=True,
        blank=True,
        related_name="worker",
        related_query_name="workers",
        verbose_name=_("depot"),
        help_text=_("The depot of the worker."),
    )
    manager = models.ForeignKey(
        User,
        on_delete=models.CASCADE,
        null=True,
        blank=True,
        related_name="worker",
        related_query_name="workers",
        verbose_name=_("manager"),
        help_text=_("The manager of the worker."),
    )

    class Meta:
        verbose_name = _("worker")
        verbose_name_plural = _("workers")
        ordering: list[str] = ["code"]

    def __str__(self) -> str:
        """Worker string representation

        Returns:
            str: Worker string representation
        """
        return textwrap.wrap(f"{self.first_name} {self.last_name}", 50)[0]

    def generate_code(self) -> str:
        """Generate a unique code for the worker

        Returns:
            str: Worker code
        """
        first_name: str = self.first_name[0]
        last_name: str = self.last_name[:9]

        code: str = f"{first_name}{last_name}".upper()
        new_code: str = f"{code}{Worker.objects.count()}"

        return code if not Worker.objects.filter(code=code).exists() else new_code

    def save(self, **kwargs: Any) -> None:
        """Save the worker

        Args:
            **kwargs (Any): Keyword arguments

        Returns:
            None
        """
        if not self.code:
            self.code = self.generate_code()
        super().save(**kwargs)

    def get_absolute_url(self) -> str:
        """Worker absolute url

        Returns:
            str: Worker absolute url
        """
        return reverse("worker:detail", kwargs={"pk": self.pk})

    @property
    def get_full_name(self) -> str:
        """Worker full name

        Returns:
            str: Worker full name
        """
        return f"{self.first_name} {self.last_name}"

    @property
    def get_full_address(self) -> str:
        """Worker full address

        Returns:
            str: Worker full address
        """
        return f"{self.address_line_1} {self.address_line_2} {self.city} {self.state} {self.zip_code}"


class WorkerProfile(GenericModel):
    """
    Stores the worker profile information related to the `Worker` model.
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

        NONE = "n", _("None")
        HAZMAT = "h", _("Hazmat")
        TANKER = "t", _("Tanker")
        X = "x", _("Tanker and Hazmat")

    worker = models.OneToOneField(
        Worker,
        on_delete=models.CASCADE,
        primary_key=True,
        related_name="profile",
        related_query_name="profiles",
        verbose_name=_("worker"),
        help_text=_("The worker of the profile."),
    )
    race = models.CharField(
        _("Race/Ethnicity"),
        max_length=100,
        blank=True,
        null=True,
        help_text=_("Race/Ethnicity"),
    )
    sex = models.CharField(
        _("Sex/Gender"),
        max_length=11,
        choices=WorkerSexChoices.choices,
        blank=True,
        null=True,
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
        null=True,
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
    endorsements = models.CharField(
        _("Endorsements"),
        max_length=1,
        choices=EndorsementChoices.choices,
        default=EndorsementChoices.NONE,
        help_text=_("Endorsements."),
        null=True,
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
        verbose_name = _("worker profile")
        verbose_name_plural = _("worker profiles")
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
        # if (
        #         self.endorsements
        #         in [
        #     WorkerProfile.EndorsementChoices.X,
        #     WorkerProfile.EndorsementChoices.HAZMAT,
        # ]
        #         and not self.hazmat_expiration_date
        # ):
        #     raise ValidationError(
        #         ValidationError(
        #             {
        #                 "hazmat_expiration_date": _(
        #                     "Hazmat expiration date is required for this endorsement."
        #                 ),
        #             },
        #             code="invalid",
        #         )
        #     )
        # existing_drivers = [
        #     driver
        #     for driver in WorkerProfile.objects.all()
        #     if driver.license_number == self.license_number
        # ]
        # if existing_drivers:
        #     raise ValidationError(
        #         ValidationError(
        #             {
        #                 "license_number": _(
        #                     f"License number already exists for {existing_drivers[0].worker.code}."
        #                 ),
        #             },
        #             code="invalid",
        #         )
        #     )
        validate_worker_regulatory_information(self)

    def get_absolute_url(self) -> str:
        """Worker Profile absolute url

        Returns:
            str: Worker Profile absolute url
        """
        return reverse("worker:profile-detail", kwargs={"pk": self.pk})


class WorkerContact(GenericModel):
    """
    Store contact information related to the associated `Worker`
    Model.
    """

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
        null=True,
        max_length=255,
        help_text=_("Email address of the contact."),
    )
    relationship = models.CharField(
        _("Relationship"),
        max_length=255,
        blank=True,
        null=True,
        help_text=_("Relationship to the worker."),
    )
    primary = models.BooleanField(
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
        verbose_name = _("worker contact")
        verbose_name_plural = _("worker contacts")
        ordering: list[str] = ["worker"]

    def __str__(self) -> str:
        """Worker Contact string representation

        Returns:
            str: Worker Contact string representation
        """
        return textwrap.wrap(self.name, 50)[0]

    def get_absolute_url(self) -> str:
        """Worker Contact absolute url

        Returns:
            str: Worker Contact absolute url
        """
        return reverse("worker:contact-detail", kwargs={"pk": self.pk})


class WorkerComment(GenericModel):
    """
    Store worker comments related to the associated `Worker`
    Model.
    """

    worker = models.ForeignKey(
        Worker,
        on_delete=models.CASCADE,
        related_name="comments",
        related_query_name="comments",
        verbose_name=_("worker"),
        help_text=_("Related worker."),
    )
    comment_type = models.ForeignKey(
        CommentType,
        on_delete=models.CASCADE,
        related_name="comments",
        related_query_name="comments",
        verbose_name=_("comment type"),
        help_text=_("Related comment type."),
    )
    comment = models.TextField(
        _("comment"),
        help_text=_("Comment"),
    )
    entered_by = models.ForeignKey(
        User,
        on_delete=models.CASCADE,
        related_name="worker_comments",
        related_query_name="worker_comments",
        verbose_name=_("entered by"),
        help_text=_("User who entered the comment."),
    )

    class Meta:
        verbose_name = _("worker comment")
        verbose_name_plural = _("worker comments")
        ordering: list[str] = ["worker"]

    def __str__(self) -> str:
        """Worker Comment string representation

        Returns:
            str: Worker Comment string representation
        """
        return textwrap.wrap(self.comment, 50)[0]

    def get_absolute_url(self) -> str:
        """Worker Comment absolute url

        Returns:
            str: Worker Comment absolute url
        """
        return reverse("worker:comment-detail", kwargs={"pk": self.pk})
