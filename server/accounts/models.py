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

from django.contrib.auth.models import (
    AbstractBaseUser,
    BaseUserManager,
    PermissionsMixin,
)
from django.core.exceptions import ValidationError
from django.core.validators import MinLengthValidator, RegexValidator
from django.db import models
from django.db.models.functions import Lower
from django.urls import reverse
from django.utils import timezone
from django.utils.functional import cached_property
from django.utils.translation import gettext_lazy as _
from localflavor.us.models import USStateField, USZipCodeField

from accounts import services
from utils.models import ChoiceField, GenericModel, PrimaryStatusChoices


class UserManager(BaseUserManager):
    """
    Base user manager
    """

    def create_user(
        self,
        username: str,
        email: str,
        password: str | None = None,
        **extra_fields: Any,
    ) -> User:
        """
        Create and save a user with the given email and password.

        Args:
            username (str): Username of the user.
            email (str): Email address of the user.
            password (str | None, optional): Password for the user. Defaults to None.
            **extra_fields (Any): Extra fields for the user.

        Returns:
            User: User object.
        """

        if not username:
            raise ValueError(_("The username must be set"))
        if not email:
            raise ValueError(_("The email must be set"))

        user: User = self.model(  # type: ignore
            username=username.lower(),
            email=self.normalize_email(email),
            **extra_fields,
        )
        user.set_password(password)
        user.save()
        return user

    def create_superuser(
        self,
        username: str,
        email: str,
        password: str | None = None,
        **extra_fields: Any,
    ) -> User:
        """Create and save a superuser with the given username, email and password.

        Args:
            username (str): Username of the user.
            email (str): Email address of the user.
            password (str): Password for the user.
            **extra_fields (str): Extra fields for the user.
        """
        extra_fields.setdefault("is_staff", True)
        extra_fields.setdefault("is_superuser", True)

        if extra_fields.get("is_staff") is not True:
            raise ValueError(_("Superuser must have is_staff=True."))
        if extra_fields.get("is_superuser") is not True:
            raise ValueError(_("Superuser must have is_superuser=True."))

        return self.create_user(username, email, password, **extra_fields)


