"""
COPYRIGHT 2022 MONTA

This file is part of Monta.

Monta is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

Monta is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with Monta.  If not, see <https://www.gnu.org/licenses/>.
"""

from django.core.exceptions import ValidationError
from django.utils.translation import gettext_lazy as _

from utils.models import RatingMethodChoices, StatusChoices


class OrderValidation:
    """
    Class to validate the order model.
    """

    def __init__(self, *, order):
        self.order = order
        self.validate_rating_method()
        self.validate_order_control()
        self.validate_ready_to_bill()
        self.validate_order_locations()

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
            raise ValidationError(
                {
                    "freight_charge_amount": _(
                        "Freight Rate Method is Flat but Freight Charge Amount is not set. Please try again."
                    )
                },
                code="invalid",
            )

        # Validate 'mileage' is entered if 'rate_method' is 'PER_MILE'
        if (
            self.order.rate_method == RatingMethodChoices.PER_MILE
            and not self.order.mileage
        ):
            raise ValidationError(
                {
                    "mileage": _(
                        "Rating Method 'PER-MILE' requires Mileage to be set. Please try again."
                    )
                },
                code="invalid",
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
            raise ValidationError(
                {
                    "origin_location": _(
                        "Origin and Destination locations cannot be the same. Please try again."
                    )
                },
                code="invalid",
            )

        # Validate revenue code is entered if Order Control requires it for the organization.
        if (
            self.order.organization.order_control.enforce_rev_code
            and not self.order.revenue_code
        ):
            raise ValidationError(
                {"revenue_code": _("Revenue code is required. Please try again.")},
                code="invalid",
            )

        # Validate commodity is entered if Order Control requires it for the organization.
        if (
            self.order.organization.order_control.enforce_commodity
            and not self.order.commodity
        ):
            raise ValidationError(
                {"commodity": _("Commodity is required. Please try again.")},
                code="invalid",
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

        # Validate order not marked 'ready_to_bill' if 'status' is not COMPLETED
        if self.order.ready_to_bill and self.order.status != StatusChoices.COMPLETED:
            raise ValidationError(
                {
                    "ready_to_bill": _(
                        "Cannot mark an order ready to bill if status is not 'COMPLETED'. Please try again."
                    )
                },
                code="invalid",
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
            raise ValidationError(
                {
                    "origin_address": _(
                        "Origin Location or Address is required. Please try again."
                    ),
                },
                code="invalid",
            )

        # Validate that destination_location or destination_address is provided.
        if not self.order.destination_location and not self.order.destination_address:
            raise ValidationError(
                {
                    "destination_address": _(
                        "Destination Location or Address is required. Please try again."
                    ),
                },
                code="invalid",
            )
