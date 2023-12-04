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

import typing

from core.permissions import CustomObjectPermissions
from customer import models, serializers
from django.db.models import Count, Max, Prefetch, Q, QuerySet
from rest_framework import status, viewsets
from rest_framework.decorators import action
from rest_framework.exceptions import ValidationError
from rest_framework.generics import get_object_or_404
from rest_framework.response import Response
from utils.models import StatusChoices

if typing.TYPE_CHECKING:
    from rest_framework.request import Request


class DeliverySlotViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing delivery slot information in the system.

    The viewset provides default operations for creating, updating, and deleting delivery slots,
    as well as listing and retrieving delivery slots. It uses the `DeliverySlotSerializer`
    class to convert the delivery slot instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by delivery slot ID, start time,
    and end time.
    """

    queryset = models.DeliverySlot.objects.all()
    serializer_class = serializers.DeliverySlotSerializer
    filterset_fields = ("start_time", "end_time")
    permission_classes = [CustomObjectPermissions]


class CustomerViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing customer information in the system.

    The viewset provides default operations for creating, updating, and deleting customers,
    as well as listing and retrieving customers. It uses the `CustomerSerializer`
    class to convert the customer instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by customer ID, name, and
    code.
    """

    queryset = models.Customer.objects.all()
    serializer_class = serializers.CustomerSerializer
    filterset_fields = ("code", "name")
    search_fields = ("code", "name")
    permission_classes = [CustomObjectPermissions]

    def get_queryset(self) -> QuerySet[models.Customer]:
        """Returns a queryset of customers for the current organization.

        Returns:
            A queryset of customers for the current organization.
        """

        user_org = self.request.user.organization_id

        queryset = (
            self.queryset.filter(organization_id=user_org)
            .select_related(
                "email_profile",
                "rule_profile",
            )
            .prefetch_related(
                Prefetch(
                    lookup="contacts",
                    queryset=models.CustomerContact.objects.filter(
                        organization_id=user_org
                    ).all(),
                ),
                Prefetch(
                    lookup="delivery_slots",
                    queryset=models.DeliverySlot.objects.filter(
                        organization_id=user_org
                    ).all(),
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
            .all()
        )

        return queryset


class CustomerFuelTableViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing customer fuel table information in the system.

    The viewset provides default operations for creating, updating, and deleting customer
    fuel tables, as well as listing and retrieving customer fuel tables. It uses the
    `CustomerFuelTableSerializer` class to convert the customer fuel table instances to
    and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by customer fuel table ID,
    customer ID, and customer name.
    """

    queryset = models.CustomerFuelTable.objects.all()
    serializer_class = serializers.CustomerFuelTableSerializer
    filterset_fields = (
        "id",
        "name",
    )
    permission_classes = [CustomObjectPermissions]

    def get_queryset(self) -> QuerySet[models.CustomerFuelTable]:
        """Get the queryset for the viewset.

        The queryset is filtered by the organization of the user making the request.

        Returns:
            The filtered queryset.
        """
        queryset = (
            self.queryset.filter(
                organization_id=self.request.user.organization_id  # type: ignore
            )
            .prefetch_related(
                Prefetch(
                    lookup="customer_fuel_table_details",
                    queryset=models.CustomerFuelTableDetail.objects.filter(
                        organization_id=self.request.user.organization_id  # type: ignore
                    ).only(
                        "id",
                        "customer_fuel_table_id",
                        "percentage",
                        "start_price",
                        "method",
                        "organization_id",
                        "amount",
                    ),
                ),
            )
            .only(
                "id",
                "organization_id",
                "name",
                "description",
            )
        )

        return queryset


class CustomerRuleProfileViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing customer rule profile information in the system.

    The viewset provides default operations for creating, updating, and deleting customer
    rule profiles, as well as listing and retrieving customer rule profiles. It uses the
    `CustomerRuleProfileSerializer` class to convert the customer rule profile instances
    to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by customer rule profile ID,
    customer ID, and customer name.
    """

    queryset = models.CustomerRuleProfile.objects.all()
    serializer_class = serializers.CustomerRuleProfileSerializer
    filterset_fields = ("name",)
    permission_classes = [CustomObjectPermissions]

    @action(
        detail=False, methods=["GET"], name="Get Customer Rule Profile by Customer ID"
    )
    def get_by_customer_id(self, request: "Request") -> Response:
        """Get Customer Rule Profile by Customer ID.

        Args:
            request: The HTTP request object.

        Returns:
            A Response object containing the serialized CustomerRuleProfile.

        Raises:
            ValidationError: If the `customer_id` query parameter is missing.

        Example:
            An example of how to call the API to get the Customer Rule Profile by Customer ID:

                GET /api/customer-rule-profiles/get-by-customer-id?customer_id=123456
        """
        customer_id = request.query_params.get("customer_id")
        if not customer_id:
            raise ValidationError("Query param `customer_id` is required.")

        customer_rule_profile = get_object_or_404(
            models.CustomerRuleProfile, id=customer_id
        )
        serializer = serializers.CustomerRuleProfileSerializer(customer_rule_profile)
        return Response(serializer.data, status=status.HTTP_200_OK)

    def get_queryset(self) -> QuerySet[models.CustomerRuleProfile]:
        """Get the queryset for the viewset.

        The queryset is filtered by the organization of the user making the request.

        Returns:
            The filtered queryset.
        """
        queryset = (
            self.queryset.filter(
                organization_id=self.request.user.organization_id  # type: ignore
            )
            .prefetch_related(
                Prefetch(
                    lookup="document_class",
                    queryset=models.DocumentClassification.objects.filter(
                        organization_id=self.request.user.organization_id  # type: ignore
                    ).only("id"),
                )
            )
            .annotate()
            .only(
                "id",
                "organization_id",
                "name",
            )
        )

        return queryset


class CustomerEmailProfileViewSet(viewsets.ModelViewSet):
    """Model View Set for CustomerEmailProfile.

    This ViewSet provides complete CRUD operations for CustomerEmailProfile.
    It uses Django queryset to fetch all objects of Customer Email Profile and also uses
    a custom serializer 'CustomerEmailProfileSerializer' for the serialization of the data.
    The users are able to filter the data based on the field "name".
    In addition, for listening to appropriate actions, CustomObjectPermissions are maintained.

    Attributes:
        queryset (object): A Django Queryset that fetches all CustomerEmailProfile objects from the database.
        serializer_class (class): The serializer class used for the serialization of this model's data.
        permission_classes (list): Permissions that must be fulfilled to access or modify the data.
    """

    queryset = models.CustomerEmailProfile.objects.all()
    serializer_class = serializers.CustomerEmailProfileSerializer
    permission_classes = [CustomObjectPermissions]

    def get_queryset(self) -> QuerySet[models.CustomerEmailProfile]:
        """
        Get queryset specifically for the current user's organization.

        Get and return queryset of CustomerEmailProfile objects which belong to the same organization
        as the user making the request. The user's organization is determined using the `organization_id`
        attribute from the authenticated user's request object.

        Returns:
            QuerySet[models.CustomerEmailProfile]: A QuerySet containing CustomerEmailProfile objects belonging to the
            current user's organization.
        """
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        )
        return queryset

    @action(
        detail=False, methods=["GET"], name="Get Customer Email Profile by Customer ID"
    )
    def get_by_customer_id(self, request: "Request") -> "Response":
        """Get Customer Email Profile by Customer ID.

        Args:
            request: The HTTP request object.

        Returns:
            A Response object containing the serialized CustomerEmailProfile.

        Raises:
            ValidationError: If the `customer_id` query parameter is missing.

        Example:
            An example of how to call the API to get the Customer Email Profile by Customer ID:

                GET /api/customer-email-profiles/get-by-customer-id?customer_id=123456
        """
        customer_id = request.query_params.get("customer_id")
        if not customer_id:
            raise ValidationError("Query param `customer_id` is required.")

        customer_email_profile = get_object_or_404(
            models.CustomerEmailProfile, id=customer_id
        )

        serializer = serializers.CustomerEmailProfileSerializer(customer_email_profile)
        return Response(serializer.data, status=status.HTTP_200_OK)
