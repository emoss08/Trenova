# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2024 MONTA                                                                         -
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

import typing

from django.db.models import Count
from django_filters import FilterSet
from graphene import Field, Int, ObjectType, Schema, relay
from graphene_django import DjangoObjectType
from graphene_django.filter import DjangoFilterConnectionField

from equipment import models

if typing.TYPE_CHECKING:
    from django.db.models import QuerySet
    from graphql import GraphQLResolveInfo

    from dispatch.models import FleetCode


class EquipmentTypeFilterSet(FilterSet):
    """
    FilterSet for EquipmentType
    """

    class Meta:
        model = models.EquipmentType
        fields = ["status", "equipment_class"]


class EquipmentTypeNode(DjangoObjectType):
    """
    DjangoObjectType for EquipmentType
    """

    class Meta:
        """
        Meta class defining the model and filter set for EquipmentTypeNode
        """

        model = models.EquipmentType
        filterset_class = EquipmentTypeFilterSet
        interfaces = (relay.Node,)
        exclude = ["tractor", "trailer"]

    @classmethod
    def get_queryset(
        cls, queryset: "QuerySet[models.EquipmentType]", info: "GraphQLResolveInfo"
    ) -> "QuerySet[models.EquipmentType]":
        return queryset.filter(organization_id=info.context.user.organization_id)


class TractorFilterSet(FilterSet):
    """
    FilterSet for Comment Type
    """

    class Meta:
        model = models.Tractor
        fields = ["status", "manufacturer"]


class TractorNode(DjangoObjectType):
    """
    DjangoObjectType for Tractor
    """

    equipment_type = Field(EquipmentTypeNode)
    fleet_code = Field("dispatch.schema.FleetCodeNode")

    class Meta:
        """
        Meta class defining the model and filter set for TractorNode
        """

        model = models.Tractor
        filterset_class = TractorFilterSet
        interfaces = (relay.Node,)
        fields = "__all__"

    @classmethod
    def get_queryset(
        cls, queryset: "QuerySet[models.Tractor]", info: "GraphQLResolveInfo"
    ) -> "QuerySet[models.Tractor]":
        return queryset.filter(
            organization_id=info.context.user.organization_id
        ).select_related(
            "equipment_type",
            "fleet_code",
            "fleet_code__manager",
            "fleet_code__manager__profiles",
            "fleet_code__manager__profiles__job_title",
        )

    def resolve_equipment_type(
        self, info: "GraphQLResolveInfo"
    ) -> models.EquipmentType:
        return self.equipment_type

    def resolve_fleet_code(self, info: "GraphQLResolveInfo") -> "FleetCode":
        return self.fleet_code


class TrailerFilterSet(FilterSet):
    """
    FilterSet for Comment Type
    """

    class Meta:
        model = models.Trailer
        fields = ["status", "is_leased"]


class TrailerNode(DjangoObjectType):
    """
    DjangoObjectType for Trailer
    """

    times_used = Int()

    class Meta:
        """
        Meta class defining the model and filter set for TrailerNode
        """

        model = models.Trailer
        filterset_class = TrailerFilterSet
        interfaces = (relay.Node,)
        fields = "__all__"

    @classmethod
    def get_queryset(
        cls, queryset: "QuerySet[models.Trailer]", info: "GraphQLResolveInfo"
    ) -> "QuerySet[models.Trailer]":
        return (
            queryset.filter(organization_id=info.context.user.organization_id)
            .select_related("equipment_type", "fleet_code")
            .annotate(
                times_used=Count("movement", distinct=True),
            )
        )


class Query(ObjectType):
    """
    The Query class defines the GraphQL queries that can be made to the server
    """

    equipment_type = relay.Node.Field(EquipmentTypeNode)
    equipment_types = DjangoFilterConnectionField(EquipmentTypeNode)
    tractor = relay.Node.Field(TractorNode)
    tractors = DjangoFilterConnectionField(TractorNode)
    trailer = relay.Node.Field(TrailerNode)
    trailers = DjangoFilterConnectionField(TrailerNode)


schema = Schema(query=Query)
