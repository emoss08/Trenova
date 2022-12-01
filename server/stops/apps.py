from django.apps import AppConfig


class StopsConfig(AppConfig):
    default_auto_field = "django.db.models.BigAutoField"
    name = "stops"

    def ready(self):
        from stops import signals
