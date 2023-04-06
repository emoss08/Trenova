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

import textwrap
import uuid
from typing import Any

from django.contrib.auth import password_validation
from django.contrib.auth.hashers import make_password
from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _

from utils.models import GenericModel


class KubeConfiguration(GenericModel):
    """
    Model that stores the kubectl configuration for related :model:`organization.Organization`.
    """

    id = models.UUIDField(
        _("ID"),
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    organization = models.OneToOneField(
        "organization.Organization",
        on_delete=models.CASCADE,
        related_name="kube_configuration",
        verbose_name=_("Organization"),
    )
    name = models.CharField(
        _("Name"),
        max_length=255,
        help_text=_("The name of the configuration."),
        default="default",
    )
    host = models.URLField(
        _("Host"),
        max_length=255,
        help_text=_("The host URL to use when communicating with the API server."),
        default="http://localhost",
    )
    api_key = models.CharField(
        _("API Key"),
        blank=True,
        max_length=255,
        help_text=_("The API key to use when authenticating with the API server."),
    )
    api_key_prefix = models.CharField(
        _("API Key Prefix"),
        blank=True,
        max_length=255,
        help_text=_(
            "The prefix for API key when used as HTTP header. The default is 'Bearer'."
        ),
    )
    refresh_api_key_hook = models.CharField(
        _("Refresh API Key Hook"),
        blank=True,
        max_length=255,
        help_text=_(
            "A function that receives the unmodified configuration and returns a new one with updated authentication information."
        ),
    )
    username = models.CharField(
        _("Username"),
        blank=True,
        max_length=255,
        help_text=_("The username for HTTP basic authentication."),
    )
    password = models.CharField(
        _("Password"),
        blank=True,
        max_length=128,
        help_text=_("The password for HTTP basic authentication."),
    )
    discard_unknown_keys = models.BooleanField(
        _("Discard Unknown Keys"),
        default=False,
        help_text=_(
            "If true, unknown properties will be discarded during deserialization."
        ),
    )
    logger = models.CharField(
        _("Logger"),
        blank=True,
        max_length=255,
        help_text=_("The logger to use for logging. The default is 'logging'."),
    )
    logger_format = models.CharField(
        _("Logger Format"),
        blank=True,
        max_length=255,
        help_text=_(
            "The format of the log messages. The default is '%(asctime)s %(levelname)s %(message)s'."
        ),
        default="%(asctime)s %(levelname)s %(message)s",
    )
    logger_stream_handler = models.CharField(
        _("Logger Stream Handler"),
        blank=True,
        max_length=255,
        help_text=_(
            "The stream handler to use for logging. The default is 'sys.stderr'."
        ),
        default="sys.stderr",
    )
    logger_file_handler = models.CharField(
        _("Logger File Handler"),
        blank=True,
        max_length=255,
        help_text=_(
            "The file handler to use for logging. The default is 'logging.FileHandler'."
        ),
        default="logging.FileHandler",
    )
    logger_file = models.CharField(
        _("Logger File"),
        blank=True,
        max_length=255,
        help_text=_("The file to use for logging. The default is 'kubernetes.log'."),
        default="kubernetes.log",
    )
    debug = models.BooleanField(
        _("Debug"),
        default=True,
        help_text=_("If true, will log additional debugging information."),
    )
    verify_ssl = models.BooleanField(
        _("Verify SSL"),
        default=False,
        help_text=_(
            "If true, the SSL certificates will be verified. A CA_BUNDLE path can also be provided."
        ),
    )
    ssl_ca_cert = models.CharField(
        _("SSL CA Cert"),
        blank=True,
        max_length=255,
        help_text=_(
            "A filename of the CA cert file to use in verifying the server's certificate."
        ),
    )
    key_file = models.CharField(
        _("Key File"),
        blank=True,
        max_length=255,
        help_text=_(
            "A filename of the client key file used to authenticate with the API server."
        ),
    )
    proxy = models.URLField(
        _("Proxy"),
        blank=True,
        max_length=255,
        help_text=_(
            "A proxy URL or proxy pre-formatted URL string to use during the HTTP request."
        ),
    )
    no_proxy = models.CharField(
        _("No Proxy"),
        blank=True,
        max_length=255,
        help_text=_(
            "A comma-separated list of hostnames and/or CIDRs for which the proxy should not be used."
        ),
    )
    proxy_headers = models.CharField(
        _("Proxy Headers"),
        blank=True,
        max_length=255,
        help_text=_(
            "Additional headers to send when using the proxy. The expected format is a dictionary with header name as key and header value as value."
        ),
    )
    safe_chars_for_path_param = models.CharField(
        _("Safe Chars For Path Param"),
        blank=True,
        max_length=255,
        help_text=_(
            "A list of safe characters for path parameter. The default is '/'."
        ),
        default="/",
    )
    retries = models.PositiveIntegerField(
        _("Retries"),
        default=5,
        help_text=_(
            "The number of retries each connection should attempt. The default is 5."
        ),
    )

    def __str__(self) -> str:
        """Kube Configuration string representation.

        Returns:
            str: String Representation of Kube Configuration.
        """
        return textwrap.shorten(self.name, width=30, placeholder="...")

    class Meta:
        """
        Metaclass for Kube Configuration.
        """

        verbose_name = _("Kube Configuration")
        verbose_name_plural = _("Kube Configurations")
        db_table = "kube_configuration"

    def save(self, **kwargs: Any) -> None:
        """
        Save the Kube Configuration.
        """
        super().save(**kwargs)
        if self.password:
            hashed_password = make_password(self.password)
            self.password = hashed_password

    def get_absolute_url(self) -> str:
        """Kube Configuration absolute URL.

        Returns:
            str: Absolute URL of Kube Configuration.
        """
        return reverse("kube_configuration_detail", kwargs={"pk": self.pk})
