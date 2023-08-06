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


from stops import models
from utils.serializers import GenericSerializer


class QualifierCodeSerializer(GenericSerializer):
    """A serializer for the `QualifierCode` model.

    A serializer class for the QualifierCode Model. This serializer is used
    to convert the QualifierCode model instances into a Python dictionary
    format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model.
    """

    class Meta:
        """Metaclass for the `QualifierCodeSerializer` class

        Attributes:
            model (models.QualifierCode): The model that the serializer is for.
        """

        model = models.QualifierCode


class StopCommentSerializer(GenericSerializer):
    """A serializer for the `StopComment` model.

    A serializer class for the StopComment Model. This serializer is used
    to convert the StopComment model instances into a Python dictionary
    format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model.
    """

    class Meta:
        """Metaclass for the `StopCommentSerializer` class

        Attributes:
            model (models.Order): The model that the serializer is for.
            extra_fields (tuple): A tuple of extra fields that should be included
            in the serialized representation of the model.
        """

        model = models.StopComment


class StopSerializer(GenericSerializer):
    """A serializer for the `Stop` model.

    A serializer class for the Stop Model. This serializer is used to convert the Stop model instances
    into a Python dictionary format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model.
    """

    class Meta:
        """Metaclass for the `StopSerializer` class

        Attributes:
            model (models.Stop): The model that the serializer is for.
        """

        model = models.Stop


class ServiceIncidentSerializer(GenericSerializer):
    """A serializer for the `ServiceIncident` model.

    A serializer class for the ServiceIncident Model. This serializer is used to convert the ServiceIncident model instances
    into a Python dictionary format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model.
    """

    class Meta:
        """Metaclass for the `ServiceIncidentSerializer` class

        Attributes:
            model (models.ServiceIncident): The model that the serializer is for.
            extra_fields (tuple): A tuple of extra fields that should be included
            in the serialized representation of the model.
        """

        model = models.ServiceIncident