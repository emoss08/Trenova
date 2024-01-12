# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2024 Trenova                                                                       -
#                                                                                                  -
#  This file is part of Trenova.                                                                   -
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

import typing

from django_filters import FilterSet
from graphene import Field, ObjectType, Schema, relay
from graphene_django import DjangoObjectType
from graphene_django.filter import DjangoFilterConnectionField

from shipment import models

if typing.TYPE_CHECKING:
    from django.db.models import QuerySet
    from graphql import GraphQLResolveInfo


class ShipmentControlNode(DjangoObjectType):
    """Shipment Control Node for GraphQL.

    Notes:
        - This is only available to users with the `shipment.view_shipmentcontrol` permission.
        - Available by default to anyone that is an admin.
    """

    class Meta:
        model = models.ShipmentControl
        interfaces = (relay.Node,)
        fields = "__all__"

    @staticmethod
    def has_read_permission(*, info: "GraphQLResolveInfo") -> bool:
        user = info.context.user
        return user.is_superuser or user.has_perm("shipment.view_shipmentcontrol")


class ShipmentTypeFilterSet(FilterSet):
    """
    FilterSet for ShipmentType
    """

    class Meta:
        model = models.ShipmentType
        fields = ["status", "code"]


class ShipmentTypeNode(DjangoObjectType):
    class Meta:
        model = models.ShipmentType
        filterset_class = ShipmentTypeFilterSet
        interfaces = (relay.Node,)
        fields = "__all__"

    @classmethod
    def get_queryset(
        cls, queryset: "QuerySet[models.ShipmentType]", info: "GraphQLResolveInfo"
    ) -> "QuerySet[models.ShipmentType]":
        return queryset.filter(organization_id=info.context.user.organization_id)


class ReasonCodeFilterSet(FilterSet):
    """
    FilterSet for ReasonCode
    """

    class Meta:
        model = models.ReasonCode
        fields = ["status", "code", "description"]


class ReasonCodeNode(DjangoObjectType):
    """
    DjangoObjectType for ReasonCode
    """

    class Meta:
        """
        Meta class defining the model and filter set for ReasonCodeNode
        """

        model = models.ReasonCode
        filterset_class = ReasonCodeFilterSet
        interfaces = (relay.Node,)
        fields = "__all__"

    @classmethod
    def get_queryset(
        cls, queryset: "QuerySet[models.ReasonCode]", info: "GraphQLResolveInfo"
    ) -> "QuerySet[models.ReasonCode]":
        return queryset.filter(organization_id=info.context.user.organization_id)


class AdditionalChargeFilterSet(FilterSet):
    """
    FilterSet for AdditionalCharge
    """

    class Meta:
        model = models.AdditionalCharge
        fields = ["accessorial_charge", "entered_by"]


class AdditionalChargeNode(DjangoObjectType):
    """
    DjangoObjectType for AdditionalCharge
    """

    class Meta:
        """
        Meta class defining the model and filter set for AdditionalChargeNode
        """

        model = models.AdditionalCharge
        filterset_class = AdditionalChargeFilterSet
        interfaces = (relay.Node,)
        fields = "__all__"

    @classmethod
    def get_queryset(
        cls, queryset: "QuerySet[models.AdditionalCharge]", info: "GraphQLResolveInfo"
    ) -> "QuerySet[models.AdditionalCharge]":
        return queryset.filter(organization_id=info.context.user.organization_id)


class ServiceTypeFilterSet(FilterSet):
    """
    FilterSet for ServiceType
    """

    class Meta:
        model = models.ServiceType
        fields = ["status", "code"]


class ServiceTypeNode(DjangoObjectType):
    """
    ServiceType Node for GraphQL.
    """

    class Meta:
        """
        Meta class defining the model and filter set for ServiceTypeNode
        """

        model = models.ServiceType
        filterset_class = ServiceTypeFilterSet
        interfaces = (relay.Node,)
        fields = "__all__"

    @classmethod
    def get_queryset(
        cls, queryset: "QuerySet[models.ServiceType]", info: "GraphQLResolveInfo"
    ) -> "QuerySet[models.ServiceType]":
        return queryset.filter(organization_id=info.context.user.organization_id)


# TODO(WOLFRED): Add the shipment schema.


class Query(ObjectType):
    """
    The Query class defines the GraphQL queries that can be made to the server
    """

    shipment_type = relay.Node.Field(ShipmentTypeNode)
    shipment_types = DjangoFilterConnectionField(ShipmentTypeNode)
    shipment_control = Field(ShipmentControlNode)
    reason_code = relay.Node.Field(ReasonCodeNode)
    reason_codes = DjangoFilterConnectionField(ReasonCodeNode)
    additional_charge = relay.Node.Field(AdditionalChargeNode)
    additional_charges = DjangoFilterConnectionField(AdditionalChargeNode)
    service_type = relay.Node.Field(ServiceTypeNode)
    service_types = DjangoFilterConnectionField(ServiceTypeNode)

    def resolve_shipment_control(
        self, info: "GraphQLResolveInfo", **kwargs: typing.Any
    ) -> models.ShipmentControl:
        if not ShipmentControlNode.has_read_permission(info=info):
            raise PermissionError("You do not have permission to view ShipmentControl")

        return models.ShipmentControl.objects.get(
            organization_id=info.context.user.organization_id
        )


schema = Schema(query=Query)
