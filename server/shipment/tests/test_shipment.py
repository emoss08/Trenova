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

import datetime
import decimal

import pytest
from django.core.exceptions import ValidationError
from django.utils import timezone
from rest_framework.response import Response
from rest_framework.test import APIClient

from accounting.models import RevenueCode
from accounts.models import User
from customer.models import Customer
from dispatch.factories import FleetCodeFactory
from equipment.models import EquipmentType
from equipment.tests.factories import TractorFactory
from location.factories import LocationFactory
from location.models import Location
from movements.models import Movement
from organization.models import BusinessUnit, Organization
from shipment import models, selectors
from shipment.selectors import get_shipment_stops
from shipment.tests.factories import ShipmentFactory
from stops.models import Stop
from utils.models import StatusChoices
from worker.factories import WorkerFactory

pytestmark = pytest.mark.django_db


def test_list(shipment: models.Shipment) -> None:
    """
    Test shipment list
    """
    assert shipment is not None


def test_create(
    organization: Organization,
    business_unit: BusinessUnit,
    shipment_type: models.ShipmentType,
    revenue_code: RevenueCode,
    origin_location: Location,
    destination_location: Location,
    customer: Customer,
    equipment_type: EquipmentType,
    user: User,
) -> None:
    """
    Test shipment Create
    """

    shipment = models.Shipment.objects.create(
        organization=organization,
        business_unit=business_unit,
        shipment_type=shipment_type,
        revenue_code=revenue_code,
        origin_location=origin_location,
        destination_location=destination_location,
        origin_appointment_window_start=timezone.now(),
        origin_appointment_window_end=timezone.now(),
        destination_appointment_window_start=timezone.now()
        + datetime.timedelta(days=2),
        destination_appointment_window_end=timezone.now() + datetime.timedelta(days=2),
        customer=customer,
        freight_charge_amount=100.00,
        equipment_type=equipment_type,
        entered_by=user,
        bol_number="1234567890",
    )

    assert shipment is not None
    assert shipment.shipment_type == shipment_type
    assert shipment.revenue_code == revenue_code
    assert shipment.origin_location == origin_location
    assert shipment.destination_location == destination_location
    assert shipment.customer == customer
    assert shipment.equipment_type == equipment_type
    assert shipment.entered_by == user
    assert shipment.bol_number == "1234567890"


def test_update(shipment: models.Shipment) -> None:
    """
    Test shipment update
    """

    n_shipment = models.Shipment.objects.get(id=shipment.id)

    n_shipment.weight = 20_000
    n_shipment.pieces = 12
    n_shipment.bol_number = "newbolnumber"
    n_shipment.status = "N"
    n_shipment.save()

    assert n_shipment is not None
    assert n_shipment.bol_number == "newbolnumber"
    assert n_shipment.pieces == 12
    assert n_shipment.weight == 20_000


