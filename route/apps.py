# -*- coding: utf-8 -*-
from django.apps import AppConfig


class RouteConfig(AppConfig):
    default_auto_field = "django.db.models.BigAutoField"
    name = "route"

    def ready(self) -> None:
        """
        Ready Function

        Returns:
            None
        """
        from route import signals
