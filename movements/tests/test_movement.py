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
from datetime import timedelta

import pytest
from django.core.exceptions import ValidationError
from django.utils import timezone

from commodities.factories import CommodityFactory, HazardousMaterialFactory
from movements import models
from movements.tests.factories import MovementFactory
from order.tests.factories import OrderFactory
from stops.tests.factories import StopFactory
from worker.factories import WorkerFactory

pytestmark = pytest.mark.django_db


class TestMovement:
    """
    Class to test Movement
    """

    def test_list(self, movement):
        """
        Test for Movement List
        """
        assert movement is not None

    def test_create(self, worker, equipment, organization, order):
        """
        Test Movement Create
        """

        movement = models.Movement.objects.create(
            organization=organization,
            order=order,
            equipment=equipment,
            primary_worker=worker,
        )

        assert movement is not None
        assert movement.order == order
        assert movement.equipment == equipment
        assert movement.primary_worker == worker

    def test_update(self, movement, equipment):
        """
        Test Movement Update
        """

        add_movement = models.Movement.objects.get(id=movement.id)

        add_movement.equipment = equipment
        add_movement.save()

        assert add_movement is not None
        assert add_movement.equipment == equipment


class TestMovementAPI:
    """
    Class to test Movement API
    """

    @pytest.fixture
    def movement_api(self, api_client, organization, order, equipment, worker):
        """
        Movement Factory
        """
        return api_client.post(
            "/api/movements/",
            {
                "organization": f"{organization.id}",
                "order": f"{order.id}",
                "primary_worker": f"{worker.id}",
                "equipment": f"{equipment.id}",
            },
        )

    def test_get(self, api_client):
        """
        Test get Movement
        """
        response = api_client.get("/api/movements/")
        assert response.status_code == 200

    def test_get_by_id(self, api_client, movement_api, order, worker, equipment):
        """
        Test get Movement by ID
        """

        response = api_client.get(f"/api/movements/{movement_api.data['id']}/")

        assert response.status_code == 200
        assert response.data is not None
        assert response.data["order"] == order.id
        assert response.data["primary_worker"] == worker.id
        assert response.data["equipment"] == equipment.id


