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

from django.db.models import Prefetch, QuerySet
from rest_framework import viewsets

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
                "is_active",
                "name",
            )
        )

        return queryset


class CustomerBillingProfileViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing customer billing profile information in the system.

    The viewset provides default operations for creating, updating, and deleting customer
    billing profiles, as well as listing and retrieving customer billing profiles. It uses
    the `CustomerBillingProfileSerializer` class to convert the customer billing profile
    instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by customer billing profile ID,
    customer ID, and billing profile ID.
    """

    queryset = models.CustomerBillingProfile.objects.all()
    serializer_class = serializers.CustomerBillingProfileSerializer
    filterset_fields = ("is_active", "customer", "rule_profile")

    def get_queryset(self) -> QuerySet[models.CustomerBillingProfile]:
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "is_active",
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