def test_first_stop_completion_puts_shipment_movement_in_progress(
    organization: Organization,
    shipment_type: models.ShipmentType,
    revenue_code: RevenueCode,
    origin_location: Location,
    destination_location: Location,
    customer: Customer,
    equipment_type: EquipmentType,
    user: User,
    business_unit: BusinessUnit,
) -> None:
    """
    Test when the first stop in a movement is completed. The associated movement and shipment are both
    put in progress.
    """
    shipment = models.Shipment.objects.create(
        business_unit=business_unit,
        organization=organization,
        shipment_type=shipment_type,
        revenue_code=revenue_code,
        origin_location=origin_location,
        destination_location=destination_location,
        origin_appointment_window_start=timezone.now(),
        origin_appointment_window_end=timezone.now(),
        destination_appointment_window_start=timezone.now()
        + datetime.timedelta(days=2),
        destination_appointment_window_end=timezone.now() + datetime.timedelta(days=2),
        customer=customer,
        freight_charge_amount=100.00,
        equipment_type=equipment_type,
        entered_by=user,
        bol_number="1234567890",
    )

    fleet_code = FleetCodeFactory()
    worker = WorkerFactory(fleet_code=fleet_code)
    tractor = TractorFactory(primary_worker=worker, fleet_code=fleet_code)
    Movement.objects.filter(shipment=shipment).update(
        tractor=tractor, primary_worker=worker
    )
    shipment_movement = Movement.objects.get(shipment=shipment)

    # Act: Complete the first stop in the movement
    stop_1 = Stop.objects.get(movement=shipment_movement, sequence=1)
    stop_1.arrival_time = timezone.now() - datetime.timedelta(hours=1)
    stop_1.departure_time = timezone.now()
    stop_1.save()

    shipment_movement.refresh_from_db()

    # Assert: Check if the first stop is completed and the movement is in progress
    assert stop_1.status == StatusChoices.COMPLETED
    assert shipment_movement.status == StatusChoices.IN_PROGRESS

    # Act: Complete the second stop in the movement
    stop_2 = Stop.objects.get(movement=shipment_movement, sequence=2)
    stop_2.arrival_time = timezone.now() + datetime.timedelta(hours=1)
    stop_2.departure_time = timezone.now() + datetime.timedelta(hours=2)
    stop_2.save()

    shipment_movement.refresh_from_db()

    # Assert: Check if the second stop is completed and the movement is completed
    assert stop_2.status == StatusChoices.COMPLETED
    assert shipment_movement.status == StatusChoices.COMPLETED
    assert shipment_movement.shipment.status == StatusChoices.COMPLETED


def test_create_initial_movement_signal(
    organization: Organization,
    shipment_type: models.ShipmentType,
    revenue_code: RevenueCode,
    origin_location: Location,
    destination_location: Location,
    customer: Customer,
    equipment_type: EquipmentType,
    user: User,
    business_unit: BusinessUnit,
) -> None:
    """
    Test create initial movement hook when shipment is created.
    """

    shipment = models.Shipment.objects.create(
        business_unit=business_unit,
        organization=organization,
        shipment_type=shipment_type,
        revenue_code=revenue_code,
        origin_location=origin_location,
        destination_location=destination_location,
        origin_appointment_window_start=timezone.now(),
        origin_appointment_window_end=timezone.now(),
        destination_appointment_window_start=timezone.now()
        + datetime.timedelta(days=2),
        destination_appointment_window_end=timezone.now() + datetime.timedelta(days=2),
        customer=customer,
        freight_charge_amount=100.00,
        equipment_type=equipment_type,
        entered_by=user,
        bol_number="1234567890",
    )

    movement_count = Movement.objects.filter(shipment=shipment).count()

    assert movement_count == 1


def test_get(api_client: APIClient) -> None:
    """
    Test get Reason Code
    """
    response = api_client.get("/api/shipments/")
    assert response.status_code == 200


def test_get_by_id(
    api_client: APIClient,
    shipment_api: Response,
    shipment_type: models.ShipmentType,
    revenue_code: RevenueCode,
    origin_location: Location,
    destination_location: Location,
    customer: Customer,
    equipment_type: EquipmentType,
    user: User,
) -> None:
    """
    Test get shipment by id
    """
    response = api_client.get(f"/api/shipments/{shipment_api.data['id']}/")
    assert response.status_code == 200
    assert response.data["shipment_type"] == shipment_type.id
    assert response.data["revenue_code"] == revenue_code.id
    assert response.data["origin_location"] == origin_location.id
    assert response.data["origin_address"] == origin_location.get_address_combination
    assert response.data["destination_location"] == destination_location.id
    assert (
        response.data["destination_address"]
        == destination_location.get_address_combination
    )
    assert response.data["customer"] == customer.id
    assert response.data["equipment_type"] == equipment_type.id
    assert response.data["entered_by"] == user.id
    assert response.data["bol_number"] == "newbol"


