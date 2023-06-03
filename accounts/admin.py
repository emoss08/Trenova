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

from typing import Any

from django.conf import settings
from django.contrib import admin, messages
from django.contrib.admin.options import IS_POPUP_VAR, csrf_protect_m
from django.contrib.admin.utils import unquote
from django.contrib.auth import update_session_auth_hash
from django.contrib.auth.admin import sensitive_post_parameters_m
from django.contrib.auth.forms import (
    AdminPasswordChangeForm,
    UserChangeForm,
    UserCreationForm,
)
from django.core.exceptions import PermissionDenied
from django.db import router, transaction
from django.forms.models import ModelForm
from django.http import Http404, HttpRequest, HttpResponse, HttpResponseRedirect
from django.template.response import TemplateResponse
from django.urls import URLPattern, path, reverse
from django.utils.html import escape
from django.utils.translation import gettext
from django.utils.translation import gettext_lazy as _

from accounts import models
from utils.admin import GenericAdmin, GenericStackedInline


class ProfileInline(GenericStackedInline[models.User, models.UserProfile]):
    """
    Profile inline
    """

    model = models.UserProfile
    fk_name = "user"


@admin.register(models.Token)
class TokenAdmin(admin.ModelAdmin[models.Token]):
    """
    Token Admin
    """

    model = models.Token
    list_display = (
        "user",
        "created",
    )
    search_fields = (
        "user",
        "key",
    )


