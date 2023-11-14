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

from datetime import timedelta

import pytest
from django.core.exceptions import ValidationError
from django.utils import timezone
from rest_framework.response import Response
from rest_framework.test import APIClient

from commodities.factories import CommodityFactory, HazardousMaterialFactory
from dispatch.factories import FleetCodeFactory
from equipment.models import Tractor
from equipment.tests.factories import TractorFactory
from movements import models, services
from movements.tests.factories import MovementFactory
from organization.models import BusinessUnit, Organization
from shipment.models import Shipment
from shipment.tests.factories import ShipmentFactory
from worker.factories import WorkerFactory
from worker.models import Worker

pytestmark = pytest.mark.django_db


def test_create(
    worker: Worker,
    tractor: Tractor,
    organization: Organization,
    shipment: Shipment,
    business_unit: BusinessUnit,
) -> None:
    """
    Test Movement Create
    """

    fleet_code = FleetCodeFactory()

    tractor.fleet_code = fleet_code
    tractor.save()

    worker.fleet_code = fleet_code
    worker.save()

    movement = models.Movement.objects.create(
        organization=organization,
        business_unit=business_unit,
        shipment=shipment,
        tractor=tractor,
        primary_worker=worker,
    )

    assert movement is not None
    assert movement.shipment == shipment
    assert movement.tractor == tractor
    assert movement.primary_worker == worker


def test_update(movement: models.Movement, tractor: Tractor) -> None:
    """
    Test Movement Update
    """

    add_movement = models.Movement.objects.get(id=movement.id)

    add_movement.tractor = tractor
    add_movement.save()

    assert add_movement is not None
    assert add_movement.tractor == tractor


def test_initial_stop_creation_hook(
    worker: Worker,
    tractor: Tractor,
    organization: Organization,
    business_unit: BusinessUnit,
) -> None:
    """
    Test that an initial stop is created when a movement is created.
    """
    shipment = ShipmentFactory(
        origin_appointment_window_start=timezone.now(),
        origin_appointment_window_end=timezone.now(),
        destination_appointment_window_start=timezone.now() + timedelta(days=2),
        destination_appointment_window_end=timezone.now() + timedelta(days=2),
    )

    movement = models.Movement.objects.create(
        organization=organization,
        business_unit=business_unit,
        shipment=shipment,
        tractor=tractor,
        primary_worker=worker,
    )

    services.create_initial_stops(movement=movement, shipment=shipment)

    assert movement.stops.count() == 2


def test_generate_reference_number(movement: models.Movement) -> None:
    """Test that a movement reference number is created when a movement is created.

    Args:
        movement (models.Movement): Movement instance

    Returns:
        None: This function does not return anything.
    """
    assert movement.ref_num is not None


def test_get(api_client: APIClient) -> None:
    """
    Test get Movement
    """
    response = api_client.get("/api/movements/")
    assert response.status_code == 200


def test_get_by_id(
    api_client: APIClient,
    movement_api: Response,
    shipment: Shipment,
    worker: Worker,
    tractor: Tractor,
) -> None:
    """
    Test get Movement by ID
    """

    response = api_client.get(f"/api/movements/{movement_api.data['id']}/")

    assert response.status_code == 200
    assert response.data is not None
    assert response.data["shipment"] == shipment.id
    assert response.data["primary_worker"] == worker.id
    assert response.data["tractor"] == tractor.id


def test_post_movement(
    api_client: APIClient,
    organization: Organization,
    shipment: Shipment,
    worker: Worker,
    tractor: Tractor,
) -> None:
    """
    Test post Movement

    Args:
        api_client (APIClient): API Client
        organization (Organization): Organization instance
        shipment (Shipment): Order instance
        worker (Worker): Worker instance
        tractor (Tractor): Tractor instance

    Returns:
        None: This function does not return anything.

    """
    response = api_client.post(
        "/api/movements/",
        {
            "organization": f"{organization.id}",
            "shipment": f"{shipment.id}",
            "primary_worker": f"{worker.id}",
            "tractor": f"{tractor.id}",
        },
    )
    assert response.status_code == 201
    assert response.data is not None
    assert response.data["shipment"] == shipment.id
    assert response.data["primary_worker"] == worker.id
    assert response.data["tractor"] == tractor.id


def test_primary_worker_license_expiration_date() -> None:
    """
    Test ValidationError is thrown when the primary worker
    license_expiration_date is less than today's date.
    """
    worker = WorkerFactory()
    worker.profile.license_expiration_date = timezone.now() - timedelta(days=1)
    worker.profile.license_number = "123456789"
    worker.profile.license_state = "CA"
    worker.profile.save()

    dispatch_control = worker.organization.dispatch_control
    dispatch_control.regulatory_check = True
    dispatch_control.save()

    with pytest.raises(ValidationError) as excinfo:
        MovementFactory(
            organization=worker.organization,
            primary_worker=worker,
            business_unit=worker.business_unit,
        )

    assert excinfo.value.message_dict["primary_worker"] == [
        "Cannot assign a worker with an expired license. Please update the worker's profile and try again."
    ]


