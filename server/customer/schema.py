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

from django.db.models import Count, F, Max, Prefetch, Q
from django_filters import FilterSet
from graphene import Date, Field, Int, List, ObjectType, Schema, relay
from graphene_django import DjangoObjectType
from graphene_django.filter import DjangoFilterConnectionField

from customer import models
from utils.models import StatusChoices

if typing.TYPE_CHECKING:
    from django.db.models import QuerySet
    from graphql import GraphQLResolveInfo


class DeliverySlotNode(DjangoObjectType):
    """
    Delivery Slot Node for GraphQL
    """

    class Meta:
        model = models.DeliverySlot
        interfaces = (relay.Node,)
        exclude = ["customer"]

    @classmethod
    def get_queryset(
        cls, queryset: "QuerySet[models.DeliverySlot]", info: "GraphQLResolveInfo"
    ) -> "QuerySet[models.DeliverySlot]":
        return queryset.filter(organization_id=info.context.user.organization_id)


class CustomerContactNode(DjangoObjectType):
    """
    Customer Contact Node for GraphQL
    """

    class Meta:
        model = models.CustomerContact
        interfaces = (relay.Node,)
        exclude = ["customer"]

    @classmethod
    def get_queryset(
        cls, queryset: "QuerySet[models.CustomerContact]", info: "GraphQLResolveInfo"
    ) -> "QuerySet[models.CustomerContact]":
        return queryset.filter(organization_id=info.context.user.organization_id)


class CustomerEmailProfileNode(DjangoObjectType):
    """
    Customer Email Profile Node for GraphQL
    """

    class Meta:
        model = models.CustomerEmailProfile
        interfaces = (relay.Node,)
        exclude = ["customer"]

    @classmethod
    def get_queryset(
        cls,
        queryset: "QuerySet[models.CustomerEmailProfile]",
        info: "GraphQLResolveInfo",
    ) -> "QuerySet[models.CustomerEmailProfile]":
        return queryset.filter(organization_id=info.context.user.organization_id)


class CustomerRuleProfileNode(DjangoObjectType):
    """
    Customer Rule Profile Node for GraphQL
    """

    class Meta:
        model = models.CustomerRuleProfile
        interfaces = (relay.Node,)
        exclude = ["customer"]

    @classmethod
    def get_queryset(
        cls,
        queryset: "QuerySet[models.CustomerRuleProfile]",
        info: "GraphQLResolveInfo",
    ) -> "QuerySet[models.CustomerRuleProfile]":
        return queryset.filter(organization_id=info.context.user.organization_id)


class CustomerFilterSet(FilterSet):
    """
    FilterSet for Customer
    """

    class Meta:
        model = models.Customer
        fields = ["code", "name"]


class CustomerNode(DjangoObjectType):
    """
    Customer Node for GraphQL
    """

    ruleProfile = Field(CustomerRuleProfileNode)
    emailProfile = Field(CustomerEmailProfileNode)
    contacts = List(CustomerContactNode)
    deliverySlots = List(DeliverySlotNode)
    last_ship_date = Date()
    last_bill_date = Date()
    total_shipments = Int()

    class Meta:
        model = models.Customer
        interfaces = (relay.Node,)
        filterset_class = CustomerFilterSet
        fields = "__all__"

    @classmethod
    def get_queryset(
        cls, queryset: "QuerySet[models.Customer]", info: "GraphQLResolveInfo"
    ) -> "QuerySet[models.Customer]":
        user = info.context.user

        return (
            queryset.filter(organization_id=info.context.user.organization_id)
            .select_related(
                "email_profile",
                "rule_profile",
            )
            .prefetch_related(
                # Prefetch contacts for customer
                Prefetch(
                    lookup="contacts",
                    queryset=models.CustomerContact.objects.filter(
                        organization_id=user.organization_id
                    ).all(),
                ),
                # Prefetch Delivery slots for customer
                Prefetch(
                    lookup="delivery_slots",
                    queryset=models.DeliverySlot.objects.filter(
                        organization_id=user.organization_id
                    )
                    .annotate(
                        location_name=F("location__name"),
                    )
                    .all(),
                ),
                # Prefetch document classes for rule profile
                Prefetch(
                    lookup="rule_profile__document_class",
                    queryset=models.DocumentClassification.objects.filter(
                        organization_id=user.organization_id
                    ).only("id"),
                ),
            )
            .annotate(
                last_bill_date=Max(
                    "shipment__bill_date",
                    filter=Q(shipment__status=StatusChoices.COMPLETED),
                ),
                last_ship_date=Max(
                    "shipment__ship_date",
                    filter=Q(shipment__status=StatusChoices.COMPLETED),
                ),
                total_shipments=Count(
                    "shipment__id", filter=Q(shipment__status=StatusChoices.COMPLETED)
                ),
            )
            .order_by("code")
        )


class Query(ObjectType):
    """
    The Query class defines the GraphQL queries that can be made to the server
    """

    customer = relay.Node.Field(CustomerNode)
    customers = DjangoFilterConnectionField(CustomerNode)


schema = Schema(query=Query)