def test_put(
    api_client: APIClient,
    shipment_api: Response,
    origin_location: Location,
    destination_location: Location,
    shipment_type: models.ShipmentType,
    revenue_code: RevenueCode,
    customer: Customer,
    equipment_type: EquipmentType,
    user: User,
) -> None:
    """
    Test put shipment
    """
    response = api_client.put(
        f"/api/shipments/{shipment_api.data['id']}/",
        {
            "origin_location": f"{origin_location.id}",
            "destination_location": f"{destination_location.id}",
            "shipment_type": f"{shipment_type.id}",
            "revenue_code": f"{revenue_code.id}",
            "origin_appointment_window_start": f"{timezone.now()}",
            "origin_appointment_window_end": f"{timezone.now()}",
            "destination_appointment_window_start": f"{timezone.now() + datetime.timedelta(days=2)}",
            "destination_appointment_window_end": f"{timezone.now() + datetime.timedelta(days=2)}",
            "customer": f"{customer.id}",
            "equipment_type": f"{equipment_type.id}",
            "entered_by": f"{user.id}",
            "bol_number": "anotherbol",
        },
    )
    assert response.status_code == 200
    assert response.data["origin_location"] == origin_location.id
    assert response.data["origin_address"] == origin_location.get_address_combination
    assert response.data["destination_location"] == destination_location.id
    assert (
        response.data["destination_address"]
        == destination_location.get_address_combination
    )
    assert response.data["shipment_type"] == shipment_type.id
    assert response.data["revenue_code"] == revenue_code.id
    assert response.data["customer"] == customer.id
    assert response.data["equipment_type"] == equipment_type.id
    assert response.data["entered_by"] == user.id
    assert response.data["bol_number"] == "anotherbol"


def test_patch(
    api_client: APIClient,
    shipment_api: Response,
) -> None:
    """
    Test patch shipment
    """
    response = api_client.patch(
        f"/api/shipments/{shipment_api.data['id']}/",
        {
            "bol_number": "patchedbol",
        },
    )

    assert response.status_code == 200
    assert response.data["bol_number"] == "patchedbol"


def test_flat_method_requires_freight_charge_amount() -> None:
    """
    Test ValidationError is thrown when the shipment has `FLAT` rating method
    and the `freight_charge_amount` is None
    """
    with pytest.raises(ValidationError) as excinfo:
        ShipmentFactory(rate_method="F", freight_charge_amount=None)

    assert excinfo.value.message_dict["freight_charge_amount"] == [
        "Freight Rate Method is Flat but Freight Charge Amount is not set. Please try again."
    ]


def test_per_mile_requires_mileage() -> None:
    """
    Test ValidationError is thrown when the shipment has `PER-MILE` rating method
    and the `mileage` is None
    """
    with pytest.raises(ValidationError) as excinfo:
        ShipmentFactory(rate_method="PM", mileage=None)

    assert excinfo.value.message_dict["mileage"] == [
        "Rating Method 'PER-MILE' requires Mileage to be set. Please try again."
    ]


def test_shipment_origin_destination_location_cannot_be_the_same() -> None:
    """
    Test ValidationError is thrown when the shipment `origin_location and
    `destination_location` is the same.
    """
    shipment = ShipmentFactory()
    shipment.organization.shipment_control.enforce_origin_destination = True

    location = LocationFactory()

    with pytest.raises(ValidationError) as excinfo:
        shipment.origin_location = location
        shipment.destination_location = location
        shipment.save()

    assert excinfo.value.message_dict["origin_location"] == [
        "Origin and Destination locations cannot be the same. Please try again."
    ]


def test_shipment_revenue_code_is_enforced() -> None:
    """
    Test ValidationError is thrown if the `shipment_control` has `enforce_rev_code`
    set as `TRUE`
    """
    shipment = ShipmentFactory()
    shipment.organization.shipment_control.enforce_rev_code = True

    with pytest.raises(ValidationError) as excinfo:
        shipment.revenue_code = None
        shipment.save()

    assert excinfo.value.message_dict["revenue_code"] == [
        "Revenue code is required. Please try again."
    ]


def test_shipment_commodity_is_enforced() -> None:
    """
    Test ValidationError is thrown if the `shipment_control` has `enforce_commodity`
    set as `TRUE`
    """
    shipment = ShipmentFactory()
    shipment.organization.shipment_control.enforce_commodity = True

    with pytest.raises(ValidationError) as excinfo:
        shipment.revenue_code = None
        shipment.save()

    assert excinfo.value.message_dict["commodity"] == [
        "Commodity is required. Please try again."
    ]


