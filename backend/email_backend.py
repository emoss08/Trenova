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

from typing import Sequence
from django.core.mail.backends.base import BaseEmailBackend
from django.core.mail.message import EmailMessage
from django.db import transaction

from smtplib import SMTPException

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
                        error=str(email_error)
                    )
                    email_log.save()
        return num_sent
