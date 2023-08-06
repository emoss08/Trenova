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

from collections.abc import Sequence
from smtplib import SMTPException

from django.core.mail.backends.base import BaseEmailBackend
from django.core.mail.message import EmailMessage
from django.db import transaction

from organization.models import EmailLog


class DatabaseEmailBackend(BaseEmailBackend):
    """
    A custom email backend that logs email failures to a database.
    """

    def send_messages(self, email_messages: Sequence[EmailMessage]) -> int:
        """
        Sends the provided list of email messages.

        If sending fails for a message, it creates a new `EmailLog` instance with the
        subject, recipient email address, and error message and saves it to the database.

        The database operations are wrapped in a transaction to ensure atomicity.

        Args:
            email_messages: A list of `EmailMessage` objects to send.

        Returns:
            The number of sent messages.
        """

        num_sent = 0
        with transaction.atomic():
            for message in email_messages:
                try:
                    message.send()
                    num_sent += 1
                except SMTPException as email_error:
                    email_log = EmailLog(
                        subject=message.subject,
                        to_email=", ".join(message.to),
                        error=str(email_error),
                    )
                    email_log.save()
        return num_sent