class User(AbstractBaseUser, PermissionsMixin):
    """
    Stores basic user information.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
        help_text=_("Unique ID for the user."),
    )
    business_unit = models.ForeignKey(
        "organization.BusinessUnit",
        on_delete=models.CASCADE,
        related_name="users",
        related_query_name="user",
        verbose_name=_("Business Unit"),
    )
    organization = models.ForeignKey(
        "organization.Organization",
        on_delete=models.CASCADE,
        related_name="users",
        related_query_name="user",
        verbose_name=_("Organization"),
    )
    department = models.ForeignKey(
        "organization.Department",
        on_delete=models.CASCADE,
        related_name="users",
        related_query_name="user",
        verbose_name=_("Department"),
        null=True,
        blank=True,
    )
    is_active = models.BooleanField(
        _("Active"),
        default=True,
        help_text=_(
            "Designates whether this user should be treated as active. "
            "Unselect this instead of deleting accounts."
        ),
    )
    username = models.CharField(
        _("Username"),
        max_length=30,
        unique=True,
        help_text=_(
            "Required. 30 characters or fewer. Letters, digits and @/./+/-/_ only."
        ),
    )
    email = models.EmailField(
        _("Email Address"),
        unique=True,
        help_text=_("Required. A valid email address."),
    )
    is_staff = models.BooleanField(
        _("Staff Status"),
        default=False,
        help_text=_("Designates whether the user can log into this admin site."),
    )
    date_joined = models.DateTimeField(_("Date Joined"), default=timezone.now)
    online = models.BooleanField(
        _("Online"),
        default=False,
        help_text=_("Designates whether the user is currently online."),
    )
    session_key = models.CharField(
        _("Session Key"),
        max_length=40,
        blank=True,
        null=True,
        help_text=_("Stores the current session key."),
    )

    objects = UserManager()

    # To get around `Unresolved Attribute` problem with AbstractBaseUser, we have to
    # define the UserProfile here. This is a bit of a hack, but it works.
    # It will also give proper autocomplete in IDEs.
    profile: UserProfile
    USERNAME_FIELD = "username"
    REQUIRED_FIELDS: list[str] = [
        "email",
    ]

    class Meta:
        """
        Metaclass for the User Model
        """

        verbose_name = _("User")
        verbose_name_plural = _("Users")
        db_table = "user"
        permissions = [("admin.users.view", "Can view all users")]

    def __str__(self) -> str:
        """
        Returns:
            str: String representation of the User
        """
        return self.username

    def get_absolute_url(self) -> str:
        """
        Returns:
            str: Absolute URL for the User
        """
        return reverse("users-detail", kwargs={"pk": self.pk})

    def update_user(self, **kwargs: Any) -> None:
        """
        Updates the user with the given kwargs
        """
        for key, value in kwargs.items():
            setattr(self, key, value)
        self.save()


class UserProfile(GenericModel):
    """
    Stores additional information for a related :model:`accounts.User`.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    user = models.OneToOneField(
        User,
        on_delete=models.CASCADE,
        related_name="profile",
        related_query_name="profiles",
        verbose_name=_("User"),
    )
    job_title = models.ForeignKey(
        "JobTitle",
        on_delete=models.PROTECT,
        related_name="profile",
        related_query_name="profiles",
        verbose_name=_("Job Title"),
    )
    first_name = models.CharField(
        _("First Name"),
        max_length=255,
        help_text=_("The first name of the user"),
    )
    last_name = models.CharField(
        _("Last Name"),
        max_length=255,
        help_text=_("The last name of the user"),
    )
    profile_picture = models.ImageField(
        _("Profile Picture"),
        upload_to="user_profiles/pictures",
        help_text=_("The profile picture of the user"),
        null=True,
        blank=True,
    )
    address_line_1 = models.CharField(
        _("Address"),
        max_length=100,
        help_text=_("The address line 1 of the user"),
    )
    address_line_2 = models.CharField(
        _("Address Line 2"),
        max_length=100,
        blank=True,
        help_text=_("The address line 2 of the user"),
    )
    city = models.CharField(
        _("City"),
        max_length=100,
        help_text=_("The city of the user"),
    )
    state = USStateField(
        _("State"),
        help_text=_("The state of the user"),
    )
    zip_code = USZipCodeField(
        _("Zip Code"),
        help_text=_("The zip code of the user"),
    )
    phone_number = models.CharField(
        _("Phone Number"),
        max_length=15,
        blank=True,
        help_text=_("The phone number of the user"),
        validators=[
            RegexValidator(
                regex=r"^\(?([0-9]{3})\)?[-. ]?([0-9]{3})[-. ]?([0-9]{4})$",
                message=_("Phone number must be in the format (xxx) xxx-xxxx"),
            )
        ],
    )
    is_phone_verified = models.BooleanField(
        _("Phone Number Verified"),
        default=False,
        help_text=_("Designates whether the user's phone number has been verified."),
    )

    class Meta:
        """
        Metaclass for the Profile model
        """

        ordering = ["-created"]
        verbose_name = _("Profile")
        verbose_name_plural = _("Profiles")
        indexes = [
            models.Index(fields=["-created"]),
        ]
        db_table = "user_profile"

    def __str__(self) -> str:
        """Profile string representation

        Returns:
            str: String representation of the Profile
        """
        return textwrap.wrap(self.user.username, 30)[0]

    def save(self, *args: Any, **kwargs: Any) -> None:
        """Save the model

        Returns:
            None
        """
        self.first_name = self.first_name.title()
        self.last_name = self.last_name.title()

        super().save(*args, **kwargs)

    def get_absolute_url(self) -> str:
        """Absolute URL for the Profile.

        Returns:
            str: Get the absolute url of the Profile
        """
        return reverse("user:profile-view", kwargs={"pk": self.pk})

    def update_profile(self, **kwargs: Any) -> None:
        """
        Updates the profile with the given kwargs

        Args:
            kwargs: Keyword Arguments
        """

        for key, value in kwargs.items():
            setattr(self, key, value)
        self.save()

    def clean(self) -> None:
        """Clean the model

        Returns:
            None

        Raises:
            ValidationError: Validation error for the UserProfile Model
        """

        if self.job_title.status == PrimaryStatusChoices.INACTIVE:
            raise ValidationError(
                {
                    "title": _(
                        "The selected job title is not active. Please select a different job title.",
                    )
                },
                code="invalid",
            )

    @property
    def get_user_profile_pic(self) -> Any | str:
        """Get the user profile picture.

        Returns:
            str: Get the user profile picture
        """
        if self.profile_picture:
            return self.profile_picture.url
        return "/static/media/avatars/blank.avif"

    @cached_property
    def get_full_address_combo(self) -> str:
        """get the full address combo

        Returns:
            str: Get the full address combo
        """
        return f"{self.address_line_1} {self.address_line_2} {self.city} {self.state} {self.zip_code}"

    @cached_property
    def get_user_city_state(self) -> str | None:
        """User City and state combination.

        Returns:
            str: Get the city and state of the user
        """
        return f"{self.city}, {self.state}"

    @cached_property
    def get_full_name(self) -> str:
        """Full name of the user.

        Returns:
            str: Get the full name of the user
        """
        return textwrap.shorten(
            f"{self.first_name} {self.last_name}",
            width=30,
            placeholder="...",
        )