def test_shipment_must_be_completed_to_bill() -> None:
    """
    Test ValidationError is thrown if the shipment status is not `COMPLETED`
    and `ready_to_bill` is marked `TRUE`
    """
    with pytest.raises(ValidationError) as excinfo:
        ShipmentFactory(status="N", ready_to_bill=True)

    assert excinfo.value.message_dict["ready_to_bill"] == [
        "Cannot mark an shipment ready to bill if status is not 'COMPLETED'. Please try again."
    ]


def test_shipment_origin_location_or_address_is_required() -> None:
    """
    Test ValidationError is thrown if the shipment `origin_location` and
    `origin_address` is blank
    """
    with pytest.raises(ValidationError) as excinfo:
        ShipmentFactory(
            origin_location=None,
            origin_address=None,
        )

    assert excinfo.value.message_dict["origin_address"] == [
        "Origin Location or Address is required. Please try again."
    ]


def test_shipment_destination_location_or_address_is_required() -> None:
    """
    Test ValidationError is thrown if the shipment `destination_location` and
    `destination_address` is blank
    """
    with pytest.raises(ValidationError) as excinfo:
        ShipmentFactory(
            destination_location=None,
            destination_address=None,
        )

    assert excinfo.value.message_dict["destination_address"] == [
        "Destination Location or Address is required. Please try again."
    ]


def test_remove_shipments_validation(
    shipment: models.Shipment, organization: Organization
) -> None:
    """
    Test ValidationError is thrown if the stop in an shipment is being deleted,
    and shipment_control does not allow it.
    """

    with pytest.raises(ValidationError) as excinfo:
        for stop in get_shipment_stops(shipment=shipment):
            stop.delete()

    assert excinfo.value.message_dict["ref_num"] == [
        "Organization does not allow Stop removal. Please contact your administrator."
    ]


def test_set_shipment_pro_number_signal(shipment: models.Shipment) -> None:
    """
    Test set_shipment_pro_number `pre_save` signal.
    """

    assert shipment.pro_number


def test_shipment_pro_number_increments(
    shipment: models.Shipment, organization: Organization
) -> None:
    """
    Test shipment pro_number increments by one.
    """
    today = datetime.datetime.now().strftime("%y%m%d")

    shipment_2 = ShipmentFactory(organization=organization)

    assert shipment.pro_number == f"{today}-0001"
    assert shipment_2.pro_number == f"{today}-0002"


def test_set_total_piece_and_weight_signal(
    shipment: models.Shipment,
) -> None:
    """
    Test set_total_piece_and_weight `pre_save` signal.
    """
    movements = selectors.get_shipment_movements(shipment=shipment)
    stops = selectors.get_shipment_stops(shipment=shipment)

    fleet_code = FleetCodeFactory()
    worker = WorkerFactory(fleet_code=fleet_code)
    tractor = TractorFactory(primary_worker=worker, fleet_code=fleet_code)

    for movement in movements:
        movement.worker = worker
        movement.tractor = tractor
        movement.save()

    for stop in stops:
        if stop.sequence == 2:
            stop.appointment_time_window_start = timezone.now() + datetime.timedelta(
                days=1
            )
            stop.appointment_time_window_end = timezone.now() + datetime.timedelta(
                days=1
            )

        stop.arrival_time = timezone.now()
        stop.departure_time = timezone.now() + datetime.timedelta(hours=1)
        stop.pieces = 100
        stop.weight = 100
        stop.save()

    shipment.refresh_from_db()


def test_validate_origin_appointment_window_start_not_after_end(
    shipment: models.Shipment,
) -> None:
    """Test origin appointment window end is not before the start of the window..

    Args:
        shipment(models.Shipment): shipmentobject

    Returns:
        None: This function does not return anything.
    """
    shipment.origin_appointment_window_start = timezone.now()
    shipment.origin_appointment_window_end = timezone.now() - datetime.timedelta(days=1)
    with pytest.raises(ValidationError) as excinfo:
        shipment.clean()

    assert excinfo.value.message_dict["origin_appointment_window_end"] == [
        "Origin appointment window end cannot be before the start. Please try again."
    ]


