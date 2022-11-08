from accounts.models import User as User, UserProfile as UserProfile
from django.contrib import admin
from django.db.models import Model, QuerySet as QuerySet
from django.forms import BaseModelForm
from django.http import HttpRequest
from organization.models import Organization as Organization
from typing import Optional, Type

class GenericModel(Model):
    organization: Organization

class AuthHttpRequest(HttpRequest):
    user: User
    @property
    def profile(self) -> Optional[UserProfile]: ...

class GenericAdmin(admin.ModelAdmin):
    exclude: Incomplete
    def get_queryset(self, request: HttpRequest) -> QuerySet[Model]: ...
    def save_model(
        self,
        request: AuthHttpRequest,
        obj: GenericModel,
        form: Type[BaseModelForm],
        change: bool,
    ) -> None: ...
    def save_formset(self, request, form, formset, change) -> None: ...
