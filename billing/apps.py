# -*- coding: utf-8 -*-
from django.apps import AppConfig


class BillingConfig(AppConfig):
    default_auto_field = "django.db.models.BigAutoField"
    name = "billing"

    def ready(self) -> None:
        """
        Ready function for billing app
        """
        import billing.signals
