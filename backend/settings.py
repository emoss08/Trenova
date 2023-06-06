# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2023 MONTA                                                                         -
#                                                                                                  -
#  This file is part of Monta.                                                                     -
#                                                                                                  -
#  The Monta software is licensed under the Business Source License 1.1. You are granted the right -
#  to copy, modify, and redistribute the software, but only for non-production use or with a total -
#  of less than three server instances. Starting from the Change Date (November 16, 2026), the     -
#  software will be made available under version 2 or later of the GNU General Public License.     -
#  If you use the software in violation of this license, your rights under the license will be     -
#  terminated automatically. The software is provided "as is," and the Licensor disclaims all      -
#  warranties and conditions. If you use this license's text or the "Business Source License" name -
#  and trademark, you must comply with the Licensor's covenants, which include specifying the      -
#  Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use     -
#  Grant, and not modifying the license in any other way.                                          -
# --------------------------------------------------------------------------------------------------

import os
import sys
from pathlib import Path

import django_stubs_ext
import environ

# Check if running on pypy. If so, monkey patch psycopg2cffiq
if sys.implementation.name == "pypy":
    from psycopg2cffi import compat

    compat.register()

django_stubs_ext.monkeypatch()

env = environ.Env(DEBUG=(bool, False))

# Build paths inside the project like this: BASE_DIR / 'subdir'.
BASE_DIR = Path(__file__).resolve().parent.parent
environ.Env.read_env(os.path.join(BASE_DIR, ".env"))
SECRET_KEY = env("SECRET_KEY")
DEBUG = env("DEBUG")
CORS_ORIGIN_ALLOW_ALL = True
INTERNAL_IPS = ["127.0.0.1", "monta.local", "localhost"]
ALLOWED_HOSTS = ["monta.local", "127.0.0.1", "localhost"]

# Sentry Configuration
# sentry_sdk.init(
#     dsn=env("SENTRY_DSN"),
#     integrations=[
#         DjangoIntegration(),
#     ],
#     traces_sample_rate=1.0,
#     send_default_pii=True,
#     # To set a uniform sample rate
#     # Set profiles_sample_rate to 1.0 to profile 100%
#     # of sampled transactions.
#     # We recommend adjusting this value in production
#     profiles_sample_rate=1.0,
# )

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
    "drf_standardized_errors",
    "django_extensions",
    "localflavor",
    "cacheops",
    "rest_framework",
    "corsheaders",
    "django_filters",
    "phonenumber_field",
    "django_celery_results",
    "django_celery_beat",
    "silk",
    "encrypted_model_fields",
    "nested_inline",
    "drf_spectacular",
    "djmoney",
    # "notifications",
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
    "reports",
    "plugin",
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

# Channels
ASGI_APPLICATION = "backend.asgi.application"
CHANNEL_LAYERS = {
    "default": {
        "BACKEND": "channels_redis.core.RedisChannelLayer",
        "CONFIG": {
            "hosts": [("127.0.0.1", 6379)],
        },
    },
}

# Databases
DATABASES = {
    "default": {
        "ENGINE": "django.db.backends.postgresql",
        "NAME": env("DB_NAME"),
        "USER": env("DB_USER"),
        "PASSWORD": env("DB_PASSWORD"),
        "HOST": env("DB_HOST"),
        "PORT": env("DB_PORT"),
        "ATOMIC_REQUESTS": True,
        "CONN_HEALTH_CHECK": True,
    }
}

# Internationalization
LANGUAGE_CODE = "en-us"
TIME_ZONE = "US/Eastern"
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
# SESSION_CACHE_ALIAS = "sessions"
# SESSION_COOKIE_SECURE = True
# SESSION_COOKIE_HTTPONLY = True
# SESSION_COOKIE_SAMESITE = "Strict"
# SESSION_EXPIRE_AT_BROWSER_CLOSE = True

# CSRF Configurations
# CSRF_COOKIE_SECURE = True
# CSRF_COOKIE_HTTPONLY = True
# Security Configurations
# SECURE_HSTS_SECONDS = 31536000
# SECURE_SSL_REDIRECT = True
# SECURE_HSTS_PRELOAD = True
# SECURE_HSTS_INCLUDE_SUBDOMAINS = True
# SECURE_BROWSER_XSS_FILTER = True
# SECURE_CONTENT_TYPE_NOSNIFF = True
# X_FRAME_OPTIONS = "DENY"

#
# LOGGING = {
#     "version": 1,
#     "disable_existing_loggers": False,
#     "handlers": {
#         "kafka_file": {
#             "level": "DEBUG",
#             "class": "logging.handlers.TimedRotatingFileHandler",
#             "filename": "logs/kafka/listener.log",
#             "when": "D",
#             "interval": 1,  # 1 day
#             "backupCount": 30,
#         },
#     },
#     "loggers": {
#         "kafka": {
#             "handlers": ["kafka_file"],
#             "level": "DEBUG",
#             "propagate": True,
#         },
#     },
# }

# Rest Framework Configurations
REST_FRAMEWORK = {
    "DEFAULT_AUTHENTICATION_CLASSES": [
        "accounts.authentication.BearerTokenAuthentication",
        "rest_framework.authentication.BasicAuthentication",
        "rest_framework.authentication.SessionAuthentication",
    ],
    "DEFAULT_PERMISSION_CLASSES": [
        "rest_framework.permissions.IsAuthenticated",
        # "rest_framework.permissions.DjangoObjectPermissions",
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
    "DEFAULT_FILTER_BACKENDS": [
        "django_filters.rest_framework.DjangoFilterBackend",
        "rest_framework.filters.SearchFilter",
        "rest_framework.filters.OrderingFilter",
    ],
    "EXCEPTION_HANDLER": "drf_standardized_errors.handler.exception_handler",
    "DEFAULT_VERSIONING_CLASS": "rest_framework.versioning.URLPathVersioning",
}

# Celery Configurations
CELERY_BROKER_URL = "redis://127.0.0.1:6379/2"
CELERY_RESULT_BACKEND = "django-db"
CELERY_CACHE_BACKEND = "celery"
CELERY_RESULT_EXTENDED = True
CELERY_TASK_TRACK_STARTED = True
CELERY_BEAT_SCHEDULER = "django_celery_beat.schedulers:DatabaseScheduler"

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

# Cacheops configurations
CACHEOPS_REDIS = env("CACHEOPS_REDIS_LOCATION")

CACHEOPS_DEFAULTS = {
    "timeout": 60 * 60,
}

CACHEOPS = {
    "order.ordercontrol": {"ops": "all"},
    "invoicing.invoicecontrol": {"ops": "all"},
    "route.routecontrol": {"ops": "all"},
    "billing.billingcontrol": {"ops": "all"},
    "dispatch.dispatchcontrol": {"ops": "all"},
    "organization.emailcontrol": {"ops": "all"},
}
CACHEOPS_DEGRADE_ON_FAILURE = True

SILKY_PYTHON_PROFILER = True
