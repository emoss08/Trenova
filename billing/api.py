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
from django.db.models import QuerySet
from drf_spectacular.types import OpenApiTypes
from drf_spectacular.utils import OpenApiParameter, extend_schema
from rest_framework import permissions, status
from rest_framework.decorators import api_view
from rest_framework.request import Request
from rest_framework.response import Response

from billing import models, serializers, tasks
from utils.views import OrganizationMixin


class BillingControlViewSet(OrganizationMixin):
    """A viewset for viewing and editing BillingControl in the system.

    The viewset provides default operations for creating, updating Order Control,
    as well as listing and retrieving Order Control. It uses the ``BillingControlSerializer``
    class to convert the order control instance to and from JSON-formatted data.

    Only admin users are allowed to access the views provided by this viewset.

    Attributes:
        queryset (QuerySet): A queryset of BillingControl objects that will be used to
        retrieve and update BillingControl objects.

        serializer_class (BillingControlSerializer): A serializer class that will be used to
        convert BillingControl objects to and from JSON-formatted data.

        permission_classes (list): A list of permission classes that will be used to
        determine if a user has permission to perform a particular action.
    """

    queryset = models.BillingControl.objects.all()
    permission_classes = [permissions.IsAdminUser]
    serializer_class = serializers.BillingControlSerializer
    http_method_names = ["get", "put", "patch", "head", "options"]

    def get_queryset(self) -> QuerySet[models.BillingControl]:
        queryset = self.queryset.filter(
            organization=self.request.user.organization  # type: ignore
        ).only(
            "id",
            "order_transfer_criteria",
            "auto_mark_ready_to_bill",
            "remove_billing_history",
            "validate_customer_rates",
            "auto_bill_criteria",
            "auto_bill_orders",
            "enforce_customer_billing",
            "organization__id",
        )
        return queryset


class BillingQueueViewSet(OrganizationMixin):
    """
    A viewset for viewing and editing billing queue in the system.

    The viewset provides default operations for creating, updating, and deleting records in
    billing queue,as well as listing and retrieving charge types. It uses the `BillingQueueSerializer`
    class to convert the charge type instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by `order` pro_number, `worker` code, `customer`
    code, `revenue_code` code and `order_type` id.
    """

    queryset = models.BillingQueue.objects.all()
    serializer_class = serializers.BillingQueueSerializer
    filterset_fields = (
        "order__pro_number",
        "worker__code",
        "customer__code",
        "revenue_code__code",
        "order_type",
    )
    http_method_names = ["get", "put", "patch", "post", "head", "options"]

    def get_queryset(self) -> QuerySet[models.BillingQueue]:
        queryset = (
            self.queryset.filter(
                organization=self.request.user.organization  # type: ignore
            )
            .select_related("organization")
            .only(
                "id",
                "freight_charge_amount_currency",
                "commodity__id",
                "bol_number",
                "total_amount_currency",
                "other_charge_total_currency",
                "commodity_descr",
                "consignee_ref_number",
                "mileage",
                "total_amount",
                "pieces",
                "user__id",
                "freight_charge_amount",
                "invoice_number",
                "customer__id",
                "bill_date",
                "weight",
                "ready_to_bill",
                "is_cancelled",
                "worker__id",
                "other_charge_total",
                "order_type__id",
                "organization__id",
                "order__id",
                "revenue_code__id",
                "is_summary",
                "bill_type",
            )
        )

        return queryset