def test_validate_destination_appointment_window_start_not_after_end(
    shipment: models.Shipment,
) -> None:
    """Test destination appointment window end is not before the start of the window.

    Args:
        shipment(models.Shipment): shipment object.

    Returns:
        None: This function does not return anything.
    """
    shipment.destination_appointment_window_start = timezone.now()
    shipment.destination_appointment_window_end = timezone.now() - datetime.timedelta(
        days=1
    )
    with pytest.raises(ValidationError) as excinfo:
        shipment.clean()

    assert excinfo.value.message_dict["destination_appointment_window_end"] == [
        "Destination appointment window end cannot be before the start. Please try again."
    ]


def test_validate_appointment_window_against_customer_delivery_slots(
    shipment, delivery_slot
):
    # Set delivery slot to a Sunday
    delivery_slot.customer = shipment.customer
    delivery_slot.day_of_week = 6  # Sunday
    delivery_slot.start_time = datetime.time(9, 0)  # 9:00 AM
    delivery_slot.end_time = datetime.time(17, 0)  # 5:00 PM
    delivery_slot.location = shipment.destination_location
    delivery_slot.save()

    delivery_slot.refresh_from_db()

    # Set shipment's appointment window to a time not allowed by the customer
    sunday_date = next_weekday(timezone.now(), 6)  # Next Sunday
    shipment.destination_appointment_window_start = datetime.datetime.combine(
        sunday_date.date(), datetime.time(18, 0), tzinfo=datetime.UTC
    )  # 6:00 PM
    shipment.destination_appointment_window_end = datetime.datetime.combine(
        sunday_date.date(), datetime.time(19, 0), tzinfo=datetime.UTC
    )  # 7:00 PM

    with pytest.raises(ValidationError) as excinfo:
        shipment.save()

    assert excinfo.value.message_dict["origin_appointment_window_start"] == [
        "The chosen appointment window for the location is not allowed by the customer. Please try again."
    ]


def next_weekday(d, weekday):
    days_ahead = weekday - d.weekday()
    if days_ahead <= 0:  # Target day already happened this week
        days_ahead += 7
    return d + datetime.timedelta(days_ahead)


def test_calculate_shipment_per_pound_total(shipment: models.Shipment) -> None:
    """Test calculate shipment per pound calculation.

    Args:
        shipment (models.Shipment): shipment object.

    Returns:
        None: This function does not return anything.
    """
    shipment.weight = 43000  # 43,000 lbs
    shipment.rate_method = "PP"
    shipment.freight_charge_amount = 0.5  # $0.50 per pound
    shipment.save()
    shipment.refresh_from_db()

    assert shipment.sub_total == 21500.0000


def test_calculate_shipment_per_pound_exception(shipment: models.Shipment) -> None:
    """Test ValidationError thrown when weight on shipment is less than 1.

    Args:
        shipment (models.Shipment): shipment object.

    Returns:
        None: this function does not return anything.
    """
    shipment.weight = 0
    shipment.rate_method = "PP"

    with pytest.raises(ValidationError) as excinfo:
        shipment.save()

    assert excinfo.value.message_dict["rate_method"] == [
        "Weight cannot be 0, and rating method is per weight. Please try again."
    ]


def test_calculate_shipment_flat_total(shipment: models.Shipment) -> None:
    """Test calculate shipment ``flat`` fee calculation

    Args:
        shipment (models.Shipment): shipment object.

    Returns:
        None: this function does not return anything.
    """

    shipment.rate_method = "F"
    shipment.freight_charge_amount = 1000.00

    shipment.save()

    assert shipment.sub_total == 1000.00


def test_calculate_shipment_per_mile_total(shipment: models.Shipment) -> None:
    """Test calculate shipment ``PER_MILE`` rate method calculation.

    Args:
        shipment (models.Shipment): shipment object.

    Returns:
        None: This function does not return anything.
    """

    shipment.rate_method = "PM"
    shipment.mileage = 100
    shipment.freight_charge_amount = 10.00

    shipment.save()

    assert shipment.sub_total == 1000.00