def test_primary_worker_physical_due_date() -> None:
    """
    Test ValidationError is thrown when the primary worker
    license_expiration_date is less than today's date.
    """
    worker = WorkerFactory()
    worker.profile.physical_due_date = timezone.now() - timedelta(days=1)
    worker.profile.save()

    dispatch_control = worker.organization.dispatch_control
    dispatch_control.regulatory_check = True
    dispatch_control.save()

    with pytest.raises(ValidationError) as excinfo:
        MovementFactory(organization=worker.organization, primary_worker=worker)

    assert excinfo.value.message_dict["primary_worker"] == [
        "Cannot assign a worker with an expired physical. Please update the worker's profile and try again."
    ]


def test_primary_worker_medical_cert_date() -> None:
    """
    Test ValidationError is thrown when the primary worker
    license_expiration_date is less than today's date.
    """
    worker = WorkerFactory()
    worker.profile.medical_cert_date = timezone.now() - timedelta(days=1)
    worker.profile.save()

    dispatch_control = worker.organization.dispatch_control
    dispatch_control.regulatory_check = True
    dispatch_control.save()

    with pytest.raises(ValidationError) as excinfo:
        MovementFactory(organization=worker.organization, primary_worker=worker)

    assert excinfo.value.message_dict["primary_worker"] == [
        "Cannot assign a worker with an expired medical certificate. Please update the worker's profile and try again."
    ]


def test_primary_worker_mvr_due_date() -> None:
    """
    Test ValidationError is thrown when the primary worker
    mvr_due_date is less than today's date.
    """

    worker = WorkerFactory()
    worker.profile.mvr_due_date = timezone.now() - timedelta(days=1)
    worker.profile.save()

    dispatch_control = worker.organization.dispatch_control
    dispatch_control.regulatory_check = True
    dispatch_control.save()

    with pytest.raises(ValidationError) as excinfo:
        MovementFactory(organization=worker.organization, primary_worker=worker)

    print("Excinfo", excinfo.value.message_dict)

    assert excinfo.value.message_dict["primary_worker"] == [
        "Cannot assign a worker with an expired MVR. Please update the worker's profile and try again."
    ]


def test_primary_worker_termination_date() -> None:
    """
    Test ValidationError is thrown when the primary worker termination_date
    is filled with any date.
    """
    worker = WorkerFactory()
    worker.profile.termination_date = timezone.now()
    worker.profile.save()

    dispatch_control = worker.organization.dispatch_control
    dispatch_control.regulatory_check = True
    dispatch_control.save()

    with pytest.raises(ValidationError) as excinfo:
        MovementFactory(organization=worker.organization, primary_worker=worker)

    assert excinfo.value.message_dict["primary_worker"] == [
        "Cannot assign a terminated worker. Please update the worker's profile and try again."
    ]


def test_primary_worker_tractor_fleet_validation(worker: Worker, organization) -> None:
    """
    Test ValidationError is thrown when the primary worker and the tractor
    are not a part of the same fleet.
    """

    organization.dispatch_control.tractor_worker_fleet_constraint = True
    organization.dispatch_control.save()

    with pytest.raises(ValidationError) as excinfo:
        MovementFactory(
            primary_worker=worker,
            tractor=TractorFactory(organization=worker.organization),
            organization=organization,
        )

    assert excinfo.value.message_dict["primary_worker"] == [
        "The primary worker and tractor must belong to the same fleet to add or update a record. Please ensure they are part of the same fleet and try again."
    ]
    assert excinfo.value.message_dict["tractor"] == [
        "The primary worker and tractor must belong to the same fleet to add or update a record. Please ensure they are part of the same fleet and try again."
    ]


def test_primary_worker_cannot_be_assigned_to_movement_without_hazmat() -> None:
    """
    Test ValidationError is thrown when the worker is being assigned
    to a movement with hazardous material and the worker does not have
    a hazmat endorsement
    """

    hazardous_material = HazardousMaterialFactory()
    commodity = CommodityFactory(hazardous_material=hazardous_material)
    shipment = ShipmentFactory(
        commodity=commodity, hazardous_material=hazardous_material
    )
    worker = WorkerFactory()

    with pytest.raises(ValidationError) as excinfo:
        MovementFactory(shipment=shipment, primary_worker=worker)

    assert excinfo.value.message_dict["primary_worker"] == [
        "Worker must be hazmat certified to haul this shipment. Please try again."
    ]


