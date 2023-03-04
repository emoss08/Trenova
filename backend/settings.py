"""
COPYRIGHT 2022 MONTA

This file is part of Monta.

Monta is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

Monta is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with Monta.  If not, see <https://www.gnu.org/licenses/>.
"""

# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2023 MONTA                                                                         -
#                                                                                                  -
#  This file is part of Monta.                                                                     -
#                                                                                                  -
#  Monta is free software: you can redistribute it and/or modify                                   -
#  it under the terms of the GNU General Public License as published by                            -
#  the Free Software Foundation, either version 3 of the License, or                               -
#  (at your option) any later version.                                                             -
#                                                                                                  -
#  Monta is distributed in the hope that it will be useful,                                        -
#  but WITHOUT ANY WARRANTY; without even the implied warranty of                                  -
#  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the                                   -
#  GNU General Public License for more details.                                                    -
#                                                                                                  -
#  You should have received a copy of the GNU General Public License                               -
#  along with Monta.  If not, see <https://www.gnu.org/licenses/>.                                 -
# --------------------------------------------------------------------------------------------------

import os
from pathlib import Path

import django_stubs_ext
import environ

django_stubs_ext.monkeypatch()

env = environ.Env(DEBUG=(bool, False))

# Build paths inside the project like this: BASE_DIR / 'subdir'.
BASE_DIR = Path(__file__).resolve().parent.parent
environ.Env.read_env(os.path.join(BASE_DIR, ".env"))
SECRET_KEY = env("SECRET_KEY")
DEBUG = env("DEBUG")
CORS_ORIGIN_ALLOW_ALL = True
INTERNAL_IPS = [
    "127.0.0.1",
]

# Application definition
INSTALLED_APPS = [
    # Django Apps
    "daphne",
    "django.contrib.admin",
    "django.contrib.auth",
    "django.contrib.contenttypes",
    "django.contrib.sessions",
    "django.contrib.messages",
    "django.contrib.staticfiles",
    "django.contrib.admindocs",
    # Third-Party apps
    "django_extensions",
    "localflavor",
    "cacheops",
    "rest_framework",
    "corsheaders",
    "django_filters",
    "phonenumber_field",
    "compressor",
    "django_celery_results",
    "django_celery_beat",
    "silk",
    "encrypted_model_fields",
    "pgtrigger",
    "nested_inline",
    "drf_spectacular",
    "auditlog",
    "djmoney",
    "notifications",
    # Monta Apps
    "backend",
    "core",
    "accounts",
    "organization",
    "integration",
    "equipment",
    "worker",
    "dispatch",
    "location",
    "order",
    "route",
    "billing",
    "customer",
    "accounting",
    "stops",
    "movements",
    "commodities",
    "fuel",
    "invoicing",
]

# Middleware configurations
MIDDLEWARE = [
    "silk.middleware.SilkyMiddleware",
    "django.middleware.security.SecurityMiddleware",
    "whitenoise.middleware.WhiteNoiseMiddleware",
    "django.contrib.sessions.middleware.SessionMiddleware",
    "django.middleware.common.CommonMiddleware",
    "django.middleware.csrf.CsrfViewMiddleware",
    "corsheaders.middleware.CorsMiddleware",
    "django.contrib.auth.middleware.AuthenticationMiddleware",
    "django.contrib.messages.middleware.MessageMiddleware",
    "django.middleware.clickjacking.XFrameOptionsMiddleware",
    "auditlog.middleware.AuditlogMiddleware",
]
ROOT_URLCONF = "backend.urls"
TEMPLATES = [
    {
        "BACKEND": "django.template.backends.django.DjangoTemplates",
        "DIRS": [os.path.join(BASE_DIR, "templates")],
        "APP_DIRS": True,
        "OPTIONS": {
            "context_processors": [
                "django.template.context_processors.debug",
                "django.template.context_processors.request",
                "django.contrib.auth.context_processors.auth",
                "django.contrib.messages.context_processors.messages",
            ],
        },
    },
]
ASGI_APPLICATION = "backend.asgi.application"

# Databases
DATABASES = {
    "default": {
        "ENGINE": "django.db.backends.postgresql",
        "NAME": env("DB_NAME"),
        "USER": env("DB_USER"),
        "PASSWORD": env("DB_PASSWORD"),
        "HOST": "localhost",
        "PORT": 5432,
        "ATOMIC_REQUESTS": True,
        "CONN_HEALTH_CHECK": True,
    }
}

# Internationalization
LANGUAGE_CODE = "en-us"
TIME_ZONE = "US/Eastern"
# TIME_ZONE = "UTC" if DEBUG else TIME_ZONE
USE_I18N = True
USE_TZ = True

# Static files (CSS, JavaScript, Images)
STATIC_URL = "/static/"
STATICFILES_DIRS = (os.path.join(BASE_DIR, "static"),)
STATIC_ROOT = os.path.join(BASE_DIR, "staticfiles")
STATICFILES_FINDERS = (
    "django.contrib.staticfiles.finders.FileSystemFinder",
    "django.contrib.staticfiles.finders.AppDirectoriesFinder",
    "compressor.finders.CompressorFinder",
)
STATICFILES_STORAGE = "whitenoise.storage.CompressedManifestStaticFilesStorage"