@admin.register(models.User)
class UserAdmin(admin.ModelAdmin[models.User]):
    """
    User Admin

    Notes:
        This code was taken from Django Admin defaults and modified to allow for
        password changes.
    """

    change_user_password_template: None = None
    fieldsets = (
        (
            None,
            {
                "fields": (
                    "is_active",
                    "organization",
                    "department",
                    "username",
                    "email",
                    "password",
                    "online",
                )
            },
        ),
        (
            "Permissions",
            {"fields": ("is_staff", "is_superuser", "groups", "user_permissions")},
        ),
        ("Important dates", {"fields": ("last_login", "date_joined")}),
    )
    add_fieldsets = (
        (
            None,
            {
                "classes": ("wide",),
                "fields": (
                    "organization",
                    "department",
                    "username",
                    "email",
                    "password1",
                    "password2",
                ),
            },
        ),
    )
    form: type[UserChangeForm] = UserChangeForm
    add_form: type[UserCreationForm] = UserCreationForm
    change_password_form: type[AdminPasswordChangeForm] = AdminPasswordChangeForm
    list_display = ("username", "email", "is_staff")
    list_filter = ("is_staff", "is_superuser", "groups")
    search_fields = ("username", "email")
    ordering = ("username",)
    filter_horizontal = (
        "groups",
        "user_permissions",
    )
    autocomplete_fields: tuple[str, ...] = ("organization", "department")
    inlines: tuple[type[ProfileInline]] = (ProfileInline,)

    def get_fieldsets(self, request: HttpRequest, obj: models.User | None = None):
        """Return fieldsets for add/change view

        Args:
            request (HttpRequest): request
            obj (models.User): object

        Returns:
            fieldsets
        """
        return super().get_fieldsets(request, obj) if obj else self.add_fieldsets

    def get_form(
        self,
        request: HttpRequest,
        obj: Any | None = ...,
        change: bool = True,
        **kwargs: Any,
    ) -> type[ModelForm[models.User]]:
        """Get form for user admin

        Args:
            change (bool): change
            request (HttpRequest): request
            obj (Optional[Any], optional): object. Defaults to ....
            **kwargs (Any): kwargs

        Returns:

        """
        defaults = {}
        if obj is None:
            defaults["form"] = self.add_form
        defaults |= kwargs
        return super().get_form(request=request, obj=obj, change=True, **defaults)

    def get_urls(self) -> list[URLPattern]:
        """Get urls for user admin

        Returns:
            list[URLPattern]: urls for user admin
        """
        return [
            path(
                "<id>/password/",
                self.admin_site.admin_view(self.user_change_password),
                name="auth_user_password_change",
            ),
        ] + super().get_urls()

    def lookup_allowed(self, lookup: str, value: str) -> bool:
        """Allow lookup for username

        Args:
            lookup (str): lookup
            value (str): value

        Returns:
            bool: True if lookup is allowed
        """
        # Don't allow lookups involving passwords.
        return not lookup.startswith("password") and super().lookup_allowed(
            lookup, value
        )

    @sensitive_post_parameters_m  # type: ignore
    @csrf_protect_m  # type: ignore
    def add_view(
        self, request: HttpRequest, form_url: str = "", extra_context: Any = None
    ) -> HttpResponse:
        """The 'add' admin view for this model.

        Args:
            request (HttpRequest): request
            form_url (str): form url
            extra_context (Any): extra context

        Returns:
            HttpResponse: response
        """
        with transaction.atomic(using=router.db_for_write(self.model)):
            return self._add_view(request, form_url, extra_context)

    def _add_view(
        self,
        request: HttpRequest,
        form_url: str = "",
        extra_context: dict | None = None,
    ) -> HttpResponse:
        # It's an error for a user to have added permission but NOT change
        # permission for users. If we allowed such users to add users, they
        # could create superusers, which would mean they would essentially have
        # the permission to change users. To avoid the problem entirely, we
        # disallow users from adding users if they don't have change
        # permission.
        if not self.has_change_permission(request):
            if self.has_add_permission(request) and settings.DEBUG:
                # Raise Http404 in debug mode so that the user gets a helpful
                # error message.
                raise Http404(
                    'Your user does not have the "Change user" permission. In '
                    "order to add users, Django requires that your user "
                    'account have both the "Add user" and "Change user" '
                    "permissions set."
                )
            raise PermissionDenied
        if extra_context is None:
            extra_context = {}
        username_field = self.model._meta.get_field(self.model.USERNAME_FIELD)
        defaults = {
            "auto_populated_fields": (),
            "username_help_text": username_field.help_text,  # type: ignore
        }
        extra_context.update(defaults)
        return super().add_view(request, form_url, extra_context)

    @sensitive_post_parameters_m  # type: ignore
    def user_change_password(
        self, request: HttpRequest, id: str, form_url: str = ""
    ) -> HttpResponseRedirect | TemplateResponse:
        """Allow a user to change their password from the admin.

        Args:
            request (HttpRequest): request object
            id (int): user id
            form_url (str): form url

        Returns:
            HttpResponseRedirect | TemplateResponse: response
        """
        user = self.get_object(request, unquote(id))
        if not self.has_change_permission(request, user):
            raise PermissionDenied
        if user is None:
            raise Http404(
                _("%(name)s object with primary key %(key)r does not exist.")
                % {
                    "name": self.model._meta.verbose_name,
                    "key": escape(id),
                }
            )
        if request.method == "POST":
            form: AdminPasswordChangeForm = self.change_password_form(
                user, request.POST
            )
            if form.is_valid():
                form.save()
                change_message = self.construct_change_message(request, form, None)  # type: ignore
                self.log_change(request, user, change_message)
                msg: str = gettext("Password changed successfully.")
                messages.success(request, msg)
                update_session_auth_hash(request, form.user)
                return HttpResponseRedirect(
                    reverse(
                        f"{self.admin_site.name}:{user._meta.app_label}_{user._meta.model_name}_change",
                        args=(user.pk,),
                    )
                )
        else:
            form: AdminPasswordChangeForm = self.change_password_form(user)  # type: ignore

        fieldsets = [(None, {"fields": list(form.base_fields)})]
        adminForm = admin.helpers.AdminForm(form, fieldsets, {})  # type: ignore

        context = {
            "title": _("Change password: %s") % escape(user.get_username()),
            "adminForm": adminForm,
            "form_url": form_url,
            "form": form,
            "is_popup": (IS_POPUP_VAR in request.POST or IS_POPUP_VAR in request.GET),
            "is_popup_var": IS_POPUP_VAR,
            "add": True,
            "change": False,
            "has_delete_permission": False,
            "has_change_permission": True,
            "has_absolute_url": False,
            "opts": self.model._meta,
            "original": user,
            "save_as": False,
            "show_save": True,
            **self.admin_site.each_context(request),
        }

        request.current_app = self.admin_site.name

        return TemplateResponse(
            request,
            "admin/auth/user/change_password.html",
            context,
        )


@admin.register(models.JobTitle)
class JobTitleAdmin(GenericAdmin[models.JobTitle]):
    """
    Job title admin
    """

    search_fields = ("name",)
    list_display = ("name", "is_active", "description")
