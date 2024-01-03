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

from billing import models
from django.core.exceptions import PermissionDenied
from django_filters import FilterSet
from graphene import Field, ObjectType, Schema, relay
from graphene_django import DjangoObjectType
from graphene_django.filter import DjangoFilterConnectionField

if typing.TYPE_CHECKING:
    from django.db.models import QuerySet
    from graphql import GraphQLResolveInfo


class BillingcontrolNode(DjangoObjectType):
    """Billing Control Node for GraphQL.

    Notes:
        - This is only available to users with the `billing.view_billingcontrol` permission.
        - Available by default to anyone that is an admin.
    """

    class Meta:
        model = models.BillingControl
        interfaces = (relay.Node,)
        fields = "__all__"

    @staticmethod
    def has_read_permission(*, info: "GraphQLResolveInfo") -> bool:
        user = info.context.user
        return user.is_superuser or user.has_perm("billing.view_billingcontrol")


class ChargeTypeFilterSet(FilterSet):
    """
    FilterSet for ChargeType
    """

    class Meta:
        model = models.ChargeType
        fields = ["name"]


class ChargeTypeNode(DjangoObjectType):
    """
    Charge Type Node for GraphQL
    """

    class Meta:
        model = models.ChargeType
        interfaces = (relay.Node,)
        filterset_class = ChargeTypeFilterSet
        fields = "__all__"

    @classmethod
    def get_queryset(
        cls, queryset: "QuerySet[models.ChargeType]", info: "GraphQLResolveInfo"
    ) -> "QuerySet[models.ChargeType]":
        return queryset.filter(organization_id=info.context.user.organization_id)


class AccessorialChargeFilterSet(FilterSet):
    """
    FilterSet for AccessorialCharge
    """

    class Meta:
        model = models.AccessorialCharge
        fields = ["code", "is_detention", "method"]


class AccessorialChargeNode(DjangoObjectType):
    """
    Charge Type Node for GraphQL
    """

    class Meta:
        model = models.AccessorialCharge
        interfaces = (relay.Node,)
        filterset_class = AccessorialChargeFilterSet
        fields = "__all__"

    @classmethod
    def get_queryset(
        cls, queryset: "QuerySet[models.AccessorialCharge]", info: "GraphQLResolveInfo"
    ) -> "QuerySet[models.AccessorialCharge]":
        return queryset.filter(organization_id=info.context.user.organization_id)


class DocumentClassificationFilterSet(FilterSet):
    """
    FilterSet for DocumentClassification
    """

    class Meta:
        model = models.DocumentClassification
        fields = ["name"]


class DocumentClassificationNode(DjangoObjectType):
    """
    Charge Type Node for GraphQL
    """

    class Meta:
        model = models.DocumentClassification
        interfaces = (relay.Node,)
        filterset_class = DocumentClassificationFilterSet
        fields = "__all__"

    @classmethod
    def get_queryset(
        cls,
        queryset: "QuerySet[models.DocumentClassification]",
        info: "GraphQLResolveInfo",
    ) -> "QuerySet[models.DocumentClassification]":
        return queryset.filter(organization_id=info.context.user.organization_id)


class Query(ObjectType):
    """
    The Query class defines the GraphQL queries that can be made to the server
    """

    billing_control = Field(BillingcontrolNode)
    charge_type = relay.Node.Field(ChargeTypeNode)
    charge_types = DjangoFilterConnectionField(ChargeTypeNode)
    accessorial_charge = relay.Node.Field(AccessorialChargeNode)
    accessorial_charges = DjangoFilterConnectionField(AccessorialChargeNode)
    document_classification = relay.Node.Field(DocumentClassificationNode)
    document_classifications = DjangoFilterConnectionField(DocumentClassificationNode)

    def resolve_billing_control(
        self, info: "GraphQLResolveInfo", **kwargs: typing.Any
    ) -> models.BillingControl:
        if not BillingcontrolNode.has_read_permission(info=info):
            raise PermissionDenied("You do not have permission to view billing control")

        return models.BillingControl.objects.get(
            organization_id=info.context.user.organization_id
        )


schema = Schema(query=Query)
