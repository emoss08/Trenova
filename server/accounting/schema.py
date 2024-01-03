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

from django.db.models import QuerySet, Prefetch
from django_filters import FilterSet
from graphene import ObjectType, relay, Schema, Field, List
from graphene_django import DjangoObjectType
from graphene_django.filter import DjangoFilterConnectionField

from accounting import models

if typing.TYPE_CHECKING:
    from graphql import GraphQLResolveInfo


class AccountingControlNode(DjangoObjectType):
    """Accounting Control Node for GraphQL.

    Notes:
        - This is only available to users with the `accounting.view_accountingcontrol` permission.
        - Available by default to anyone that is an admin.
    """

    class Meta:
        model = models.AccountingControl
        interfaces = (relay.Node,)
        fields = "__all__"

    @staticmethod
    def has_read_permission(*, info: "GraphQLResolveInfo") -> bool:
        user = info.context.user
        return user.is_superuser or user.has_perm("accounting.view_accountingcontrol")


class TagFilterSet(FilterSet):
    """
    FilterSet for Tag
    """

    class Meta:
        model = models.Tag
        fields = ["name"]


class TagNode(DjangoObjectType):
    """
    Tag Node for GraphQL.
    """

    class Meta:
        """
        Meta class for Tag Node
        """

        model = models.Tag
        filterset_class = TagFilterSet
        interfaces = (relay.Node,)
        fields = ["name", "description", "created", "modified"]

    @classmethod
    def get_queryset(
        cls, queryset: "QuerySet[models.Tag]", info: "GraphQLResolveInfo"
    ) -> "QuerySet[models.Tag]":
        return queryset.filter(organization_id=info.context.user.organization_id)


class GeneralLedgerAccountFilterSet(FilterSet):
    """
    FilterSet for GeneralLedgerAccount
    """

    class Meta:
        model = models.GeneralLedgerAccount
        fields = [
            "status",
            "account_number",
            "account_type",
            "cash_flow_type",
            "account_sub_type",
            "account_classification",
        ]


class GeneralLedgerAccountNode(DjangoObjectType):
    tags = List(TagNode)

    class Meta:
        model = models.GeneralLedgerAccount
        filterset_class = GeneralLedgerAccountFilterSet
        interfaces = (relay.Node,)
        fields = "__all__"

    @classmethod
    def get_queryset(
        cls,
        queryset: "QuerySet[models.GeneralLedgerAccount]",
        info: "GraphQLResolveInfo",
    ) -> "QuerySet[models.GeneralLedgerAccount]":
        user_org_id = info.context.user.organization_id

        return queryset.filter(organization_id=user_org_id).prefetch_related(
            Prefetch(
                "tags", queryset=models.Tag.objects.filter(organization_id=user_org_id)
            )
        )

    def resolve_tags(self, info: "GraphQLResolveInfo") -> typing.List[models.Tag]:
        # Since we have already prefetched tags, we can directly return them
        return self.tags.all()


class RevenueCodeFilterSet(FilterSet):
    """
    FilterSet for RevenueCode
    """

    class Meta:
        model = models.RevenueCode
        fields = ["code"]


class RevenueCodeNode(DjangoObjectType):
    """
    RevenueCode Node for GraphQL.
    """

    class Meta:
        model = models.RevenueCode
        filterset_class = RevenueCodeFilterSet
        interfaces = (relay.Node,)
        fields = "__all__"

    @classmethod
    def get_queryset(
        cls, queryset: "QuerySet[models.RevenueCode]", info: "GraphQLResolveInfo"
    ) -> "QuerySet[models.RevenueCode]":
        return queryset.filter(organization_id=info.context.user.organization_id)


class DivisionCodeFilterSet(FilterSet):
    """
    FilterSet for DivisionCode
    """

    class Meta:
        model = models.DivisionCode
        fields = ["status", "cash_account", "ap_account", "expense_account"]


class DivisionCodeNode(DjangoObjectType):
    """
    DivisionCode Node for GraphQL.
    """

    class Meta:
        model = models.DivisionCode
        filterset_class = DivisionCodeFilterSet
        interfaces = (relay.Node,)
        fields = "__all__"

    @classmethod
    def get_queryset(
        cls, queryset: "QuerySet[models.DivisionCode]", info: "GraphQLResolveInfo"
    ) -> "QuerySet[models.DivisionCode]":
        return queryset.filter(organization_id=info.context.user.organization_id)


class Query(ObjectType):
    """
    The Query class defines the GraphQL queries that can be made to the server
    """

    accounting_control = Field(AccountingControlNode)
    tag = relay.Node.Field(TagNode)
    tags = DjangoFilterConnectionField(TagNode)
    gl_account = relay.Node.Field(GeneralLedgerAccountNode)
    gl_accounts = DjangoFilterConnectionField(GeneralLedgerAccountNode)
    revenue_code = relay.Node.Field(RevenueCodeNode)
    revenue_codes = DjangoFilterConnectionField(RevenueCodeNode)
    division_code = relay.Node.Field(DivisionCodeNode)
    division_codes = DjangoFilterConnectionField(DivisionCodeNode)

    def resolve_accounting_control(
        self, info: "GraphQLResolveInfo", **kwargs: typing.Any
    ) -> models.AccountingControl:
        if not AccountingControlNode.has_read_permission(info=info):
            raise PermissionError(
                "You do not have permission to view AccountingControl"
            )

        return models.AccountingControl.objects.get(
            organization_id=info.context.user.organization_id
        )


schema = Schema(query=Query)
