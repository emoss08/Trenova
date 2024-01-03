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

from django.core.exceptions import PermissionDenied
from django_filters import FilterSet
from graphene import Field, ObjectType, Schema, relay
from graphene_django import DjangoObjectType
from graphene_django.filter import DjangoFilterConnectionField

from dispatch import models

if typing.TYPE_CHECKING:
    from django.db.models import QuerySet
    from graphql import GraphQLResolveInfo

    from accounts.models import User


class DispatchControlNode(DjangoObjectType):
    """Dispatch Control Node for GraphQL.

    Notes:
        - This is only available to users with the `dispatch.view_dispatchcontrol` permission.
        - Available by default to anyone that is an admin.
    """

    class Meta:
        model = models.DispatchControl
        interfaces = (relay.Node,)
        fields = "__all__"

    @staticmethod
    def has_read_permission(*, info: "GraphQLResolveInfo") -> bool:
        user = info.context.user
        return user.is_superuser or user.has_perm("dispatch.view_dispatchcontrol")


class FeasibilityToolControlNode(DjangoObjectType):
    """Dispatch Control Node for GraphQL.

    Notes:
        - This is only available to users with the `dispatch.view_feasibilitytoolcontrol` permission.
        - Available by default to anyone that is an admin.
    """

    class Meta:
        model = models.FeasibilityToolControl
        interfaces = (relay.Node,)
        fields = "__all__"

    @staticmethod
    def has_read_permission(*, info: "GraphQLResolveInfo") -> bool:
        user = info.context.user
        return user.is_superuser or user.has_perm(
            "dispatch.view_feasibilitytoolcontrol"
        )


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
        interfaces = (relay.Node,)
        fields = "__all__"

    @classmethod
    def get_queryset(
        cls, queryset: "QuerySet[models.DelayCode]", info: "GraphQLResolveInfo"
    ) -> "QuerySet[models.DelayCode]":
        return queryset.filter(organization_id=info.context.user.organization_id)


class CommentTypeFilterSet(FilterSet):
    """
    FilterSet for Comment Type
    """

    class Meta:
        model = models.CommentType
        fields = ["status"]


class CommentTypeNode(DjangoObjectType):
    """
    DjangoObjectType for CommentType
    """

    class Meta:
        """
        Meta class defining the model and filter set for CommentTypeNode
        """

        model = models.CommentType
        filterset_class = CommentTypeFilterSet
        interfaces = (relay.Node,)
        fields = "__all__"

    @classmethod
    def get_queryset(
        cls, queryset: "QuerySet[models.CommentType]", info: "GraphQLResolveInfo"
    ) -> "QuerySet[models.CommentType]":
        return queryset.filter(organization_id=info.context.user.organization_id)


class FleetCodeFilterSet(FilterSet):
    """
    FilterSet for Comment Type
    """

    class Meta:
        model = models.FleetCode
        fields = ["status"]


class FleetCodeNode(DjangoObjectType):
    """
    DjangoObjectType for FleetCode
    """

    manager = Field("accounts.schema.UserNode")

    class Meta:
        """
        Meta class defining the model and filter set for FleetCodeNode
        """

        model = models.FleetCode
        filterset_class = FleetCodeFilterSet
        interfaces = (relay.Node,)
        exclude = ["tractor", "trailer"]

    @classmethod
    def get_queryset(
        cls, queryset: "QuerySet[models.FleetCode]", info: "GraphQLResolveInfo"
    ) -> "QuerySet[models.FleetCode]":
        return queryset.filter(
            organization_id=info.context.user.organization_id
        ).select_related("manager", "manager__profiles", "manager__profiles__job_title")

    def resolve_manager(self, info: "GraphQLResolveInfo") -> "User":
        return self.manager


class Query(ObjectType):
    """
    The Query class defines the GraphQL queries that can be made to the server
    """

    dispatch_control = Field(DispatchControlNode)
    feasibility_tool_control = Field(FeasibilityToolControlNode)
    delay_code = relay.Node.Field(DelayCodeNode)
    delay_codes = DjangoFilterConnectionField(DelayCodeNode)
    comment_type = relay.Node.Field(CommentTypeNode)
    comment_types = DjangoFilterConnectionField(CommentTypeNode)
    fleet_code = relay.Node.Field(FleetCodeNode)
    fleet_codes = DjangoFilterConnectionField(FleetCodeNode)

    def resolve_dispatch_control(
        self, info: "GraphQLResolveInfo", **kwargs: typing.Any
    ) -> models.DispatchControl:
        if not DispatchControlNode.has_read_permission(info=info):
            raise PermissionDenied(
                "You do not have permission to view dispatch control"
            )

        return models.DispatchControl.objects.get(
            organization_id=info.context.user.organization_id
        )

    def resolve_feasibility_tool_control(
        self, info: "GraphQLResolveInfo", **kwargs: typing.Any
    ) -> models.FeasibilityToolControl:
        if not FeasibilityToolControlNode.has_read_permission(info=info):
            raise PermissionDenied(
                "You do not have permission to view feasibility tool control"
            )

        return models.FeasibilityToolControl.objects.get(
            organization_id=info.context.user.organization_id
        )


schema = Schema(query=Query)
