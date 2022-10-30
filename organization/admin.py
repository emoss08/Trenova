# -*- coding: utf-8 -*-
from django.contrib import admin

from .models import Organization


@admin.register(Organization)
class OrganizationAdmin(admin.ModelAdmin):
    """
    Organization Admin
    """
    list_display: tuple[str, ...] = (
        "name",
        "scac_code",
        "org_type",
        "timezone",
    )
    list_filter: tuple[str, ...] = (
        "org_type",
    )
    search_fields: tuple[str, ...] = (
        "name",
        "scac_code",
    )