class BillingHistoryViewSet(OrganizationMixin):
    """
    A viewset for viewing and editing billing history in the system.

    The viewset provides default operation for viewing billing history,
    as well as listing and retrieving charge types. It uses the `BillingHistorySerializer` class to
    convert the charge type instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by `order` pro_number, `worker` code, `customer`
    code, `revenue_code` code and `order_type` id.
    """

    queryset = models.BillingHistory.objects.all()
    serializer_class = serializers.BillingHistorySerializer
    filterset_fields = (
        "order__pro_number",
        "worker__code",
        "customer__code",
        "revenue_code__code",
        "order_type",
    )
    http_method_names = ["get", "head", "options"]

    def get_queryset(self) -> QuerySet[models.BillingHistory]:
        queryset = (
            self.queryset.filter(
                organization=self.request.user.organization  # type: ignore
            )
            .select_related("organization")
            .only(
                "id",
                "freight_charge_amount_currency",
                "commodity__id",
                "bol_number",
                "total_amount_currency",
                "other_charge_total_currency",
                "commodity_descr",
                "consignee_ref_number",
                "mileage",
                "total_amount",
                "pieces",
                "user__id",
                "freight_charge_amount",
                "invoice_number",
                "customer__id",
                "bill_date",
                "weight",
                "is_cancelled",
                "worker__id",
                "other_charge_total",
                "order_type__id",
                "organization__id",
                "order__id",
                "revenue_code__id",
                "is_summary",
                "bill_type",
            )
        )
        return queryset


class BillingTransferLogViewSet(OrganizationMixin):
    """
    A viewset for viewing billing transfers in the system.

    The viewset provides default operation for viewing billing history,
    as well as listing and retrieving charge types. It uses the `BillingTransferLogSerializer`
    class to convert the charge type instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by `order` pro_number, & `user` username.
    """

    queryset = models.BillingTransferLog.objects.all()
    serializer_class = serializers.BillingTransferLogSerializer
    filterset_fields = (
        "order__pro_number",
        "transferred_by__username",
    )
    http_method_names = ["get", "head", "options"]


class ChargeTypeViewSet(OrganizationMixin):
    """
    A viewset for viewing and editing charge types in the system.

    The viewset provides default operations for creating, updating, and deleting charge types,
    as well as listing and retrieving charge types. It uses the `ChargeTypeSerializer` class to
    convert the charge type instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by charge type ID, name, and code.
    """

    queryset = models.ChargeType.objects.all()
    serializer_class = serializers.ChargeTypeSerializer
    filterset_fields = ("name",)

    def get_queryset(self) -> QuerySet[models.ChargeType]:
        queryset = (
            self.queryset.filter(
                organization=self.request.user.organization  # type: ignore
            )
            .select_related("organization")
            .only(
                "id",
                "organization__id",
                "name",
                "description",
            )
        )
        return queryset


class AccessorialChargeViewSet(OrganizationMixin):
    """
    A viewset for viewing and editing accessorial charges in the system.

    The viewset provides default operations for creating, updating, and
    deleting accessorial charges, as well as listing and retrieving accessorial
    charges. It uses the `AccessorialChargeSerializer` class to convert the
    accessorial charge instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by accessorial charge
    ID, code, and method.
    """

    queryset = models.AccessorialCharge.objects.all()
    serializer_class = serializers.AccessorialChargeSerializer
    filterset_fields = ("code", "is_detention", "method")

    def get_queryset(self) -> QuerySet[models.AccessorialCharge]:
        queryset = (
            self.queryset.filter(
                organization=self.request.user.organization  # type: ignore
            )
            .select_related("organization")
            .only(
                "id",
                "is_detention",
                "charge_amount",
                "code",
                "description",
                "method",
                "organization__id",
                "charge_amount_currency",
            )
        )

        return queryset


class DocumentClassificationViewSet(OrganizationMixin):
    """
    A viewset for viewing and editing document classifications in the system.

    The viewset provides default operations for creating, updating, and
    deleting document classifications, as well as listing and retrieving document classifications.
    It uses the `DocumentClassificationSerializer`
    class to convert the document classification instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by document classification
    ID, and name.
    """

    queryset = models.DocumentClassification.objects.all()
    serializer_class = serializers.DocumentClassificationSerializer
    filterset_fields = ("name",)

    def get_queryset(self) -> QuerySet[models.DocumentClassification]:
        queryset = (
            self.queryset.filter(
                organization=self.request.user.organization  # type: ignore
            )
            .select_related("organization")
            .only(
                "id",
                "name",
                "organization__id",
                "description",
            )
        )
        return queryset


