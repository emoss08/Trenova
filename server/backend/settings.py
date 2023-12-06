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
INTERNAL_IPS = ["127.0.0.1", "monta.local", "localhost"]
ALLOWED_HOSTS = ["monta.local", "127.0.0.1", "localhost"]

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
    "encrypted_model_fields",
    "drf_spectacular",
    "auditlog",
    "notifications",
    "channels",
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
    "shipment",
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
    "edi",
    "document_generator",
]

# Middleware configurations
MIDDLEWARE = [
    "django.middleware.security.SecurityMiddleware",
    "django.contrib.sessions.middleware.SessionMiddleware",
    "corsheaders.middleware.CorsMiddleware",
    "whitenoise.middleware.WhiteNoiseMiddleware",
    "django.middleware.common.CommonMiddleware",
    "django.middleware.csrf.CsrfViewMiddleware",
    "django.contrib.auth.middleware.AuthenticationMiddleware",
    "django.contrib.messages.middleware.MessageMiddleware",
    "django.middleware.clickjacking.XFrameOptionsMiddleware",
    "auditlog.middleware.AuditlogMiddleware",
    # "core.middleware.logging_middleware.RocketLoggingMiddleware",
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
    },
    # "replica1": {
    #     "ENGINE": "django.db.backends.postgresql",
    #     "NAME": env("DB_REPLICA_NAME"),
    #     "USER": env("DB_REPLICA_USER"),
    #     "PASSWORD": env("DB_REPLICA_PASSWORD"),
    #     "HOST": env("DB_REPLICA_HOST"),
    #     "PORT": env("DB_REPLICA_PORT"),
    #     "ATOMIC_REQUESTS": True,
    #     "CONN_HEALTH_CHECK": True,
    # },
}
# DATABASE_ROUTERS = ["core.database.PrimaryReplicaRouter"]

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
AUTHENTICATION_BACKENDS = [
    "django.contrib.auth.backends.ModelBackend",
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
            "PREFIX": "celery",
            "PARSER_CLASS": "redis.connection.HiredisParser",
            "CONNECTION_POOL_KWARGS": {
                "max_connections": 100,
                "retry_on_timeout": True,
            },
        },
    },
    "shipments": {
        "BACKEND": "django_redis.cache.RedisCache",
        "LOCATION": "redis://127.0.0.1:6379/3",
        "OPTIONS": {
            "CLIENT_CLASS": "django_redis.client.DefaultClient",
            "PREFIX": "shipments",
            "PARSER_CLASS": "redis.connection.HiredisParser",
            "CONNECTION_POOL_KWARGS": {
                "max_connections": 100,
                "retry_on_timeout": True,
            },
        },
    },
}

# Session Configurations
# SESSION_ENGINE = "django.contrib.sessions.backends.cached_db"

# Cors Configurations
CORS_ALLOWED_ORIGINS = [
    "http://localhost:5173",
    "http://127.0.0.1:5173",
    "http://localhost:3000",
    "http://127.0.0.1:3000",
]

CORS_ALLOW_CREDENTIALS = True

# CSRF Configurations
CSRF_TRUSTED_ORIGINS = [
    "http://localhost:5173",
]
CSRF_COOKIE_SECURE = False
CSRF_COOKIE_HTTPONLY = False

# Rest Framework Configurations
REST_FRAMEWORK = {
    "DEFAULT_AUTHENTICATION_CLASSES": [
        "accounts.authentication.BearerTokenAuthentication",
        "rest_framework.authentication.BasicAuthentication",
        "rest_framework.authentication.SessionAuthentication",
    ],
    "DEFAULT_PERMISSION_CLASSES": [
        "rest_framework.permissions.IsAuthenticated",
    ],
    "DEFAULT_RENDERER_CLASSES": [
        "djangorestframework_camel_case.render.CamelCaseJSONRenderer",
        "djangorestframework_camel_case.render.CamelCaseBrowsableAPIRenderer",
        "rest_framework.renderers.JSONRenderer",
        "rest_framework.renderers.BrowsableAPIRenderer",
        "rest_framework.renderers.AdminRenderer",
    ],
    "DEFAULT_THROTTLE_CLASSES": [
        "rest_framework.throttling.UserRateThrottle",
        "rest_framework.throttling.ScopedRateThrottle",
    ],
    "DEFAULT_SCHEMA_CLASS": "drf_spectacular.openapi.AutoSchema",
    "DEFAULT_THROTTLE_RATES": {
        "user": "100/minute",
        "auth": "20/minute",
    },
    "DEFAULT_PAGINATION_CLASS": "utils.pagination.MontaPagination",
    "PAGE_SIZE": 10,
    "DEFAULT_FILTER_BACKENDS": [
        "django_filters.rest_framework.DjangoFilterBackend",
        "rest_framework.filters.SearchFilter",
        "rest_framework.filters.OrderingFilter",
    ],
    "DEFAULT_PARSER_CLASSES": (
        "djangorestframework_camel_case.parser.CamelCaseFormParser",
        "djangorestframework_camel_case.parser.CamelCaseMultiPartParser",
        "djangorestframework_camel_case.parser.CamelCaseJSONParser",
    ),
    "EXCEPTION_HANDLER": "drf_standardized_errors.handler.exception_handler",
    "DEFAULT_VERSIONING_CLASS": "rest_framework.versioning.URLPathVersioning",
}

