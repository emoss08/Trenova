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
    Validation Class for validating Order Model
    """

    def __init__(self, *, order, organization, order_control):
        self.order = order
        self.organization = organization
        self.order_control = order_control

    def validate(self) -> None:
        """Validate the order

        Returns:
            None

        Raises:
            ValidationError: If the order is invalid
        """
        self.validate_freight_rate_method()
        self.validate_revenue_code()
        self.validate_ready_to_bill()
        self.validate_per_mile_rate_method()
        self.validate_compare_origin_destination()
        self.validate_location()

    def validate_freight_rate_method(self) -> None:
        """Validate the freight charge amount

        If the rate method is flat, the freight charge
        amount must be set.

        Returns:
            None

        Raises:
            ValidationError: If the freight charge amount is not set
        """
        if self.order.rate_method == "F" and self.order.freight_charge_amount is None:
            raise ValidationError(
                {
                    "rate_method": ValidationError(
                        _(
                            "Freight Charge Amount is required for flat rating method. Please try again."
                        ),
                        code="invalid",
                    )
                }
            )

    def validate_revenue_code(self) -> None:
        """Validate the revenue code

        Returns:
            None

        Raises:
            ValidationError: If the revenue code is not set
        """
        order_control = self.order_control.objects.get(organization=self.organization)

        if order_control.enforce_rev_code and not self.order.revenue_code:
            raise ValidationError(
                {
                    "revenue_code": ValidationError(
                        _("Revenue Code is required. Please try again."),
                        code="invalid",
                    )
                }
            )

    def validate_ready_to_bill(self) -> None:
        """Validate the order is ready to be billed

        Order must be marked completed before it can be marked
        ready to bill.

        Returns:
            None

        Raises:
            ValidationError: If the order is not completed
        """

        if self.order.ready_to_bill and self.order.status != StatusChoices.COMPLETED:
            raise ValidationError(
                {
                    "ready_to_bill": _(
                        "Cannot mark an order ready to bill if the order status"
                        " is not complete. Please try again."
                    ),
                },
                code="invalid",
            )

    def validate_per_mile_rate_method(self) -> None:
        """Validate the per mile rate method

        If the rate method is per mile, the mileage must be set.

        Returns:
            None

        Raises:
            ValidationError: If the mileage is not set
        """

        if (
            self.order.rate_method == RatingMethodChoices.PER_MILE
            and self.order.mileage is None
        ):
            raise ValidationError(
                {
                    "rate_method": _(
                        "Mileage is required for per mile rating method. Please try again."
                    ),
                },
                code="invalid",
            )

    def validate_compare_origin_destination(self) -> None:
        """Validate the origin and destination locations
        Returns:
            None
        Raises:
            ValidationError: If the origin and destination locations are the same
        """

        order_control = self.order_control.objects.get(organization=self.organization)
        if (
            self.order.origin_location
            and order_control.enforce_origin_destination
            and self.order.origin_location == self.order.destination_location
        ):
            raise ValidationError(
                {
                    "origin_location": _(
                        "Origin and Destination cannot be the same. Please try again."
                    ),
                }
            )

    def validate_location(self) -> None:
        """Validate location is provided.

        If origin_location and destination_location are not provided,
        then require the address field for both origin and destination.

        Returns:
            None

        Raises:
            ValidationError: If the location is not provided and the address is not provided.
        """

        if not self.order.origin_location and not self.order.origin_address:
            raise ValidationError(
                {
                    "origin_address": _(
                        "Origin Location or Address is required. Please try again."
                    ),
                },
                code="invalid",
            )

        if not self.order.destination_location and not self.order.destination_address:
            raise ValidationError(
                {
                    "destination_address": _(
                        "Destination Location or Address is required. Please try again."
                    ),
                },
                code="invalid",
            )
