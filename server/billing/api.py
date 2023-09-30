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
from django.core.exceptions import ObjectDoesNotExist
from django.db.models import QuerySet
from drf_spectacular.types import OpenApiTypes
from drf_spectacular.utils import OpenApiParameter, extend_schema
from rest_framework import permissions, status, viewsets
from rest_framework.decorators import api_view
from rest_framework.request import Request
from rest_framework.response import Response

from billing import models, selectors, serializers, services, tasks, validation
from core.permissions import CustomObjectPermissions


class BillingControlViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing BillingControl in the system.

    The viewset provides default operations for creating, updating Shipment Control,
    as well as listing and retrieving Shipment Control. It uses the ``BillingControlSerializer``
    class to convert the Shipment Control instance to and from JSON-formatted data.

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
    filterset_fields = ("organization_id",)

    def get_queryset(self) -> QuerySet[models.BillingControl]:
        """The get_queryset function is used to filter the queryset based on the request.
        In this case, we are filtering by `organization_id` so that each user can only see their own billing controls.

        Args:
            self: Refer to the class itself, and is used in this case to access the queryset attribute

        Returns:
            QuerySet[models.BillingControl]: A queryset of the `billingcontrol` model
        """
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "shipment_transfer_criteria",
            "auto_mark_ready_to_bill",
            "remove_billing_history",
            "validate_customer_rates",
            "auto_bill_criteria",
            "auto_bill_shipments",
            "enforce_customer_billing",
            "organization_id",
        )
        return queryset


class BillingQueueViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing billing queue in the system.

    The viewset provides default operations for creating, updating, and deleting records in
    billing queue,as well as listing and retrieving charge types. It uses the `BillingQueueSerializer`
    class to convert the charge type instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by `shipment` pro_number, `worker` code, `customer`
    code, `revenue_code` code and `shipment_type` id.
    """

    queryset = models.BillingQueue.objects.all()
    serializer_class = serializers.BillingQueueSerializer
    filterset_fields = (
        "shipment__pro_number",
        "worker__code",
        "customer__code",
        "revenue_code__code",
        "shipment_type",
    )
    http_method_names = ["get", "put", "patch", "post", "head", "options"]
    permission_classes = [CustomObjectPermissions]

    def get_queryset(self) -> QuerySet[models.BillingQueue]:
        """The get_queryset function is used to filter the queryset based on the request.
        The function returns a QuerySet of `BillingQueue` objects that are filtered by `organization_id`,
        which is equal to the user's `organization_id`. The only() method limits which fields are returned in
        the response.

        Args:
            self: Represent the instance of the class

        Returns:
            QuerySet[models.BillingQueue]: A queryset that is filtered by the `organization_id`
        """

        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "commodity_id",
            "bol_number",
            "commodity_descr",
            "consignee_ref_number",
            "mileage",
            "total_amount",
            "pieces",
            "user_id",
            "freight_charge_amount",
            "invoice_number",
            "customer_id",
            "bill_date",
            "weight",
            "ready_to_bill",
            "is_cancelled",
            "worker_id",
            "other_charge_total",
            "shipment_type_id",
            "organization_id",
            "shipment_id",
            "revenue_code_id",
            "is_summary",
            "bill_type",
        )

        return queryset


class BillingLogEntryViewSet(viewsets.ModelViewSet):
    """A viewset for viewing and editing billing log entry in the system.

    The viewset provides default operations for creating, updating, and deleting records in
    billing queue,as well as listing and retrieving charge types. It uses the `BillingLogEntrySerializer`
    class to convert the charge type instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    """

    queryset = models.BillingLogEntry.objects.all()
    serializer_class = serializers.BillingLogEntrySerializer
    permission_classes = [CustomObjectPermissions]

    def get_queryset(self) -> QuerySet[models.BillingLogEntry]:
        """The get_queryset function is used to filter the queryset based on the request.
        The function returns a QuerySet of `BillingLogEntry` objects that are filtered by `organization_id`,
        which is equal to the user's `organization_id`. The only() method limits which fields are returned in
        the response.

        Args:
            self: Represent the instance of the class

        Returns:
            QuerySet[models.BillingLogEntry]: A queryset that is filtered by the `organization_id`
        """

        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "task_id",
            "object_pk",
            "content_type",
            "action",
            "shipment",
            "invoice_number",
            "customer",
            "actor",
        )

        return queryset


class BillingHistoryViewSet(viewsets.ModelViewSet):
    """
    A viewset for viewing and editing billing history in the system.

    The viewset provides default operation for viewing billing history,
    as well as listing and retrieving charge types. It uses the `BillingHistorySerializer` class to
    convert the charge type instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this viewset.
    Filtering is also available, with the ability to filter by `order` pro_number, `worker` code, `customer`
    code, `revenue_code` code and `shipment_type` id.
    """

    queryset = models.BillingHistory.objects.all()
    serializer_class = serializers.BillingHistorySerializer
    filterset_fields = (
        "shipment__pro_number",
        "worker__code",
        "customer",
        "revenue_code__code",
        "shipment_type",
    )
    http_method_names = ["get", "head", "options"]
    permission_classes = [CustomObjectPermissions]

    def get_queryset(self) -> QuerySet[models.BillingHistory]:
        """The get_queryset function is used to filter the queryset based on the request.
        The function takes in a self parameter, which is an instance of the class that inherits from viewset.ModelViewSet.
        In this case, it's BillingHistoryViewSet. The ``get_queryset`` function returns a QuerySet object.

        Args:
            self: Refer to the class itself

        Returns:
            QuerySet[models.BillingHistory]: A queryset that is filtered by the `organization_id`
        """
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "commodity_id",
            "bol_number",
            "commodity_descr",
            "consignee_ref_number",
            "mileage",
            "total_amount",
            "pieces",
            "user_id",
            "freight_charge_amount",
            "invoice_number",
            "customer_id",
            "bill_date",
            "weight",
            "is_cancelled",
            "worker_id",
            "other_charge_total",
            "shipment_type_id",
            "organization_id",
            "shipment_id",
            "revenue_code_id",
            "is_summary",
            "bill_type",
        )
        return queryset


class ChargeTypeViewSet(viewsets.ModelViewSet):
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
    permission_classes = [CustomObjectPermissions]

    def get_queryset(self) -> QuerySet[models.ChargeType]:
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "organization_id",
            "name",
            "description",
        )
        return queryset


class AccessorialChargeViewSet(viewsets.ModelViewSet):
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
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "is_detention",
            "charge_amount",
            "code",
            "description",
            "method",
            "organization_id",
        )

        return queryset


class DocumentClassificationViewSet(viewsets.ModelViewSet):
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
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "name",
            "organization_id",
            "description",
        )
        return queryset


@extend_schema(
    tags=["Bill Order"],
    description="Starts the billing tasks for one shipment.",
    parameters=[
        OpenApiParameter(
            name="shipment_id",
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

    if validation.invoice_billed(invoice_id=invoice_id):
        return Response(
            {"message": "Invoice already billed."},
            status=status.HTTP_400_BAD_REQUEST,
        )

    tasks.bill_invoice_task.delay(user_id=str(request.user.id), invoice_id=invoice_id)
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
def mass_shipments_bill(request: Request) -> Response:
    """
    Mass bill shipments.

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

    tasks.mass_shipments_bill_task.delay(user_id=request.user.id)

    return Response(
        {"message": "Mass Billing task started."}, status=status.HTTP_200_OK
    )