DRF_STANDARDIZED_ERRORS = {
    "EXCEPTION_HANDLER_CLASS": "core.exceptions.CustomExceptionHandler",
    "EXCEPTION_FORMATTER_CLASS": "core.exceptions.CustomExceptionFormatter",
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
    "VERSION": "0.1.0",
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

# Cache Ops settings
CACHEOPS = {
    "shipment.shipmentcontrol": {"ops": "all"},
    "invoicing.invoicecontrol": {"ops": "all"},
    "route.routecontrol": {"ops": "all"},
    "billing.billingcontrol": {"ops": "all"},
    "dispatch.dispatchcontrol": {"ops": "all"},
    "organization.emailcontrol": {"ops": "all"},
    "organization.organization": {"ops": "all"},
    "organization.department": {"ops": "all"},
    "accounting.generalledgeraccount": {"ops": "all"},
    "customer.customer": {"ops": "all"},
    "location.states": {"ops": "all"},
}
CACHEOPS_DEGRADE_ON_FAILURE = True

# Billing Client Configurations
BILLING_CLIENT_PASSWORD = env("BILLING_CLIENT_REDIS_PASSWORD")
BILLING_CLIENT_HOST = env("BILLING_CLIENT_REDIS_HOST")
BILLING_CLIENT_PORT = env("BILLING_CLIENT_REDIS_PORT")
BILLING_CLIENT_DB = env("BILLING_CLIENT_REDIS_DB")

# Audit Log Configurations
AUDITLOG_EXCLUDE_TRACKING_FIELDS = ("organization", "business_unit")
AUDITLOG_INCLUDE_TRACKING_MODELS = (
    "accounts.User",
    "accounts.UserProfile",
    "accounts.JobTitle",
    "accounts.Token",
    "accounting.GeneralLedgerAccount",
    "accounting.RevenueCode",
    "accounting.DivisionCode",
    "billing.BillingControl",
    "billing.ChargeType",
    "billing.AccessorialCharge",
    "billing.DocumentClassification",
    "billing.BillingQueue",
    "billing.BillingLogEntry",
    "billing.BillingHistory",
    "billing.BillingException",
    "customer.Customer",
    "customer.CustomerEmailProfile",
    "customer.CustomerRuleProfile",
    "customer.CustomerContact",
    "customer.CustomerFuelProfile",
    "customer.CustomerFuelTable",
    "customer.CustomerFuelTableDetail",
    "customer.DeliverySlot",
    "shipment.ShipmentControl",
    "shipment.ShipmentType",
    "shipment.Shipment",
    "shipment.ShipmentDocumentation",
    "shipment.ShipmentComment",
    "shipment.AdditionalCharge",
    "shipment.ReasonCode",
    "shipment.FormulaTemplate",
)

# Protect against clickjacking by setting X-Frame-Options header
X_FRAME_OPTIONS = "DENY"

# Kafka Configurations
KAFKA_BOOTSTRAP_SERVERS = env("KAFKA_BOOTSTRAP_SERVERS")
KAFKA_HOST = env("KAFKA_HOST")
KAFKA_PORT = env("KAFKA_PORT")
KAFKA_GROUP_ID = env("KAFKA_GROUP_ID")
KAFKA_THREAD_POOL_SIZE = 10
KAFKA_MAX_CONCURRENT_JOBS = 100
KAFKA_ALERT_UPDATE_TOPIC = env("KAFKA_ALERT_UPDATE_TOPIC")
KAFKA_AUTO_COMMIT = env("KAFKA_AUTO_COMMIT")
KAFKA_AUTO_COMMIT_INTERVAL_MS = env("KAFKA_AUTO_COMMIT_INTERVAL_MS")
KAFKA_AUTO_OFFSET_RESET = env("KAFKA_OFFSET_RESET")

# Development Configurations
if DEBUG:
    INSTALLED_APPS.insert(0, "admin_interface")
    INSTALLED_APPS.insert(1, "colorfield")
    INSTALLED_APPS += ["silk"]
    MIDDLEWARE += [
        "silk.middleware.SilkyMiddleware",
    ]
    X_FRAME_OPTIONS = "SAMEORIGIN"
    SILK_PYTHON_PROFILER = True
