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

from auditlog.models import LogEntry
from rest_framework import serializers

from reports import models
from utils.serializers import GenericSerializer


class TableColumnSerializer(serializers.Serializer):
    """
    A serializer for the `TableColumn` model.

    This serializer converts instances of the `TableColumn` model into JSON or other data formats,
    and vice versa. It uses the specified fields (name, verbose_name) to create the serialized
    representation of the `TableColumn` model.

    Attributes:
        name (str): The name of the column.
        verbose_name (str): The verbose name of the column.
    """

    name = serializers.CharField()
    verbose_name = serializers.CharField()


class CustomReportSerializer(GenericSerializer):
    """
    A serializer for the `CustomReport` model.

    This serializer converts instances of the `CustomReport` model into JSON or other data formats,
    and vice versa. It uses the specified fields (name, description, and code) to create the serialized
    representation of the `CustomReport` model.

    See Also:
        `GenericSerializer`
    """

    class Meta:
        """
        A class representing the metadata for the `CustomReportSerializer` class.
        """

        model = models.CustomReport


class UserReportSerializer(GenericSerializer):
    """A serializer for the `UserReport` model.

    This serializer converts instances of the `UserReport` model into JSON or other data formats,
    and vice versa. It uses the specified fields (name, description, and code) to create the serialized
    representation of the `UserReport` model.

    See Also:
        `GenericSerializer`
    """

    file_name = serializers.SerializerMethodField()

    class Meta:
        """
        A class representing the metadata for the `UserReportSerializer` class.
        """

        model = models.UserReport
        fields = (
            "id",
            "organization",
            "user",
            "report",
            "created",
            "modified",
            "file_name",
        )

    def get_file_name(self, instance: models.UserReport) -> str:
        """Extracts the name of the file from the report attribute of the instance.

        Args:
            instance (models.UserReport): The `UserReport` model instance that will be serialized.

        Returns:
            str: The name of the file from the report attribute of the instance.
        """

        return os.path.basename(instance.report.name)


class LogEntrySerializer(serializers.ModelSerializer):
    actor = serializers.CharField(
        source="actor.username", read_only=True, allow_null=True
    )

    class Meta:
        """Metaclass for LogEntrySerializer

        Attributes:
            model (models.LogEntry): The model that the serializer.
        """

        model = LogEntry
        fields = (
            "content_type",
            "object_pk",
            "object_id",
            "object_repr",
            "serialized_data",
            "action",
            "changes",
            "actor",
            "remote_addr",
            "timestamp",
            "additional_data",
        )
