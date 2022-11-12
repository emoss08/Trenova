# -*- coding: utf-8 -*-
from django.apps import AppConfig


class DispatchConfig(AppConfig):
    default_auto_field = "django.db.models.BigAutoField"
    name = "dispatch"

    def ready(self) -> None:
        """Ready

        Returns:
            None
        """
        from dispatch import signals