def test_primary_worker_cannot_be_assigned_to_movement_with_expired_hazmat() -> None:
    """
    Test ValidationError is thrown when the worker is being assigned
    to a movement with hazardous material and the worker does not have
    a hazmat endorsement
    """

    hazardous_material = HazardousMaterialFactory()
    commodity = CommodityFactory(hazardous_material=hazardous_material)
    shipment = ShipmentFactory(
        commodity=commodity, hazardous_material=hazardous_material
    )
    worker = WorkerFactory()
    worker.profile.endorsements = "H"
    worker.profile.hazmat_expiration_date = timezone.now().date() - timedelta(days=30)
    worker.profile.save()

    with pytest.raises(ValidationError) as excinfo:
        MovementFactory(shipment=shipment, primary_worker=worker, secondary_worker=None)

    assert excinfo.value.message_dict["primary_worker"] == [
        "Worker hazmat certification has expired. Please try again."
    ]


# --- Secondary Worker tests ---
def test_secondary_worker_license_expiration_date() -> None:
    """
    Test ValidationError is thrown when the secondary worker
    license_expiration_date is less than today's date.
    """

    worker = WorkerFactory()
    worker.profile.license_expiration_date = timezone.now() - timedelta(days=1)
    worker.profile.save()

    dispatch_control = worker.organization.dispatch_control
    dispatch_control.regulatory_check = True
    dispatch_control.save()

    with pytest.raises(ValidationError) as excinfo:
        MovementFactory(organization=worker.organization, secondary_worker=worker)

    assert excinfo.value.message_dict["secondary_worker"] == [
        "Cannot assign a worker with an expired license. Please update the worker's profile and try again."
    ]


def test_secondary_worker_physical_due_date() -> None:
    """
    Test ValidationError is thrown when the secondary worker
    license_expiration_date is less than today's date.
    """
    worker = WorkerFactory()
    worker.profile.physical_due_date = timezone.now() - timedelta(days=1)
    worker.profile.save()

    dispatch_control = worker.organization.dispatch_control
    dispatch_control.regulatory_check = True
    dispatch_control.save()

    with pytest.raises(ValidationError) as excinfo:
        MovementFactory(organization=worker.organization, secondary_worker=worker)

    assert excinfo.value.message_dict["secondary_worker"] == [
        "Cannot assign a worker with an expired physical. Please update the worker's profile and try again."
    ]


def test_secondary_worker_medical_cert_date() -> None:
    """
    Test ValidationError is thrown when the secondary worker
    license_expiration_date is less than today's date.
    """
    worker = WorkerFactory()
    worker.profile.medical_cert_date = timezone.now() - timedelta(days=1)
    worker.profile.save()

    dispatch_control = worker.organization.dispatch_control
    dispatch_control.regulatory_check = True
    dispatch_control.save()

    with pytest.raises(ValidationError) as excinfo:
        MovementFactory(organization=worker.organization, secondary_worker=worker)

    assert excinfo.value.message_dict["secondary_worker"] == [
        "Cannot assign a worker with an expired medical certificate. Please update the worker's profile and try again."
    ]


def test_secondary_worker_mvr_due_date() -> None:
    """
    Test ValidationError is thrown when the secondary worker
    mvr_due_date is less than today's date.
    """
    worker = WorkerFactory()
    worker.profile.mvr_due_date = timezone.now() - timedelta(days=1)
    worker.profile.save()

    dispatch_control = worker.organization.dispatch_control
    dispatch_control.regulatory_check = True
    dispatch_control.save()

    with pytest.raises(ValidationError) as excinfo:
        MovementFactory(organization=worker.organization, secondary_worker=worker)

    assert excinfo.value.message_dict["secondary_worker"] == [
        "Cannot assign a worker with an expired MVR. Please update the worker's profile and try again."
    ]


def test_secondary_worker_termination_date() -> None:
    """
    Test ValidationError is thrown when the secondary worker
    termination_date is filled with any date.
    """
    worker = WorkerFactory()
    worker.profile.termination_date = timezone.now()
    worker.profile.save()

    dispatch_control = worker.organization.dispatch_control
    dispatch_control.regulatory_check = True
    dispatch_control.save()

    with pytest.raises(ValidationError) as excinfo:
        MovementFactory(organization=worker.organization, secondary_worker=worker)

    assert excinfo.value.message_dict["secondary_worker"] == [
        "Cannot assign a terminated worker. Please update the worker's profile and try again."
    ]


