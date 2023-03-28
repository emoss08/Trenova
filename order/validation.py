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

from django.core.exceptions import ValidationError
from django.utils.functional import Promise
from django.utils.translation import gettext_lazy as _

from utils.models import RatingMethodChoices, StatusChoices


class OrderValidation:
    """
    Class to validate the order model.
    """

    def __init__(self, *, order):
        self.order = order
        self.errors: dict[str, Promise] = {}
        self.validate()

    def validate(self) -> None:
        """Validate order.

        Validate the order model based on the organization's order control
        and rating method. For example, if the organization has enforce_rev_code
        as `TRUE` and the revenue_code is not entered throw a ValidationError.

        Returns:
            None

        Raises:
            ValidationError: If any of the order_control params are true and the associated fields
            are do not fall within the criteria.
        """

        self.validate_rating_method()
        self.validate_order_control()
        self.validate_ready_to_bill()
        self.validate_order_locations()
        self.validate_duplicate_order_bol()
        self.validate_order_movement_in_progress()
        self.validate_order_movements_completed()

        if self.errors:
            raise ValidationError(self.errors)

    def validate_rating_method(self) -> None:
        """Validate rating method.

        Validate that the given rating method follows the proper exceptions.
        For example, if rate_method is flat and no freight_charge_amount is given
        throw a ValidationError.

        Returns:
            None

        Raises:
            ValidationError: If the associated field to the rating method
            is not valid.
        """

        # Validate 'freight_charge_amount' is entered if 'rate_method' is 'FLAT'
        if (
            self.order.rate_method == RatingMethodChoices.FLAT
            and not self.order.freight_charge_amount
        ):
            self.errors["freight_charge_amount"] = _(
                "Freight Rate Method is Flat but Freight Charge Amount is not set. Please try again."
            )

        # Validate 'mileage' is entered if 'rate_method' is 'PER_MILE'
        if (
            self.order.rate_method == RatingMethodChoices.PER_MILE
            and not self.order.mileage
        ):
            self.errors["mileage"] = _(
                "Rating Method 'PER-MILE' requires Mileage to be set. Please try again."
            )

    def validate_order_control(self) -> None:
        """Validate organization order control.

        Validate that the respective order control params are being used to validate the
        order before it is created or updated. For example, if the organization has
        enforce_origin_destination as `TRUE` and the origin_location and destination_location
        are the same throw a ValidationError.

        Returns:
            None

        Raises:
            ValidationError: If any of the order_control params are true and the associated fields
            are do not fall within the criteria.
        """

        # Validate compare origin and destination are not the same.
        if (
            self.order.organization.order_control.enforce_origin_destination
            and self.order.origin_location
            and self.order.destination_location
            and self.order.origin_location == self.order.destination_location
        ):
            self.errors["origin_location"] = _(
                "Origin and Destination locations cannot be the same. Please try again."
            )

        # Validate revenue code is entered if Order Control requires it for the organization.
        if (
            self.order.organization.order_control.enforce_rev_code
            and not self.order.revenue_code
        ):
            self.errors["revenue_code"] = _(
                "Revenue code is required. Please try again."
            )

        # Validate commodity is entered if Order Control requires it for the organization.
        if (
            self.order.organization.order_control.enforce_commodity
            and not self.order.commodity
        ):
            self.errors["commodity"] = _("Commodity is required. Please try again.")

        # Validate voided comment is entered if Order Control requires it for the organization.
        if (
            self.order.organization.order_control.enforce_voided_comm
            and self.order.status == StatusChoices.VOIDED
            and not self.order.voided_comm
        ):
            self.errors["voided_comm"] = _(
                "Voided Comment is required. Please try again."
            )

    def validate_ready_to_bill(self) -> None:
        """validate order can be marked ready_to_bill

        Validate whether the order can be marked `ready_to_bill` based on the status
        of the order. For example, if the order is currently `IN_PROGRESS` throw a
        ValidateError because orders can only be marked `ready_to_bill` when status
        is `COMPLETED`

        Returns:
            None

        Raises:
            ValidationError: If the order is marked ready to bill and the status is not
            completed.
        """

        if (
            self.order.organization.billing_control.order_transfer_criteria
            == "READY_AND_COMPLETED"
            and self.order.ready_to_bill
            and self.order.status != StatusChoices.COMPLETED
        ):
            self.errors["ready_to_bill"] = _(
                "Cannot mark an order ready to bill if status is not 'COMPLETED'. Please try again."
            )

    def validate_order_locations(self) -> None:
        """Validate order location is entered.

        Validate that either the `location` foreign key field has input or the
        `location_address` field has input for both origin and destination.For
        example, if user creates the order without origin_location and origin_address
        a ValidationError will be thrown letting the user know that either enter origin_location
        or origin_address. The same rules apply for the destination_location
        and destination_address

        Returns:
            None

        Raises:
            ValidationError: If the location foreign key and location address is blank.
        """

        # Validate that origin_location or origin_address is provided.
        if not self.order.origin_location and not self.order.origin_address:
            self.errors["origin_address"] = _(
                "Origin Location or Address is required. Please try again."
            )

        # Validate that destination_location or destination_address is provided.
        if not self.order.destination_location and not self.order.destination_address:
            self.errors["destination_address"] = _(
                "Destination Location or Address is required. Please try again."
            )

    def validate_duplicate_order_bol(self) -> None:
        """Validate duplicate order BOL number.

        Validate that the BOL number is not a duplicate. For example, if the user
        enters a BOL number that is already in use by another order, a ValidationError
        will be thrown and the pro_numbers of the duplicate orders will be returned to the user.

        Returns:
            None

        Raises:
            ValidationError: If the BOL number is a duplicate. The error message will include the pro_numbers of the duplicate orders.
        """

        duplicates = self.order.organization.orders.filter(
            bol_number=self.order.bol_number,
            status__in=[StatusChoices.NEW, StatusChoices.IN_PROGRESS],
        ).exclude(id=self.order.id)

        if (
            self.order.organization.order_control.check_for_duplicate_bol
            and self.order.bol_number
            and self.order.status in [StatusChoices.NEW, StatusChoices.IN_PROGRESS]
            and duplicates.exists()
        ):
            pro_numbers = ", ".join([str(order.pro_number) for order in duplicates])
            self.errors["bol_number"] = _(
                f"Duplicate BOL Number found in orders with PRO numbers: {pro_numbers}. If this is a new order, please change the BOL Number."
            )

    def validate_order_movements_completed(self) -> None:
        """Validate that an order cannot be marked as 'COMPLETED' if all of its movements are not 'COMPLETED'.

        This function is used as a validation function in a Django form or model to ensure that if
        an order's status is set to 'COMPLETED', all movements related to the order have a status
        of 'COMPLETED' as well. If not, a validation error is raised.

        Args:
            self: The validation function is called on an instance of a Django form or model.

        Raises:
            ValidationError: If the order status is 'COMPLETED' and not all movements are 'COMPLETED'.
        """
        if self.order.status == StatusChoices.COMPLETED and all(
            movement.status != StatusChoices.COMPLETED
            for movement in self.order.movements.all()
        ):
            self.errors["status"] = _(
                "Cannot mark order as 'COMPLETED' if all movements are not 'COMPLETED'. Please try again."
            )

    def validate_order_movement_in_progress(self) -> None:
        if self.order.status == StatusChoices.IN_PROGRESS:
            in_progress_movements = [
                movement for movement in self.order.movements.all() if movement.status == StatusChoices.IN_PROGRESS
            ]

            if not in_progress_movements:
                self.errors["status"] = _(
                    "At least one movement must be `IN PROGRESS` for the order to be marked as `IN PROGRESS`. Please try again."
                )
