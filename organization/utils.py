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

from email.mime.multipart import MIMEMultipart
from email.mime.text import MIMEText
from email.utils import formatdate
from smtplib import SMTP, SMTP_SSL

from organization import exceptions, models


def send_email_using_profile(
    *, profile: models.EmailProfile, subject: str, content: str, recipients: str
) -> None:
    smtp_class: type[SMTP] | type[SMTP_SSL]

    msg = MIMEMultipart()
    msg["From"] = profile.email
    msg["To"] = ", ".join(recipients)
    msg["Date"] = formatdate(localtime=True)
    msg["Subject"] = subject

    msg.attach(MIMEText(content))

    if profile.protocol == models.EmailProfile.EmailProtocolChoices.SSL:
        smtp_class = SMTP_SSL
    elif profile.protocol in [
        models.EmailProfile.EmailProtocolChoices.TLS,
        models.EmailProfile.EmailProtocolChoices.UNENCRYPTED,
    ]:
        smtp_class = SMTP
    else:
        raise exceptions.InvalidEmailProtocal("Invalid email protocol")

    if not profile.port:
        raise exceptions.InvalidEmailProfile("Port is required.")

    with smtp_class(profile.host, profile.port) as smtp:
        if profile.protocol == models.EmailProfile.EmailProtocolChoices.TLS:
            smtp.starttls()

        smtp.login(profile.username, profile.password)
        smtp.sendmail(profile.email, recipients, msg.as_string())
