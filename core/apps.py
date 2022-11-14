from django.apps import AppConfig


class CoreConfig(AppConfig):
    default_auto_field = "django.db.models.BigAutoField"
    name = "core"

    def ready(self) -> None:
        """
        Ready.

        Returns:
            None
        """
        from core.management.commands.formatter import Command

        Command().handle()
