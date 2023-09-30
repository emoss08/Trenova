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

from shipment import models
from shipment.selectors import get_shipment_by_id
from utils.models import RatingMethodChoices, StatusChoices


class ShipmentValidator:
    """
    Class to validate the shipment model.
    """

    def __init__(self, *, shipment: models.Shipment):
        self.shipment = shipment
        self.errors: dict[str, Promise] = {}
        self.validate()

    def validate(self) -> None:
        """Validate shipment.

        Validate the shipment model based on the organization's Shipment Control
        and rating method. For example, if the organization has enforce_rev_code
        as `TRUE` and the revenue_code is not entered throw a ValidationError.

        Returns:
            None

        Raises:
            ValidationError: If any of the shipment_control params are true and the associated fields
            are do not fall within the criteria.
        """

        self.validate_rating_method()
        self.validate_shipment_control()
        self.validate_ready_to_bill()
        self.validate_shipment_locations()
        self.validate_duplicate_shipment_bol()
        self.validate_shipment_movement_in_progress()
        self.validate_shipment_movements_completed()
        # self.validate_location_information_cannot_change_once_shipment_completed()
        self.validate_appointment_windows()
        self.validate_per_weight_rating_method()
        self.validate_formula_template()
        self.validate_voided_shipment()
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
            self.shipment.rate_method == RatingMethodChoices.FLAT
            and not self.shipment.freight_charge_amount
        ):
            self.errors["freight_charge_amount"] = _(
                "Freight Rate Method is Flat but Freight Charge Amount is not set. Please try again."
            )

        # Validate 'mileage' is entered if 'rate_method' is 'PER_MILE'
        if (
            self.shipment.rate_method == RatingMethodChoices.PER_MILE
            and not self.shipment.mileage
        ):
            self.errors["mileage"] = _(
                "Rating Method 'PER-MILE' requires Mileage to be set. Please try again."
            )

    def validate_shipment_control(self) -> None:
        """Validate organization Shipment Control.

        Validate that the respective Shipment Control params are being used to validate the
        shipment before it is created or updated. For example, if the organization has
        enforce_origin_destination as `TRUE` and the origin_location and destination_location
        are the same throw a ValidationError.

        Returns:
            None

        Raises:
            ValidationError: If any of the shipment_control params are true and the associated fields
            are do not fall within the criteria.
        """

        # Validate compare origin and destination are not the same.
        if (
            self.shipment.organization.shipment_control.enforce_origin_destination
            and self.shipment.origin_location
            and self.shipment.destination_location
            and self.shipment.origin_location == self.shipment.destination_location
        ):
            self.errors["origin_location"] = _(
                "Origin and Destination locations cannot be the same. Please try again."
            )

        # Validate revenue code is entered if Shipment Control requires it for the organization.
        if (
            self.shipment.organization.shipment_control.enforce_rev_code
            and not self.shipment.revenue_code
        ):
            self.errors["revenue_code"] = _(
                "Revenue code is required. Please try again."
            )

        # Validate commodity is entered if Shipment Control requires it for the organization.
        if (
            self.shipment.organization.shipment_control.enforce_commodity
            and not self.shipment.commodity
        ):
            self.errors["commodity"] = _("Commodity is required. Please try again.")

        # Validate voided comment is entered if Shipment Control requires it for the organization.
        if (
            self.shipment.organization.shipment_control.enforce_voided_comm
            and self.shipment.status == StatusChoices.VOIDED
            and not self.shipment.voided_comm
        ):
            self.errors["voided_comm"] = _(
                "Voided Comment is required. Please try again."
            )

    def validate_ready_to_bill(self) -> None:
        """validate shipment can be marked ready_to_bill

        Validate whether the shipment can be marked `ready_to_bill` based on the status
        of the shipment. For example, if the shipment is currently `IN_PROGRESS` throw a
        ValidateError because shipments can only be marked `ready_to_bill` when status
        is `COMPLETED`

        Returns:
            None

        Raises:
            ValidationError: If the order is marked ready to bill and the status is not
            completed.
        """

        if (
            self.shipment.organization.billing_control.shipment_transfer_criteria
            == "READY_AND_COMPLETED"
            and self.shipment.ready_to_bill
            and self.shipment.status != StatusChoices.COMPLETED
        ):
            self.errors["ready_to_bill"] = _(
                "Cannot mark an order ready to bill if status is not 'COMPLETED'. Please try again."
            )

    def validate_shipment_locations(self) -> None:
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
        if not self.shipment.origin_location and not self.shipment.origin_address:
            self.errors["origin_address"] = _(
                "Origin Location or Address is required. Please try again."
            )

        # Validate that destination_location or destination_address is provided.
        if (
            not self.shipment.destination_location
            and not self.shipment.destination_address
        ):
            self.errors["destination_address"] = _(
                "Destination Location or Address is required. Please try again."
            )

    def validate_duplicate_shipment_bol(self) -> None:
        """Validate duplicate order BOL number.

        Validate that the BOL number is not a duplicate. For example, if the user
        enters a BOL number that is already in use by another order, a ValidationError
        will be thrown and the pro_numbers of the duplicate shipments will be returned to the user.

        Returns:
            None

        Raises:
            ValidationError: If the BOL number is a duplicate. The error message will include the pro_numbers of the duplicate shipments.
        """

        duplicates = self.shipment.organization.shipments.filter(
            bol_number=self.shipment.bol_number,
            status__in=[StatusChoices.NEW, StatusChoices.IN_PROGRESS],
        ).exclude(id=self.shipment.id)

        if (
            self.shipment.organization.shipment_control.check_for_duplicate_bol
            and self.shipment.bol_number
            and self.shipment.status in [StatusChoices.NEW, StatusChoices.IN_PROGRESS]
            and duplicates.exists()
        ):
            pro_numbers = ", ".join(
                [str(shipment.pro_number) for shipment in duplicates]
            )
            self.errors["bol_number"] = _(
                f"Duplicate BOL Number found in shipments with PRO numbers: {pro_numbers}. If this is a new order, "
                f"please change the BOL Number."
            )

    def validate_shipment_movements_completed(self) -> None:
        """Validate that an order cannot be marked as 'COMPLETED' if all of its movements are not 'COMPLETED'.

        This function is used as a validation function in a Django form or model to ensure that if
        an order's status is set to 'COMPLETED', all movements related to the order have a status
        of 'COMPLETED' as well. If not, a validation error is raised.

        Args:
            self: The validation function is called on an instance of a Django form or model.

        Raises:
            ValidationError: If the order status is 'COMPLETED' and not all movements are 'COMPLETED'.
        """
        if self.shipment.status == StatusChoices.COMPLETED and all(
            movement.status != StatusChoices.COMPLETED
            for movement in self.shipment.movements.all()
        ):
            self.errors["status"] = _(
                "Cannot mark order as 'COMPLETED' if all movements are not 'COMPLETED'. Please try again."
            )

    def validate_shipment_movement_in_progress(self) -> None:
        if self.shipment.status == StatusChoices.IN_PROGRESS:
            in_progress_movements = [
                movement
                for movement in self.shipment.movements.all()
                if movement.status == StatusChoices.IN_PROGRESS
            ]

            if not in_progress_movements:
                self.errors["status"] = _(
                    "At least one movement must be `IN PROGRESS` for the order to be marked as `IN PROGRESS`. Please "
                    "try again."
                )

    def validate_location_information_cannot_change_once_shipment_completed(
        self,
    ) -> None:
        """Validate location information in an order cannot be changed once the order is completed.

        Returns:
            None: This function does not return anything.

        Raises:
            ValidationError: If the location information in an order is changed after the order is completed.
        """
        shipment = get_shipment_by_id(shipment_id=self.shipment.id)

        if not shipment:
            return None

        if shipment.status == StatusChoices.COMPLETED:
            location_attributes = [
                ("origin_location", "Origin location"),
                ("destination_location", "Destination location"),
                ("origin_address", "Origin address"),
                ("destination_address", "Destination address"),
                (
                    "origin_appointment_window_start",
                    "Origin appointment window (start)",
                ),
                ("origin_appointment_window_end", "Origin appointment window (end)"),
                (
                    "destination_appointment_window_start",
                    "Destination appointment window (start)",
                ),
                (
                    "destination_appointment_window_end",
                    "Destination appointment window (end)",
                ),
            ]

            for attribute, display_name in location_attributes:
                if getattr(shipment, attribute) != getattr(self.shipment, attribute):
                    self.errors[attribute] = _(
                        f"{display_name} cannot be changed once the order is completed. Please try again."
                    )

    def validate_appointment_windows(self) -> None:
        """Validate origin and destination appointment window ends is not before the start.

        Returns:
            None: This function does not return anything.
        """
        if (
            self.shipment.origin_appointment_window_end
            < self.shipment.origin_appointment_window_start
        ):
            self.errors["origin_appointment_window_end"] = _(
                "Origin appointment window end cannot be before the start. Please try again."
            )

        if (
            self.shipment.destination_appointment_window_end
            < self.shipment.destination_appointment_window_start
        ):
            self.errors["destination_appointment_window_end"] = _(
                "Destination appointment window end cannot be before the start. Please try again."
            )

    def validate_per_weight_rating_method(self) -> None:
        if (
            self.shipment.rate_method == RatingMethodChoices.POUNDS
            and self.shipment.weight < 1
        ):
            self.errors["rate_method"] = _(
                "Weight cannot be 0, and rating method is per weight. Please try again."
            )

    def validate_formula_template(self) -> None:
        if (
            self.shipment.formula_template
            and self.shipment.rate_method != RatingMethodChoices.OTHER
        ):
            self.errors["formula_template"] = _(
                "Formula template can only be used with rating method 'OTHER'. Please try again."
            )

    def validate_voided_shipment(self) -> None:
        shipment = get_shipment_by_id(shipment_id=self.shipment.id)

        if not shipment:
            return None

        if shipment.status == StatusChoices.VOIDED:
            self.errors["status"] = _(
                "Cannot update an order that has been voided. Please contact your administrator."
            )