def test_second_worker_cannot_be_assigned_to_movement_without_hazmat() -> None:
    """
    Test ValidationError is thrown when the worker is being assigned
    to a movement with hazardous material and the worker does not have
    a hazmat endorsement
    """

    hazardous_material = HazardousMaterialFactory()
    commodity = CommodityFactory(hazardous_material=hazardous_material)
    shipment = ShipmentFactory(
        commodity=commodity, hazardous_material=hazardous_material
    )
    primary_worker = WorkerFactory()
    primary_worker.profile.endorsements = "H"
    worker = WorkerFactory()

    with pytest.raises(ValidationError) as excinfo:
        MovementFactory(
            shipment=shipment, primary_worker=primary_worker, secondary_worker=worker
        )

    assert excinfo.value.message_dict["secondary_worker"] == [
        "Worker must be hazmat certified to haul this shipment. Please try again."
    ]


def test_second_worker_cannot_be_assigned_to_movement_with_expired_hazmat() -> None:
    """
    Test ValidationError is thrown when the worker is being assigned
    to a movement with hazardous material and the worker does not have
    a hazmat endorsement
    """

    hazardous_material = HazardousMaterialFactory()
    commodity = CommodityFactory(hazardous_material=hazardous_material)
    shipment = ShipmentFactory(
        commodity=commodity, hazardous_material=hazardous_material
    )

    primary_worker = WorkerFactory()
    primary_worker.profile.endorsements = "H"
    primary_worker.profile.hazmat_expiration_date = timezone.now().date()

    worker = WorkerFactory()
    worker.profile.endorsements = "H"
    worker.profile.hazmat_expiration_date = timezone.now().date() - timedelta(days=30)
    worker.profile.save()

    with pytest.raises(ValidationError) as excinfo:
        MovementFactory(
            shipment=shipment, primary_worker=primary_worker, secondary_worker=worker
        )

    assert excinfo.value.message_dict["secondary_worker"] == [
        "Worker hazmat certification has expired. Please try again."
    ]


def test_workers_cannot_be_the_same() -> None:
    """
    Test ValidationError is thrown when the primary worker and the
    secondary worker are the same.
    """
    worker = WorkerFactory()

    with pytest.raises(ValidationError) as excinfo:
        MovementFactory(
            organization=worker.organization,
            primary_worker=worker,
            secondary_worker=worker,
        )

    assert excinfo.value.message_dict["primary_worker"] == [
        "Primary worker cannot be the same as secondary worker. Please try again."
    ]


def test_movement_changed_to_in_progress_with_no_worker(shipment: Shipment) -> None:
    """
    Test ValidationError is thrown when the movement status is changed
    to in progress or completed and no worker or tractor is assigned.
    """
    movement = models.Movement.objects.filter(shipment=shipment).first()

    with pytest.raises(ValidationError) as excinfo:
        movement.status = "P"
        movement.primary_worker = None
        movement.secondary_worker = None
        movement.tractor = None
        movement.clean()

    assert excinfo.value.message_dict["primary_worker"] == [
        "Primary worker is required before movement status can be changed to `In Progress` or `Completed`. Please try again."
    ]
    assert excinfo.value.message_dict["tractor"] == [
        "Tractor is required before movement status can be changed to `In Progress` or `Completed`. Please try again."
    ]


def test_movement_cannot_change_status_in_in_progress_if_stops_are_new(
    shipment: Shipment,
) -> None:
    """
    Test ValidationError is thrown when the movement status is changed to in progress ,but
    none of the stops associated are in progress.
    """
    movement = models.Movement.objects.filter(shipment=shipment).first()

    with pytest.raises(ValidationError) as excinfo:
        movement.status = "P"
        movement.clean()

    assert excinfo.value.message_dict["status"] == [
        "Cannot change status to anything other than `NEW` if any of the stops are not in progress. Please try again."
    ]


def test_movement_cannot_change_status_to_completed_if_stops_are_in_progress(
    shipment: Shipment,
) -> None:
    """
    Test ValidationError is thrown when the movement status is
    changed to in new, but the stops status is in progress.
    """

    movement = models.Movement.objects.filter(shipment=shipment).first()

    with pytest.raises(ValidationError) as excinfo:
        movement.status = "C"
        movement.clean()

    assert excinfo.value.message_dict["status"] == [
        "Cannot change status to `COMPLETED` if any of the stops are in progress or new. Please try again."
    ]


def test_cannot_delete_movement_if_org_disallows(movement: models.Movement) -> None:
    """Test ValidationError is thrown if a movement is deleted and the organization,
    does not allow it.

    Args:
        movement(models.Movement): Movement instance

    Returns:
        None: This function does not return anything.
    """

    movement.organization.shipment_control.remove_shipments = False
    movement.organization.shipment_control.save()

    with pytest.raises(ValidationError) as excinfo:
        movement.delete()

    assert excinfo.value.message_dict["ref_num"] == [
        "Organization does not allow Movement removal. Please contact your administrator."
    ]
