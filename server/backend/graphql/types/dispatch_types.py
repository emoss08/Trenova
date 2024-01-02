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

import graphene
from django_filters import FilterSet
from graphene_django import DjangoObjectType

from dispatch import models

if typing.TYPE_CHECKING:
    from django.db.models import QuerySet
    from graphql import GraphQLResolveInfo


class DelayCodeFilterSet(FilterSet):
    """
    FilterSet for DelayCode
    """

    class Meta:
        model = models.DelayCode
        fields = ["status", "code", "description", "f_carrier_or_driver"]


class DelayCodeNode(DjangoObjectType):
    """
    DjangoObjectType for DelayCode
    """

    class Meta:
        """
        Meta class defining the model and filter set for DelayCodeNode
        """

        model = models.DelayCode
        filterset_class = DelayCodeFilterSet
        interfaces = (graphene.relay.Node,)
        fields = "__all__"

    @classmethod
    def get_queryset(
        cls, queryset: "QuerySet[models.DelayCode]", info: "GraphQLResolveInfo"
    ) -> "QuerySet[models.DelayCode]":
        return queryset.filter(organization=info.context.user.organization)