class TestMovementValidation:
    """
    Test for Movement model validation
    """

    def test_primary_worker_license_expiration_date(self):
        """
        Test ValidationError is thrown when the primary worker
        license_expiration_date is less than today's date.
        """
        worker = WorkerFactory()
        worker.profile.license_expiration_date = timezone.now() - timedelta(days=1)
        worker.profile.save()

        dispatch_control = worker.organization.dispatch_control
        dispatch_control.regulatory_check = True
        dispatch_control.save()

        with pytest.raises(ValidationError) as excinfo:
            MovementFactory(organization=worker.organization, primary_worker=worker)

        assert excinfo.value.message_dict["primary_worker"] == [
            "Cannot assign a worker with an expired license. Please update the worker's profile and try again."
        ]

    def test_primary_worker_physical_due_date(self):
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

    def test_primary_worker_medical_cert_date(self):
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

    def test_primary_worker_mvr_due_date(self):
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

        assert excinfo.value.message_dict["primary_worker"] == [
            "Cannot assign a worker with an expired MVR. Please update the worker's profile and try again."
        ]

    def test_primary_worker_termination_date(self):
        """
        Test ValidationError is thrown when the primary worker
        termination_date is filled with any date.
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
    def test_primary_worker_cannot_be_assigned_to_movement_without_hazmat(self):
        """
        Test ValidationError is thrown when the worker is being assigned
        to a movement with hazardous material and the worker does not have
        a hazmat endorsement
        """

        hazmat = HazardousMaterialFactory()
        commodity = CommodityFactory(hazmat=hazmat)
        order = OrderFactory(commodity=commodity, hazmat=hazmat)
        worker = WorkerFactory()

        with pytest.raises(ValidationError) as excinfo:
            MovementFactory(order=order, primary_worker=worker)

        assert excinfo.value.message_dict["primary_worker"] == [
            "Worker must be hazmat certified to haul this order. Please try again."
        ]


    def test_primary_worker_cannot_be_assigned_to_movement_with_expired_hazmat(self):
        """
        Test ValidationError is thrown when the worker is being assigned
        to a movement with hazardous material and the worker does not have
        a hazmat endorsement
        """

        hazmat = HazardousMaterialFactory()
        commodity = CommodityFactory(hazmat=hazmat)
        order = OrderFactory(commodity=commodity, hazmat=hazmat)
        worker = WorkerFactory()
        worker.profile.endorsements = "H"
        worker.profile.hazmat_expiration_date = timezone.now().date() - timedelta(days=1)

        with pytest.raises(ValidationError) as excinfo:
            MovementFactory(order=order, primary_worker=worker)

        assert excinfo.value.message_dict["primary_worker"] == [
            "Worker hazmat certification has expired. Please try again."
        ]

    # --- Secondary Worker tests ---
    def test_secondary_worker_license_expiration_date(self):
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

    def test_secondary_worker_physical_due_date(self):
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

    def test_secondary_worker_medical_cert_date(self):
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

    def test_secondary_worker_mvr_due_date(self):
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

    def test_secondary_worker_termination_date(self):
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

    def test_second_worker_cannot_be_assigned_to_movement_without_hazmat(self):
        """
        Test ValidationError is thrown when the worker is being assigned
        to a movement with hazardous material and the worker does not have
        a hazmat endorsement
        """

        hazmat = HazardousMaterialFactory()
        commodity = CommodityFactory(hazmat=hazmat)
        order = OrderFactory(commodity=commodity, hazmat=hazmat)
        primary_worker = WorkerFactory()
        primary_worker.profile.endorsements = "H"
        worker = WorkerFactory()

        with pytest.raises(ValidationError) as excinfo:
            MovementFactory(order=order, primary_worker=primary_worker, secondary_worker=worker)

        assert excinfo.value.message_dict["secondary_worker"] == [
            "Worker must be hazmat certified to haul this order. Please try again."
        ]


    def test_second_worker_cannot_be_assigned_to_movement_with_expired_hazmat(self):
        """
        Test ValidationError is thrown when the worker is being assigned
        to a movement with hazardous material and the worker does not have
        a hazmat endorsement
        """

        hazmat = HazardousMaterialFactory()
        commodity = CommodityFactory(hazmat=hazmat)
        order = OrderFactory(commodity=commodity, hazmat=hazmat)

        primary_worker = WorkerFactory()
        primary_worker.profile.endorsements = "H"
        primary_worker.profile.hazmat_expiration_date = timezone.now().date()

        worker = WorkerFactory()
        worker.profile.endorsements = "H"
        worker.profile.hazmat_expiration_date = timezone.now().date() - timedelta(days=1)

        with pytest.raises(ValidationError) as excinfo:
            MovementFactory(order=order, primary_worker=primary_worker, secondary_worker=worker)

        assert excinfo.value.message_dict["secondary_worker"] == [
            "Worker hazmat certification has expired. Please try again."
        ]

    def test_workers_cannot_be_the_same(self):
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

    def test_movement_changed_to_in_progress_with_no_worker(self):
        """
        Test ValidationError is thrown when the movement status is changed
        to in progress or completed and no worker or equipment is assigned.
        """

        with pytest.raises(ValidationError) as excinfo:
            MovementFactory(
                status="P",
                primary_worker=None,
                secondary_worker=None,
                equipment=None,
            )

        assert excinfo.value.message_dict["primary_worker"] == [
            "Primary worker is required before movement status can be changed to `In Progress` or `Completed`. Please try again."
        ]
        assert excinfo.value.message_dict["equipment"] == [
            "Equipment is required before movement status can be changed to `In Progress` or `Completed`. Please try again."
        ]

    def test_movement_cannot_change_status_in_in_progress_if_stops_are_new(self):
        """
        Test ValidationError is thrown when the movement status is
        changed to in progress ,but none of the stops associated are
        in progress.
        """
        movement = MovementFactory()

        stop_1 = StopFactory(movement=movement)
        stop_2 = StopFactory(movement=movement)

        with pytest.raises(ValidationError) as excinfo:
            movement.status = "P"
            movement.save()

        assert excinfo.value.message_dict["status"] == [
            "Cannot change status to anything other than `NEW` if any of the stops are not in progress. Please try again."
        ]

    def test_movement_cannot_change_status_to_new_if_stops_are_in_progress(self):
        """
        Test ValidationError is thrown when the movement status is
        changed to in new, but the stops status is in progress.
        """
        movement = MovementFactory()

        stop_1 = StopFactory(movement=movement, status="P")
        stop_2 = StopFactory(movement=movement, status="P")

        with pytest.raises(ValidationError) as excinfo:
            movement.status = "N"
            movement.save()

        assert excinfo.value.message_dict["status"] == [
            "Cannot change status to `NEW` if any of the stops are in progress or completed. Please try again."
        ]

    def test_movement_cannot_change_status_to_completed_if_stops_are_in_progress(self):
        """
        Test ValidationError is thrown when the movement status is
        changed to in new, but the stops status is in progress.
        """
        movement = MovementFactory()

        stop_1 = StopFactory(movement=movement, status="P")
        stop_2 = StopFactory(movement=movement, status="P")

        with pytest.raises(ValidationError) as excinfo:
            movement.status = "C"
            movement.save()

        assert excinfo.value.message_dict["status"] == [
            "Cannot change status to `COMPLETED` if any of the stops are in progress or new. Please try again."
        ]
