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

from typing import Any, Optional, Type

from django.conf import settings
from django.contrib import admin, messages
from django.contrib.admin.options import IS_POPUP_VAR, csrf_protect_m
from django.contrib.admin.utils import unquote
from django.contrib.auth import update_session_auth_hash
from django.contrib.auth.admin import sensitive_post_parameters_m
from django.contrib.auth.forms import AdminPasswordChangeForm, UserChangeForm, UserCreationForm
from django.core.exceptions import PermissionDenied
from django.db import router, transaction
from django.forms.models import ModelForm
from django.http import Http404, HttpRequest, HttpResponse, HttpResponseRedirect
from django.template.response import TemplateResponse
from django.urls import URLPattern, path, reverse
from django.utils.html import escape
from django.utils.translation import gettext, gettext_lazy as _

from accounts import models
from core.generics.admin import GenericAdmin, GenericStackedInline


class ProfileInline(GenericStackedInline):
    """
    Profile inline
    """

    model: Type[models.UserProfile] = models.UserProfile
    can_delete: bool = False
    verbose_name_plural: str = "profiles"
    fk_name: str = "user"
    extra: int = 0


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
        (None, {"fields": ("organization", "username", "email", "password")}),
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
                    "username",
                    "email",
                    "password1",
                    "password2",
                ),
            },
        ),
    )
    form: Type[UserChangeForm] = UserChangeForm
    add_form: Type[UserCreationForm] = UserCreationForm
    change_password_form: Type[AdminPasswordChangeForm] = AdminPasswordChangeForm
    list_display = ("username", "email", "is_staff")
    list_filter = ("is_staff", "is_superuser", "groups")
    search_fields = ("username", "first_name", "last_name", "email")
    ordering = ("username",)
    filter_horizontal = (
        "groups",
        "user_permissions",
    )
    autocomplete_fields: tuple[str, ...] = ("organization",)
    inlines: tuple[Type[ProfileInline]] = (ProfileInline,)

    def get_fieldsets(self, request: HttpRequest, obj=None):
        """Return fieldsets for add/change view

        Args:
            request (HttpRequest): request
            obj (): object

        Returns:
            fieldsets
        """
        if not obj:
            return self.add_fieldsets
        return super().get_fieldsets(request, obj)

    def get_form(
        self,
        request: HttpRequest,
        obj: Optional[Any] = ...,
        change: bool = True,
        **kwargs: Any,
    ) -> Type[ModelForm]:
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
        defaults.update(kwargs)
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

    def _add_view(self, request, form_url="", extra_context=None) -> HttpResponse:
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
                        "%s:%s_%s_change"
                        % (
                            self.admin_site.name,
                            user._meta.app_label,
                            user._meta.model_name,
                        ),
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

    fieldsets = ((None, {"fields": ("name", "is_active", "description")}),)
    search_fields = ("name",)
    list_display = ("name", "is_active", "description")