# Media Configurations
MEDIA_DIR = os.path.join(BASE_DIR, "media")
MEDIA_ROOT = MEDIA_DIR
MEDIA_URL = "/media/"

# Default primary key field type
DEFAULT_AUTO_FIELD = "django.db.models.BigAutoField"

# AUTH MODEL
AUTH_USER_MODEL = "accounts.User"
AUTH_PASSWORD_VALIDATORS = [
    {
        "NAME": "django.contrib.auth.password_validation.UserAttributeSimilarityValidator",
    },
    {
        "NAME": "django.contrib.auth.password_validation.MinimumLengthValidator",
    },
    {
        "NAME": "django.contrib.auth.password_validation.CommonPasswordValidator",
    },
    {
        "NAME": "django.contrib.auth.password_validation.NumericPasswordValidator",
    },
]

# REDIS Configurations
CACHES = {
    "default": {
        "BACKEND": "django_redis.cache.RedisCache",
        "LOCATION": "redis://127.0.0.1:6379/1",
        "OPTIONS": {
            "CLIENT_CLASS": "django_redis.client.DefaultClient",
            "PREFIX": "default",
        },
    },
    "sessions": {
        "BACKEND": "django_redis.cache.RedisCache",
        "LOCATION": "redis://127.0.0.1:6379/0",
        "OPTIONS": {
            "CLIENT_CLASS": "django_redis.client.DefaultClient",
            "PREFIX": "sessions",
            "PARSER_CLASS": "redis.connection.HiredisParser",
            "CONNECTION_POOL_KWARGS": {
                "max_connections": 100,
                "retry_on_timeout": True,
            },
        },
    },
    "celery": {
        "BACKEND": "django_redis.cache.RedisCache",
        "LOCATION": "redis://127.0.0.1:6379/2",
        "OPTIONS": {
            "CLIENT_CLASS": "django_redis.client.DefaultClient",
            "PREFIX": "sessions",
            "PARSER_CLASS": "redis.connection.HiredisParser",
            "CONNECTION_POOL_KWARGS": {
                "max_connections": 100,
                "retry_on_timeout": True,
            },
        },
    },
    "orders": {
        "BACKEND": "django_redis.cache.RedisCache",
        "LOCATION": "redis://127.0.0.1:6379/3",
        "OPTIONS": {
            "CLIENT_CLASS": "django_redis.client.DefaultClient",
            "PREFIX": "sessions",
            "PARSER_CLASS": "redis.connection.HiredisParser",
            "CONNECTION_POOL_KWARGS": {
                "max_connections": 100,
                "retry_on_timeout": True,
            },
        },
    },
}

# Session Configurations
SESSION_ENGINE = "django.contrib.sessions.backends.cache"
SESSION_CACHE_ALIAS = "sessions"

# Rest Framework Configurations
REST_FRAMEWORK = {
    "DEFAULT_AUTHENTICATION_CLASSES": [
        "accounts.authentication.TokenAuthentication",
    ],
    "DEFAULT_PERMISSION_CLASSES": [
        "rest_framework.permissions.IsAuthenticated",
    ],
    "DEFAULT_RENDERER_CLASSES": [
        "rest_framework.renderers.JSONRenderer",
        "rest_framework.renderers.BrowsableAPIRenderer",
        "rest_framework.renderers.AdminRenderer",
    ],
    "DEFAULT_THROTTLE_CLASSES": [
        "rest_framework.throttling.UserRateThrottle",
        "rest_framework.throttling.ScopedRateThrottle",
    ],
    "DEFAULT_SCHEMA_CLASS": "drf_spectacular.openapi.AutoSchema",
    "DEFAULT_THROTTLE_RATES": {"user": "10/second", "auth": "5/minute"},
    "DEFAULT_PAGINATION_CLASS": "rest_framework.pagination.LimitOffsetPagination",
    "PAGE_SIZE": 10,
    "DEFAULT_FILTER_BACKENDS": ["django_filters.rest_framework.DjangoFilterBackend"],
    "EXCEPTION_HANDLER": "core.exceptions.django_error_handler",
}

# Celery Configurations
CELERY_BROKER_URL = "redis://127.0.0.1:6379/2"
CELERY_RESULT_BACKEND = "django-db"
CELERY_CACHE_BACKEND = "celery"
CELERY_RESULT_EXTENDED = True
CELERY_TASK_TRACK_STARTED = True

# Field Encryption
FIELD_ENCRYPTION_KEY = env("FIELD_ENCRYPTION_KEY")

# Django Rest Framework Spectacular Configurations
SPECTACULAR_SETTINGS = {
    "TITLE": "Monta API",
    "DESCRIPTION": "Transportation & Logistics Application backend written in Django! ",
    "VERSION": "1.0.0",
    "SERVE_INCLUDE_SCHEMA": False,
    "ENUM_NAME_OVERRIDES": {
        "LicenseStateEnum": "localflavor.us.us_states.STATE_CHOICES",
    },
}

# Django Email Backend
EMAIL_BACKEND = "django.core.mail.backends.console.EmailBackend"

# Silk Configurations
SILKY_PYTHON_PROFILER = True