def test_calculate_shipment_per_stop_total(shipment: models.Shipment) -> None:
    """Test calculate shipment ``PER_STOP`` rate method calculation.

    Args:
        shipment (models.Shipment): shipment object.

    Returns:
        None: This function does not return anything.
    """

    shipment.freight_charge_amount = 100.00
    shipment.rate_method = "PS"
    shipment.save()

    assert shipment.sub_total == 200.00


def test_calculate_shipment_other_total_with_formula(
    organization: Organization, business_unit: BusinessUnit
) -> None:
    """Test calculate shipment total using formula template.

    Args:
        organization(Organization): Organization object.
        business_unit(BusinessUnit): BusinessUnit object.

    Returns:
        None: This function does not return anything.
    """

    formula_template = models.FormulaTemplate.objects.create(
        organization=organization,
        business_unit=business_unit,
        name="Refrigerated Shipment Formula",
        formula_text="freight_charge * rating_units",
        description="Basic Rate calculation for refrigerated shipments",
        template_type="REFRIGERATED",
    )

    shipment = ShipmentFactory(
        rate_method="O",
        formula_template=formula_template,
        freight_charge_amount=100.00,
        rating_units=5,
    )

    assert shipment.sub_total == decimal.Decimal("500.00")


def test_calculate_shipment_other_total(shipment: models.Shipment) -> None:
    """Test calculate shipment total without using formula template

    Defaults to shipment.freight_charge_amount * shipment.rating_units + shipment.other_charge_amount

    Args:
        shipment (models.Shipment): shipment object.

    Returns:
        None: This function does not return anything.
    """
    shipment.rate_method = "O"
    shipment.freight_charge_amount = 100.00
    shipment.rating_units = 5
    shipment.other_charge_amount = 100.00

    shipment.save()

    assert shipment.sub_total == decimal.Decimal("600.00")


def test_temperature_differential(shipment: models.Shipment) -> None:
    """Test calculate shipment ``temperature_differential`` function.

    Args:
        shipment (models.Shipment): shipment object.

    Returns:
        None: This function does not return anything.
    """

    shipment.temperature_min = 10
    shipment.temperature_max = 60
    shipment.save()

    assert shipment.temperature_differential == 50


def test_formula_template_validation(
    organization: Organization, business_unit: BusinessUnit, shipment: models.Shipment
) -> None:
    """Test ValidationError is thrown when formula_template is populated ,but
    rate_method is not set to OTHER.

    Args:
        organization (models.Organization): Organization object.
        business_unit (models.BusinessUnit): BusinessUnit object.
        shipment (models.Shipment): shipment object.

    Returns:
        None: This function does not return anything.
    """

    formula_template = models.FormulaTemplate.objects.create(
        organization=organization,
        business_unit=business_unit,
        name="Refrigerated Shipment Formula",
        formula_text="(freight_charge + other_charge + temperature_differential * equipment_cost_per_mile) * mileage",
        description="Formula for refrigerated shipments considering temperature differential",
        template_type="REFRIGERATED",
    )

    shipment.rate_method = "F"
    shipment.formula_template = formula_template

    with pytest.raises(ValidationError) as excinfo:
        shipment.clean()

    assert excinfo.value.message_dict["formula_template"] == [
        "Formula template can only be used with rating method 'OTHER'. Please try again."
    ]


def test_validate_formula_variables(
    organization: Organization, business_unit: BusinessUnit
) -> None:
    """Test ValidationError is thrown when invalid variables are used in ``formula_text``

    Args:
        organization(Organization): Organization object.
        business_unit(BusinessUnit): BusinessUnit object.

    Returns:
        None: This function does not return anything.
    """

    with pytest.raises(ValidationError) as excinfo:
        models.FormulaTemplate.objects.create(
            organization=organization,
            business_unit=business_unit,
            name="Refrigerated Shipment Formula",
            formula_text="(bad + equipment_cost + temperature_differential * temp_factor) * mileage",
            description="Formula for refrigerated shipments considering temperature differential",
            template_type="REFRIGERATED",
        )

    assert excinfo.value.message_dict["formula_text"] == [
        "Formula template contains invalid variables: bad, equipment_cost, temp_factor. Please try again."
    ]
