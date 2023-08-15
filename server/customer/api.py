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

from django.db.models import Prefetch, QuerySet
from rest_framework import status, viewsets
from rest_framework.decorators import action
from rest_framework.exceptions import ValidationError
from rest_framework.request import Request
from rest_framework.response import Response

from core.permissions import CustomObjectPermissions
from customer import models, serializers


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
    permission_classes = [CustomObjectPermissions]

    def get_queryset(self) -> QuerySet[models.Customer]:
        """Returns a queryset of customers for the current organization.

        Returns:
            A queryset of customers for the current organization.
        """
        queryset = (
            self.queryset.filter(
                organization_id=self.request.user.organization_id  # type: ignore
            )
            .prefetch_related(
                Prefetch(
                    lookup="contacts",
                    queryset=models.CustomerContact.objects.filter(
                        organization_id=self.request.user.organization_id  # type: ignore
                    ).only("id", "customer_id"),
                ),
                Prefetch(
                    lookup="billing_profile",
                    queryset=models.CustomerBillingProfile.objects.filter(
                        organization_id=self.request.user.organization_id  # type: ignore
                    ).only("id", "customer_id"),
                ),
            )
            .only(
                "id",
                "city",
                "code",
                "zip_code",
                "address_line_1",
                "address_line_2",
                "organization_id",
                "state",
                "has_customer_portal",
                "status",
                "name",
            )
        )

        return queryset


class CustomerBillingProfileViewSet(viewsets.ModelViewSet):
    """A viewset for managing Customer Billing Profile records.

    This viewset handles HTTP requests to perform CRUD operations on the customer billing profile model. Authenticated
    users can perform operations to create, update, view, list or delete the customer billing profiles. It also support
    filtering on 'status', 'customer', and 'rule_profile' fields.

    Attributes:
        queryset (QuerySet): QuerySet of CustomerBillingProfile model instances.
        serializer_class ('CustomerBillingProfileSerializer'): The serializer to handle the conversion to/from JSON for CustomerBillingProfile.
        filterset_fields (tuple): Fields on which the queryset can be filtered.
        permission_classes (list): The class-based views to be applied to the viewset.

    Methods:
        customer_billing_profile_details: Handle GET requests and return the customer billing details for a given customer ID.
        get_queryset: Custom method to get the queryset.

    """

    queryset = models.CustomerBillingProfile.objects.all()
    serializer_class = serializers.CustomerBillingProfileSerializer
    filterset_fields = ("status", "customer", "rule_profile")
    permission_classes = [CustomObjectPermissions]

    @action(detail=False, methods=["get"])
    def customer_billing_profile_details(
        self, request: Request, *args: typing.Any, **kwargs: typing.Any
    ) -> Response:
        """Handle GET requests and return the customer billing details for a given customer ID.

        Args:
            request (Request): Django Request object.
            *args: Variable length argument list.
            **kwargs: Arbitrary keyword arguments.

        Returns:
            Response: The Django Rest Framework Response object, containing the serialized details
                      of the customer billing profile or a validation error message.

        Raises:
            ValidationError: If 'customer_id' parameter is not found in the request.

        """
        customer_id = request.query_params.get("customer_id")

        if not customer_id:
            raise ValidationError("Query parameter 'customer_id' is required.")

        queryset = models.CustomerBillingProfile.objects.get(
            customer_id=customer_id, organization_id=self.request.user.organization_id  # type: ignore
        )
        serializer = serializers.CustomerBillingProfileSerializer(queryset)
        return Response(data=serializer.data, status=status.HTTP_200_OK)

    def get_queryset(self) -> QuerySet[models.CustomerBillingProfile]:
        """Custom method to get the queryset. Filters the queryset based on organization
        that the authenticated user is part of and selects only selected fields.

        Returns:
            QuerySet: A QuerySet containing the customer billing profiles for the organization
                      that authenticated user is part of.
        """
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "status",
            "rule_profile_id",
            "email_profile_id",
            "customer_id",
            "organization_id",
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
        filterset_fields (tuple): A tuple containing the fields on which users can filter.
        permission_classes (list): Permissions that must be fulfilled to access or modify the data.
    """

    queryset = models.CustomerEmailProfile.objects.all()
    serializer_class = serializers.CustomerEmailProfileSerializer
    filterset_fields = ("name",)
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
