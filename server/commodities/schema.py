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

from commodities import models
from django_filters import FilterSet
from graphene import Field, ObjectType, Schema, relay
from graphene_django import DjangoObjectType
from graphene_django.filter import DjangoFilterConnectionField

if typing.TYPE_CHECKING:
    from django.db.models import QuerySet
    from graphql import GraphQLResolveInfo


class HazardousMaterialFilterSet(FilterSet):
    """
    FilterSet for HazardousMaterial
    """

    class Meta:
        model = models.HazardousMaterial
        fields = ["status", "name"]


class HazardousMaterialNode(DjangoObjectType):
    """
    Charge Type Node for GraphQL
    """

    class Meta:
        model = models.HazardousMaterial
        interfaces = (relay.Node,)
        filterset_class = HazardousMaterialFilterSet
        exclude = ["commodity_set"]

    @classmethod
    def get_queryset(
        cls, queryset: "QuerySet[models.HazardousMaterial]", info: "GraphQLResolveInfo"
    ) -> "QuerySet[models.HazardousMaterial]":
        return queryset.filter(organization_id=info.context.user.organization_id)


class CommodityFilterSet(FilterSet):
    """
    FilterSet for Commodity
    """

    class Meta:
        model = models.Commodity
        fields = ["status", "name"]


class CommodityNode(DjangoObjectType):
    """
    Charge Type Node for GraphQL
    """

    hazardous_material = Field(HazardousMaterialNode)

    class Meta:
        model = models.Commodity
        interfaces = (relay.Node,)
        filterset_class = CommodityFilterSet
        fields = "__all__"

    @classmethod
    def get_queryset(
        cls, queryset: "QuerySet[models.Commodity]", info: "GraphQLResolveInfo"
    ) -> "QuerySet[models.Commodity]":
        return queryset.filter(
            organization_id=info.context.user.organization_id
        ).select_related("hazardous_material")

    def resolve_hazardous_material(
        self, info: "GraphQLResolveInfo"
    ) -> models.HazardousMaterial:
        return self.hazardous_material


class Query(ObjectType):
    """
    The Query class defines the GraphQL queries that can be made to the server
    """

    hazardous_material = relay.Node.Field(HazardousMaterialNode)
    hazardous_materials = DjangoFilterConnectionField(HazardousMaterialNode)
    commodity = relay.Node.Field(CommodityNode)
    commodities = DjangoFilterConnectionField(CommodityNode)


schema = Schema(query=Query)