@extend_schema(
    tags=["Bill Order"],
    description="Starts the billing tasks for one order.",
    parameters=[
        OpenApiParameter(
            name="order_id",
            type=OpenApiTypes.UUID,
            description="The order id to be billed.",
        ),
    ],
    request=None,
    responses={
        (200, "application/json"): {
            "type": "object",
            "properties": {
                "message": {
                    "type": "string",
                    "example": "Order ID is required. Please Try Again.",
                },
            },
        },
        (400, "application/json"): {
            "type": "object",
            "properties": {
                "message": {
                    "type": "string",
                    "example": "Billing task started.",
                },
            },
        },
    },
)
@api_view(["POST"])
def bill_invoice_view(request: Request) -> Response:
    """
    Bill an invoice.

    Args:
        request (Request): The request object.

    Returns:
        Response: A response object containing the result of the operation.
    """
    invoice_id = request.data.get("invoice_id")

    if not invoice_id:
        return Response(
            {"message": "Invoice ID is required. Please Try Again."},
            status=status.HTTP_400_BAD_REQUEST,
        )

    tasks.bill_invoice_task.delay(user_id=request.user.id, invoice_id=invoice_id)
    return Response({"message": "Billing task started."}, status=status.HTTP_200_OK)


@extend_schema(
    tags=["Mass Billing Order"],
    description="Starts the mass billing task.",
    request=None,
    responses={
        (200, "application/json"): {
            "type": "object",
            "properties": {
                "message": {
                    "type": "string",
                    "example": "Mass Billing task started.",
                },
            },
        },
        (400, "application/json"): {
            "type": "object",
            "properties": {
                "message": {
                    "type": "string",
                    "example": "Mass billing does not accept any data. Please Try Again.",
                },
            },
        },
    },
)
@api_view(["POST"])
def mass_order_bill(request: Request) -> Response:
    """
    Mass bill orders.

    Args:
        request (Request): The request object.

    Returns:
        Response: A response object containing the result of the operation.
    """
    if request.data:
        return Response(
            {"message": "Mass billing does not accept any data. Please Try Again."},
            status=status.HTTP_400_BAD_REQUEST,
        )

    tasks.mass_order_bill_task.delay(user_id=request.user.id)

    return Response(
        {"message": "Mass Billing task started."}, status=status.HTTP_200_OK
    )


@api_view(["POST"])
def transfer_to_billing(request: Request) -> Response:
    """
    Starts a Celery task to transfer the specified order(s) to billing for the logged in user.

    Args:
        request: A Django Request object that contains the order(s) to transfer to billing.

    Returns:
        A Django Response object with a success message and a 200 status code if the transfer task
        was successfully started. If no order(s) are provided in the request, a 400 status code and
        an error message will be returned.

    Raises:
        N/A

    This view function expects a POST request containing an `order_pros` key in the request data,
    which should be a list of order IDs to be transferred to billing. If no `order_pros` key is
    provided, the function will return a response with a 400 status code and an error message.

    If the request data is valid, the function will start a Celery task with the provided order IDs
    and the ID of the currently logged-in user. The task will run in the background and transfer
    the specified order(s) to billing.

    Upon successfully starting the Celery task, the function will return a response with a 200 status
    code and a success message indicating that the transfer task has been started.
    """
    order_pros = request.data.get("order_pros")

    if not order_pros:
        return Response(
            {"message": "Order Pro(s) is required. Please Try Again."},
            status=status.HTTP_400_BAD_REQUEST,
        )

    tasks.transfer_to_billing_task.delay(
        user_id=str(request.user.id), order_pros=order_pros
    )

    return Response(
        {"message": "Transfer to billing task started."}, status=status.HTTP_200_OK
    )
