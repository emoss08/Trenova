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

from django.core import exceptions, validators

from edi import models


class EDICommProfileValidation:
    """Validates the communication profile for an EDI system.

    Args:
        comm_profile: The EDI communication profile to validate.

    Attributes:
        comm_profile: The EDI communication profile being validated.
        errors: A dictionary to store validation errors.

    Methods:
        validate_server_url: Validates the server URL based on the protocol.
        validate_protocol_mapping: Validates the protocol mapping for port and security expectations.
    """

    def __init__(self, *, comm_profile: models.EDICommProfile) -> None:
        """Initializes an instance of the EDICommProfileValidation class.

        Args:
            self: The instance of the EDICommProfileValidation class.
            comm_profile: The EDI communication profile to be validated.

        Returns:
            None: This function does not return anything.
        """
        self.comm_profile = comm_profile
        self.errors = {}

    def validate(self) -> None:
        """Validates the EDI communication profile.

        Args:
            self: The instance of the EDICommProfileValidation class.

        Returns:
            None: This function does not return anything.
        """
        self.validate_server_url()
        self.validate_protocol_mapping()

        if self.errors:
            raise exceptions.ValidationError(self.errors)

    def validate_server_url(self) -> None:
        """Validates the server URL based on the protocol of the EDI communication profile.

        Returns:
            None: This function does not return anything.

        Raises:
            KeyError: If the protocol of the communication profile is not found in the validators dictionary.
            ValidationError: If the server URL is invalid for the protocol of the communication profile.
        """
        protocol_validators = {
            "FTP": validators.URLValidator(schemes=["ftp"]),
            "SFTP": validators.URLValidator(schemes=["sftp"]),
            "AS2": validators.URLValidator(schemes=["http", "https"]),
            "HTTP": validators.URLValidator(schemes=["http", "https"]),
        }

        if validator := protocol_validators.get(self.comm_profile.protocol):
            try:
                validator(self.comm_profile.server_url)
            except exceptions.ValidationError:
                self.errors[
                    "server_url"
                ] = f"Invalid URL for protocol {self.comm_profile.protocol}: {self.comm_profile.server_url}"

    def validate_protocol_mapping(self) -> None:
        """Validates the protocol mapping for port and security expectations based on the EDI communication profile.

        Args:
            self: The instance of the EDICommProfileValidation class.

        Returns:
            None: This function does not return anything.
        """
        protocol_port_mapping = {
            "FTP": 21,
            "SFTP": 22,
            "AS2": 443,
            "HTTP": 80,
        }

        secure_protocols = ["AS2", "SFTP"]
        insecure_protocols = ["FTP", "HTTP"]

        # Check for protocol port consistency
        expected_port = protocol_port_mapping.get(self.comm_profile.protocol)
        if expected_port and self.comm_profile.port != expected_port:
            self.errors["port"] = (
                f"Invalid port for protocol {self.comm_profile.protocol}: {self.comm_profile.port}. Expected port "
                f"{expected_port}."
            )

        # Check for protocol security expectations
        if (
            self.comm_profile.protocol in secure_protocols
            and not self.comm_profile.is_secure
        ):
            self.errors[
                "is_secure"
            ] = f"Protocol {self.comm_profile.protocol} is expected to be secure."

        if (
            self.comm_profile.protocol in insecure_protocols
            and self.comm_profile.is_secure
        ):
            self.errors[
                "is_secure"
            ] = f"Protocol {self.comm_profile.protocol} cannot be secure."