class JobTitle(GenericModel):
    """
    Stores the job title of a :model:`accounts.User`.

    Attributes:
        id (UUIDField): The primary key of the model
        status (ChoiceField): The status of the job title
        name (CharField): The name of the job title
        description (TextField): The description of the job title
        job_function (CharField): The job function of the job title

    Methods:
        __str__ (str): String representation of the JobTitle model

    """

    @final
    class JobFunctionChoices(models.TextChoices):
        """
        A class representing the possible job function choices.

        This class inherits from the `models.TextChoices` class and defines eight constants:
        - MANAGER: represents the manager job function
        - MANAGEMENT_TRAINEE: represents the management trainee job function
        - SUPERVISOR: represents the supervisor job function
        - DISPATCHER: represents the dispatcher job function
        - BILLING: represents the billing job function
        - FINANCE: represents the finance job function
        - SAFETY: represents the safety job function
        - SYS_ADMIN: represents the system administrator job function
        """

        MANAGER = "MANAGER", _("Manager")
        MANAGEMENT_TRAINEE = "MANAGEMENT_TRAINEE", _("Management Trainee")
        SUPERVISOR = "SUPERVISOR", _("Supervisor")
        DISPATCHER = "DISPATCHER", _("Dispatcher")
        BILLING = "BILLING", _("Billing")
        FINANCE = "FINANCE", _("Finance")
        SAFETY = "SAFETY", _("Safety")
        SYS_ADMIN = "SYS_ADMIN", _("System Administrator")
        TEST = "TEST", _("Test Job Function")

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    status = ChoiceField(
        _("Status"),
        choices=PrimaryStatusChoices.choices,
        help_text=_("Status of the job title."),
        default=PrimaryStatusChoices.ACTIVE,
    )
    name = models.CharField(
        _("Name"),
        max_length=100,
        help_text=_("Name of the job title"),
    )
    description = models.TextField(
        _("Description"),
        blank=True,
        help_text=_("Description of the job title"),
    )
    job_function = ChoiceField(
        _("Job Function"),
        choices=JobFunctionChoices.choices,
        help_text=_("Relevant job function of the job title."),
    )

    class Meta:
        """
        Metaclass for the JobTitle model
        """

        verbose_name = _("Job Title")
        verbose_name_plural = _("Job Titles")
        ordering = ["name"]
        db_table = "job_title"
        constraints = [
            models.UniqueConstraint(
                Lower("name"),
                "organization",
                name="unique_job_title_organization",
            )
        ]

    def __str__(self) -> str:
        """Job Title string representation.

        Returns:
            str: String representation of the JobTitle Model.
        """
        return textwrap.wrap(self.name, 30)[0]

    def get_absolute_url(self) -> str:
        """Absolute URL for the JobTitle.

        Returns:
            str: Get the absolute url of the job title
        """
        return reverse("user:job-title-view", kwargs={"pk": self.pk})


class Token(models.Model):
    """
    Stores the token for a :model:`accounts.User
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    user = models.ForeignKey(
        User,
        on_delete=models.CASCADE,
        related_name="tokens",
        related_query_name="token",
        help_text=_("The user that the token belongs to"),
    )
    created = models.DateTimeField(
        _("Created"),
        auto_now_add=True,
        help_text=_("The date and time the token was created"),
    )
    expires = models.DateTimeField(
        _("Expires"),
        blank=True,
        null=True,
        help_text=_("The date and time the token expires"),
    )
    last_used = models.DateTimeField(
        _("Last Used"),
        null=True,
        blank=True,
        help_text=_("The date and time the token was last used"),
    )
    key = models.CharField(
        max_length=40,
        unique=True,
        validators=[MinLengthValidator(40)],
    )

    class Meta:
        """
        Metaclass for the Token model
        """

        verbose_name = _("Token")
        verbose_name_plural = _("Tokens")
        db_table = "auth_token"

    def __str__(self) -> str:
        """Token string representation.
        Returns:
            str: String representation of the Token Model.
        """
        return textwrap.shorten(
            f"{self.key[:10]} ({self.user.username})", width=30, placeholder="..."
        )

    def save(self, *args: Any, **kwargs: Any) -> None:
        """Save the model

        Returns:
            None: This function does not return anything.
        """
        if not self.key:
            self.key = services.generate_key()

        super().save(*args, **kwargs)

    @property
    def is_expired(self) -> bool:
        """
        Checks if the token is expired.
        """

        return self.expires is not None and timezone.now() >= self.expires