@extend_schema(
    tags=["Transfer to billing"],
    description="Transfer a group of shipments by pro number.",
    request={
        "type": "object",
        "properties": {
            "type": "array",
            "shipment_pros": {
                "type": "string",
                "example": "123456",
            },
        },
    },
    responses={
        (200, "application/json"): {
            "type": "string",
            "message": "Transfer to billing task started.",
        },
        (400, "application/json"): {
            "type": "string",
            "message": "Order Pro(s) is required. Please Try Again.",
        },
    },
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

    This view function expects a POST request containing an `shipment_pros` key in the request data,
    which should be a list of order IDs to be transferred to billing. If no `shipment_pros` key is
    provided, the function will return a response with a 400 status code and an error message.

    If the request data is valid, the function will start a Celery task with the provided order IDs
    and the ID of the currently logged-in user. The task will run in the background and transfer
    the specified order(s) to billing.

    Upon successfully starting the Celery task, the function will return a response with a 200 status
    code and a success message indicating that the transfer task has been started.
    """
    shipment_pros = request.data.get("shipment_pros")

    if not shipment_pros:
        return Response(
            {"message": "Order Pro(s) is required. Please Try Again."},
            status=status.HTTP_400_BAD_REQUEST,
        )

    tasks.transfer_to_billing_task.delay(
        user_id=str(request.user.id), shipment_pros=shipment_pros
    )

    return Response(
        {"message": "Transfer to billing task started."}, status=status.HTTP_200_OK
    )


@extend_schema(
    tags=["shipments Ready to Bill"],
    description="Get a list of shipments ready to bill.",
    request=None,
    responses={
        (200, "application/json"): {
            "type": "array",
            "items": {
                "type": "object",
                "properties": {
                    "id": {"type": "string"},
                    "pro_number": {"type": "string"},
                    "mileage": {"type": "float"},
                    "other_charge_amount": {"type": "decimal"},
                    "freight_charge_amount": {"type": "decimal"},
                    "sub_total": {"type": "decimal"},
                    "customer_name": {"type": "string"},
                    "missing_documents": {"type": "list"},
                    "is_missing_documents": {"type": "boolean"},
                },
            },
        },
    },
)
@api_view(["GET"])
def get_shipments_ready(request: Request) -> Response:
    shipments_ready = selectors.get_billable_shipments(organization=request.user.organization)  # type: ignore
    serializer = serializers.shipmentsReadySerializer(shipments_ready, many=True)
    return Response(
        {
            "results": serializer.data,
        },
        status=status.HTTP_200_OK,
    )


@extend_schema(
    tags=["Untransfer shipments"],
    description="Untransfers a group of shipments by invoice number.",
    request={
        "type": "object",
        "properties": {
            "type": "array",
            "invoice_numbers": {
                "type": "string",
                "example": "123456",
            },
        },
    },
    responses={
        (200, "application/json"): {
            "type": "string",
            "success": "shipments untransferred successfully.",
        },
        (400, "application/json"): {
            "type": "string",
            "error": "No invoice numbers provided.",
        },
        (404, "application/json"): {
            "type": "string",
            "error": "Invoice numbers not found.",
        },
    },
)
@api_view(["POST"])
def untransfer_shipment(request: Request) -> Response:
    invoice_numbers = request.data.get("invoice_numbers")

    if not invoice_numbers:
        return Response(
            {"error": "No invoice numbers provided."},
            status=status.HTTP_400_BAD_REQUEST,
        )

    if isinstance(invoice_numbers, list):
        invoice_numbers_list = invoice_numbers
    else:
        invoice_numbers_list = [invoice_numbers]

    try:
        billing_queues = models.BillingQueue.objects.filter(
            invoice_number__in=invoice_numbers_list
        )
        services.untransfer_shipment_service(
            invoices=billing_queues,
            task_id=str(request.user.id),
            user_id=request.user.id,
        )
        return Response(
            {"success": "Shipments untransferred successfully."},
            status=status.HTTP_200_OK,
        )
    except ObjectDoesNotExist:
        return Response(
            {"error": "Invoice numbers not found."}, status=status.HTTP_404_NOT_FOUND
        )
