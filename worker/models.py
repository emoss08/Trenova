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
from typing import Any, final

from django.conf import settings
from django.core.exceptions import ValidationError
from django.core.validators import MaxValueValidator, MinValueValidator
from django.db import models
from django.urls import reverse
from django.utils import timezone
from django.utils.functional import cached_property
from django.utils.translation import gettext_lazy as _
from encrypted_model_fields.fields import EncryptedCharField
from localflavor.us.models import USStateField, USZipCodeField

from organization.models import Depot
from utils.models import ChoiceField, GenericModel

User = settings.AUTH_USER_MODEL


class Worker(GenericModel):  # type:ignore
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
    fleet = models.ForeignKey(
        "dispatch.FleetCode",
        on_delete=models.CASCADE,
        related_name="worker",
        related_query_name="workers",
        verbose_name=_("Fleet"),
        help_text=_("The fleet of the worker."),
    )
    zip_code = USZipCodeField(
        _("zip code"),
        help_text=_("The zip code of the worker."),
    )
    depot = models.ForeignKey(
        Depot,
        on_delete=models.CASCADE,
        related_name="worker",
        related_query_name="workers",
        verbose_name=_("Depot"),
        help_text=_("The depot of the worker."),
        null=True,
        blank=True,
    )
    manager = models.ForeignKey(
        User,
        on_delete=models.CASCADE,
        related_name="worker",
        related_query_name="workers",
        verbose_name=_("Manager"),
        help_text=_("The manager of the worker."),
    )
    entered_by = models.ForeignKey(
        User,
        on_delete=models.CASCADE,
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
        ordering = ["code"]
        indexes = [
            models.Index(fields=["first_name", "last_name"]),
        ]
        db_table = "worker"
        constraints = [
            models.UniqueConstraint(
                fields=["code", "organization"],
                name="unique_worker_code_organization",
            )
        ]

    def __str__(self) -> str:
        """Worker string representation

        Returns:
            str: Worker string representation
        """

        return textwrap.wrap(f"{self.first_name} {self.last_name}", 50)[0]

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

        return (
            f"{self.address_line_1} {self.address_line_2}"
            f" {self.city} {self.state} {self.zip_code}"
        )

    def get_absolute_url(self) -> str:
        """Worker absolute url

        Returns:
            str: Worker absolute url
        """

        return reverse("worker:detail", kwargs={"pk": self.pk})

    def update_worker(self, **kwargs: Any) -> None:
        """
        Updates the user with the given kwargs
        """
        for key, value in kwargs.items():
            setattr(self, key, value)
        self.save()


class WorkerProfile(GenericModel):
    """
    Stores the worker profile information related to the :model:`worker.Worker`.
    """

    @final
    class WorkerSexChoices(models.TextChoices):
        """
        Worker Sex/Gender Choices
        """

        MALE = "MALE", _("Male")
        FEMALE = "FEMALE", _("Female")
        NON_BINARY = "NON-BINARY", _("Non-binary")
        OTHER = "OTHER", _("Other")

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
    license_number = EncryptedCharField(
        _("License Number"),
        max_length=20,
        help_text=_("Driver License Number"),
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
        default=timezone.now,
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
        db_table = "worker_profile"

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
            None

        Raises:
            ValidationError: If the worker profile is not valid.
        """
        from dispatch.validators.regulatory import (
            validate_worker_regulatory_information,
        )

        super().clean()

        # TODO(Wolfred): Rewrite this to raise the validation all at once.

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
                        "Hazmat expiration date is required for this endorsement. Please try again."
                    ),
                },
                code="invalid",
            )

        if (
            self.date_of_birth
            and (timezone.now().date() - self.date_of_birth).days < 6570
        ):
            raise ValidationError(
                {
                    "date_of_birth": _(
                        "Worker must be at least 18 years old to be entered. Please try again."
                    ),
                },
                code="invalid",
            )

        if self.license_number and not self.license_state:
            raise ValidationError(
                {
                    "license_state": _(
                        "You must provide license state. Please try again."
                    ),
                },
                code="invalid",
            )

        if self.license_number and not self.license_expiration_date:
            raise ValidationError(
                {
                    "license_expiration_date": _(
                        "You must provide license expiration date. Please try again."
                    )
                },
                code="invalid",
            )

        validate_worker_regulatory_information(self)

    def get_absolute_url(self) -> str:
        """Worker Profile absolute url

        Returns:
            str: Worker Profile absolute url
        """

        return reverse("worker:profile-detail", kwargs={"pk": self.pk})

    def update_worker_profile(self, **kwargs):
        """Update the worker profile

        Args:
            **kwargs: Keyword arguments
        """

        for key, value in kwargs.items():
            setattr(self, key, value)
        self.save()


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
        db_table = "worker_contact"

    def __str__(self) -> str:
        """Worker Contact string representation

        Returns:
            str: Worker Contact string representation
        """
        return textwrap.wrap(self.name, 50)[0]

    def update_worker_contact(self, **kwargs: Any) -> None:
        """Update the location contact

        Args:
            **kwargs: Keyword arguments

        Returns:
            None: This function does not return anything.
        """

        for key, value in kwargs.items():
            setattr(self, key, value)
        self.save()

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
        related_query_name="comment",
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
        db_table = "worker_comment"

    def __str__(self) -> str:
        """Worker Comment string representation

        Returns:
            str: Worker Comment string representation
        """

        return textwrap.wrap(self.comment, 50)[0]

    def update_worker_comment(self, **kwargs: Any) -> None:
        """Update the worker comment

        Args:
            **kwargs: Keyword arguments

        Returns:
            None: This function does not return anything.
        """

        for key, value in kwargs.items():
            setattr(self, key, value)
        self.save()

    def get_absolute_url(self) -> str:
        """Worker Comment absolute url

        Returns:
            str: Worker Comment absolute url
        """

        return reverse("worker:comment-detail", kwargs={"pk": self.pk})


class WorkerTimeAway(GenericModel):
    """
    A Django model representing a worker's time off.

    Attributes:
        id (UUIDField): The primary key field for the worker time away instance. This field is automatically generated
            using the uuid4 function from the uuid module.
        worker (ForeignKey): A foreign key that relates the worker to the worker time away. When a worker instance is
            deleted, all related worker time away instances will be deleted as well.
        start_date (DateField): The date field representing the start date of the time away.
        end_date (DateField): The date field representing the end date of the time away.
        leave_type (ChoiceField): The choice field representing the type of leave the worker is taking. It uses a
            nested class
            LeaveTypeChoices that extends Django's built-in TextChoices class to provide a list of choices for the
            leave_type field.

    Methods:
        __str__(): A method that returns a string representation of the worker time away. In this case, the code of the
            worker associated with the time away is wrapped at 50 characters.
        get_absolute_url(): A method that returns the URL to view the detail of the worker time away. It uses Django's
            reverse function to generate the URL based on the view name and the primary key of the worker time away
            instance.

    Meta:
        verbose_name (str): A human-readable name for the model. In this case, "Worker Time Away".
        verbose_name_plural (str): A human-readable plural name for the model. In this case, "Worker Time Away".
        ordering (list): A list of fields to use when ordering the model instances. In this case, the instances are
            ordered by the worker field.
        db_table (str): The name of the database table to use for the model. In this case, "worker_time_away".
    """

    @final
    class LeaveTypeChoices(models.TextChoices):
        """
        A class that defines the choices for the leave_type field in the WorkerTimeAway model.
        The choices are defined as class constants that consist of a string value and a human-readable label.

        Attributes:
            VAC (str, str): A tuple that represents the "Vacation" leave type choice. The first element is
             the string value "VAC", and the second element is the human-readable label "Vacation".
            PER (str, str): A tuple that represents the "Personal" leave type choice. The first element is
             the string value "PERS", and the second element is the human-readable label "Personal".
            HOL (str, str): A tuple that represents the "Holiday" leave type choice. The first element is
             the string value "HOL", and the second element is the human-readable label "Holiday".
            SICK (str, str): A tuple that represents the "Sick" leave type choice. The first element is
            the string value "SICK", and the second element is the human-readable label "Sick".
        """

        VAC = "VAC", _("Vacation")
        PER = "PERS", _("Personal")
        HOL = "HOL", _("Holiday")
        SICK = "SICK", _("Sick")

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    worker = models.ForeignKey(
        Worker,
        on_delete=models.CASCADE,
        related_name="worker_to",
        related_query_name="worker_to",
        verbose_name=_("worker"),
        help_text=_("Related worker."),
    )
    start_date = models.DateField(
        _("Start Date"), help_text=_("Start date for Worker Time Off")
    )
    end_date = models.DateField(
        _("End Date"), help_text=_("End date for Worker Time Off")
    )
    leave_type = ChoiceField(
        _("Leave Type"), choices=LeaveTypeChoices.choices, help_text=_("Type of Leave")
    )

    class Meta:
        """Model meta options for Worker Time Away.

        Attributes:
            verbose_name (str): A human-readable name for the model. The default value is "Worker Time Away".
            verbose_name_plural (str): A human-readable plural name for the model. The default value is
            "Worker Time Away".
            ordering (list of str): A list of model field names used to specify the default ordering of records. The
            default value is ["worker"].
            db_table (str): The name of the database table to use for the model. The default value is
            "worker_time_away".
        """

        verbose_name = _("Worker Time Away")
        verbose_name_plural = _("Worker Time Away")
        ordering = ["worker"]
        db_table = "worker_time_away"

    def __str__(self) -> str:
        """
        Returns the string representation of the Worker Time Away.

        Returns:
            String representation of the WorkerTimeAway model. For example,
            "Worker Time Away: 2021-01-01 to 2021-01-02".

        """
        return textwrap.wrap(
            f"Worker Time Away {self.start_date} to {self.end_date}", 50
        )[0]

    def get_absolute_url(self) -> str:
        """
        Returns the absolute URL to view the detail of the Worker Time Away.

        Returns:
            Absolute URL for the WorkerTimeAway object. For Example,
            `/worker_time_away/edd1e612-cdd4-43d9-b3f3-bc099872088b/`.
        """
        return reverse("worker-time-away-detail", kwargs={"pk": self.pk})


class WorkerHOS(GenericModel):
    """
    A Django model representing a worker's hours of service.

    Attributes:
        id (UUIDField): The UUID field representing the primary key of the model. It is set to be the primary key,
            read-only, and unique.
        worker (ForeignKey): The foreign key field representing the worker associated with the hours of service.
            It uses the Worker model as the related model, and it uses the CASCADE delete rule.
        drive_time (PositiveIntegerField): The positive integer field representing the drive time in minutes.
        off_duty_time (PositiveIntegerField): The positive integer field representing the off duty time in minutes.
        sleeper_berth_time (PositiveIntegerField): The positive integer field representing the sleeper berth time in
            minutes.
        on_duty_time (PositiveIntegerField): The positive integer field representing the on duty time in minutes.

    Methods:
        __str__ (str): Returns the string representation of the Worker HOS.
        get_absolute_url (str): Returns the absolute URL to view the detail of the Worker HOS.

    Meta:
        verbose_name (str): A human-readable name for the model. The default value is "Worker HOS".
        verbose_name_plural (str): A human-readable plural name for the model. The default value is "Worker HOS".
        ordering (list of str): A list of model field names used to specify the default ordering of records. The
            default value is ["worker"].
        db_table (str): The name of the database table to use for the model. The default value is "worker_hos".
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
        related_name="worker_hos",
        related_query_name="worker_hos",
        verbose_name=_("worker"),
        help_text=_("Related worker."),
    )
    drive_time = models.PositiveIntegerField(
        _("Drive Time"), help_text=_("Drive time in minutes")
    )
    off_duty_time = models.PositiveIntegerField(
        _("Off Duty Time"), help_text=_("Off duty time in minutes")
    )
    sleeper_berth_time = models.PositiveIntegerField(
        _("Sleeper Berth Time"), help_text=_("Sleeper berth time in minutes")
    )
    on_duty_time = models.PositiveIntegerField(
        _("On Duty Time"), help_text=_("On duty time in minutes")
    )
    violation_time = models.PositiveIntegerField(
        _("Violation Time"), help_text=_("Violation time in minutes")
    )
    current_status = models.CharField(
        _("Current Status"), max_length=50, help_text=_("Current status of the driver")
    )
    current_location = models.CharField(
        _("Current Location"),
        max_length=50,
        help_text=_("Current location of the driver"),
    )
    log_date = models.DateField(_("Log Date"), help_text=_("Log date"))
    last_reset_date = models.DateField(
        _("Last Reset Date"), help_text=_("Last reset date")
    )

    class Meta:
        """Model meta options for Worker HOS.

        Attributes:
            verbose_name (str): A human-readable name for the model. The default value is "Worker HOS".
            verbose_name_plural (str): A human-readable plural name for the model. The default value is
            "Worker HOS".
            ordering (list of str): A list of model field names used to specify the default ordering of records. The
            default value is ["worker"].
            db_table (str): The name of the database table to use for the model. The default value is
            "worker_hos".
        """

        verbose_name = _("Worker HOS")
        verbose_name_plural = _("Worker HOS")
        ordering = ["worker"]
        db_table = "worker_hos"

    def __str__(self) -> str:
        """
        Returns the string representation of the Worker HOS.

        Returns:
            String representation of the WorkerHOS model. For example,
            "Worker HOS: 2021-01-01".
        """
        return textwrap.wrap(f"Worker HOS {self.log_date}", 50)[0]

    def get_absolute_url(self) -> str:
        """
        Returns the absolute URL to view the detail of the Worker HOS.

        Returns:
            Absolute URL for the WorkerHOS object. For Example,
            `/worker_hos/edd1e612-cdd4-43d9-b3f3-bc099872088b/`.
        """
        return reverse("worker-hos-detail", kwargs={"pk": self.pk})
