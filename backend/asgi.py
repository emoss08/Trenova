"""
ASGI config for monta project.

It exposes the ASGI callable as a module-level variable named ``application``.

For more information on this file, see
https://docs.djangoproject.com/en/4.1/howto/deployment/asgi/
"""
import os

from art import *
from channels.routing import ProtocolTypeRouter
from django.core.asgi import get_asgi_application
from rich.console import Console

console = Console()
logo = text2art("MONTA", font="Larry 3D")
console.print(logo, style="bold purple")

os.environ.setdefault("DJANGO_SETTINGS_MODULE", "backend.settings")

application = get_asgi_application()
application = ProtocolTypeRouter(
    {
        "http": application,
    }
)
