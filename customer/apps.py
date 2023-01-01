from django.apps import AppConfig


class CustomerConfig(AppConfig):
    default_auto_field = "django.db.models.BigAutoField"
    name = "customer"

    def ready(self) -> None:
        """
        Ready function for billing app
        """
        import customer.signals
