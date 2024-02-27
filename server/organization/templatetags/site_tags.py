from django import template
from django.conf import settings

register = template.Library()


@register.simple_tag
def base_url():
    return getattr(settings, "BASE_URL", "http://localhost:8000/")
