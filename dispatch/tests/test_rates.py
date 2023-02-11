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
import datetime
import uuid

import pytest
from django.urls import reverse
from django.utils import timezone
from pydantic import BaseModel
from rest_framework import status

from billing.tests.factories import AccessorialChargeFactory
from commodities.factories import CommodityFactory
from customer.factories import CustomerFactory
from dispatch import factories, models, serializers
from dispatch.factories import RateBillingTableFactory
from equipment.tests.factories import EquipmentTypeFactory
from order.tests.factories import OrderTypeFactory

pytestmark = pytest.mark.django_db


class RateBase(BaseModel):
    """
    Rate Base Schema
    """

    organization_id: uuid.UUID
    rate_number: str
    customer: uuid.UUID | None
    effective_date: datetime.date
    expiration_date: datetime.date
    commodity_id: uuid.UUID | None
    order_type_id: uuid.UUID | None
    equipment_type_id: uuid.UUID | None


class RateCreate(RateBase):
    """
    Rate Create Schema
    """

    pass


class RateUpdate(RateBase):
    """
    Rate Update Schema
    """

    id: uuid.UUID


def test_create_schema() -> None:
    """
    Test Rate Creation Schema    Returns:
    """

    rate_create = RateCreate(
        organization_id=uuid.uuid4(),
        rate_number="R00001",
        customer=uuid.uuid4(),
        effective_date=timezone.now().date(),
        expiration_date=timezone.now().date(),
        commodity_id=uuid.uuid4(),
        order_type_id=uuid.uuid4(),
        equipment_type_id=uuid.uuid4(),
    )

    rate = rate_create.dict()
    assert rate is not None
    assert rate["organization_id"] is not None
    assert rate["rate_number"] == "R00001"
    assert rate["customer"] is not None
    assert rate["effective_date"] is not None
    assert rate["expiration_date"] is not None
    assert rate["commodity_id"] is not None
    assert rate["order_type_id"] is not None
    assert rate["equipment_type_id"] is not None


def test_update_schema() -> None:
    """
    Test Rate Update Schema
    """
    rate_update = RateUpdate(
        id=uuid.uuid4(),
        organization_id=uuid.uuid4(),
        rate_number="R00001",
        customer=uuid.uuid4(),
        effective_date=timezone.now().date(),
        expiration_date=timezone.now().date(),
        commodity_id=uuid.uuid4(),
        order_type_id=uuid.uuid4(),
        equipment_type_id=uuid.uuid4(),
    )

    rate = rate_update.dict()
    assert rate is not None
    assert rate["id"] is not None
    assert rate["organization_id"] is not None
    assert rate["rate_number"] == "R00001"
    assert rate["customer"] is not None
    assert rate["effective_date"] is not None
    assert rate["expiration_date"] is not None
    assert rate["commodity_id"] is not None
    assert rate["order_type_id"] is not None
    assert rate["equipment_type_id"] is not None


def test_delete_schema() -> None:
    """
    Test Rate Delete Schema
    """
    rates = [
        RateBase(
            organization_id=uuid.uuid4(),
            rate_number="R00001",
            customer=uuid.uuid4(),
            effective_date=timezone.now().date(),
            expiration_date=timezone.now().date(),
            commodity_id=uuid.uuid4(),
            order_type_id=uuid.uuid4(),
            equipment_type_id=uuid.uuid4(),
        ),
        RateBase(
            organization_id=uuid.uuid4(),
            rate_number="R00002",
            customer=uuid.uuid4(),
            effective_date=timezone.now().date(),
            expiration_date=timezone.now().date(),
            commodity_id=uuid.uuid4(),
            order_type_id=uuid.uuid4(),
            equipment_type_id=uuid.uuid4(),
        ),
    ]

    rate_store = rates.copy()

    rate_store.pop(0)

    assert len(rates) == 2
    assert len(rate_store) == 1
    assert rate_store[0].rate_number == "R00002"
    assert rates[0].rate_number == "R00001"


def test_rate_str_representation(rate) -> None:
    """
    Test the rate string representation.
    """

    assert str(rate) == rate.rate_number


def test_list(rate) -> None:
    """
    Test the list method.
    """
    assert rate is not None


def test_rate_create(organization) -> None:
    customer = CustomerFactory()
    commodity = CommodityFactory()
    order_type = OrderTypeFactory()
    equipment_type = EquipmentTypeFactory()

    rate = models.Rate.objects.create(
        organization=organization,
        customer=customer,
        effective_date=timezone.now().date(),
        expiration_date=timezone.now().date(),
        commodity=commodity,
        order_type=order_type,
        equipment_type=equipment_type,
        comments="Test Rate",
    )

    assert rate is not None
    assert rate.organization == organization
    assert rate.rate_number == "R00001"
    assert rate.customer == customer
    assert rate.commodity == commodity
    assert rate.order_type == order_type
    assert rate.equipment_type == equipment_type
    assert rate.comments == "Test Rate"


def test_rate_update(rate) -> None:
    """
    Test the update method.
    """
    customer = CustomerFactory()
    commodity = CommodityFactory()
    order_type = OrderTypeFactory()
    equipment_type = EquipmentTypeFactory()

    rate.customer = customer
    rate.commodity = commodity
    rate.order_type = order_type
    rate.equipment_type = equipment_type
    rate.comments = "Test Rate Update"

    rate.save()

    assert rate is not None
    assert rate.customer == customer
    assert rate.commodity == commodity
    assert rate.order_type == order_type
    assert rate.equipment_type == equipment_type
    assert rate.comments == "Test Rate Update"


def test_rate_api_get(api_client, organization) -> None:
    """
    Test the get method.
    """

    response = api_client.get(reverse("rates-list"))
    assert response.status_code == status.HTTP_200_OK

def test_rate_api_create(api_client, organization) -> None:
    """
    Test the create method.
    """
    customer = CustomerFactory()
    commodity = CommodityFactory()
    order_type = OrderTypeFactory()
    equipment_type = EquipmentTypeFactory()

    data = {
        "organization": organization.id,
        "customer": customer.id,
        "effective_date": timezone.now().date(),
        "expiration_date": timezone.now().date(),
        "commodity": commodity.id,
        "order_type": order_type.id,
        "equipment_type": equipment_type.id,
        "comments": "Test Rate",
    }

    response = api_client.post(reverse("rates-list"), data=data)
    assert response.status_code == status.HTTP_201_CREATED
    assert models.Rate.objects.count() == 1
    assert models.Rate.objects.get().customer.id == data["customer"]

def test_rate_api_update(api_client, rate_api) -> None:
    """
    Test the update method.
    """
    customer = CustomerFactory()
    commodity = CommodityFactory()
    order_type = OrderTypeFactory()
    equipment_type = EquipmentTypeFactory()

    data = {
        "customer": customer.id,
        "commodity": commodity.id,
        "order_type": order_type.id,
        "equipment_type": equipment_type.id,
        "comments": "Test Rate Update",
    }

    response = api_client.patch(
        reverse("rates-detail", kwargs={"pk": rate_api.data["id"]}), data=data
    )
    assert response.status_code == status.HTTP_200_OK
    assert models.Rate.objects.count() == 1
    assert models.Rate.objects.get().customer.id == data["customer"]
    assert models.Rate.objects.get().commodity.id == data["commodity"]
    assert models.Rate.objects.get().order_type.id == data["order_type"]
    assert models.Rate.objects.get().equipment_type.id == data["equipment_type"]
    assert models.Rate.objects.get().comments == data["comments"]

def test_rate_api_delete(api_client, rate_api) -> None:
    """
    Test the delete method.
    """

    response = api_client.delete(reverse("rates-detail", kwargs={"pk": rate_api.data["id"]}))
    assert response.status_code == status.HTTP_204_NO_CONTENT
    assert response.data is None
    assert models.Rate.objects.count() == 0


def test_set_rate_number_before_create_hook(rate) -> None:
    """
    Test the set_rate_number_before_create_hook method.
    """
    assert rate.rate_number is not None
    assert rate.rate_number == "R00001"


def test_set_rate_number_increment_hook(rate) -> None:
    """
    Test the set_rate_number_increment_hook method.
    """
    rate2 = factories.RateFactory()

    assert rate.rate_number is not None
    assert rate.rate_number == "R00001"
    assert rate2.rate_number is not None
    assert rate2.rate_number == "R00002"


def test_rate_table_str_representation(rate_table) -> None:
    """
    Test the rate table string representation.
    """

    assert str(rate_table) == rate_table.description


#
# def test_rate_table_get_absolute_url(rate_table) -> None:
#     """
#     Test the rate table get_absolute_url method.
#     """
#
#     assert rate_table.get_absolute_url() == f"/api/rate_tables/{rate_table.id}/"


def test_rate_billing_table_before_save_hook() -> None:
    """
    Test the Rate billing Table BEFORE_SAVE hook properly set values
    """

    accessorial_charge = AccessorialChargeFactory()
    rate_billing_table = RateBillingTableFactory(
        charge_code=accessorial_charge,
        charge_amount=0,
    )

    assert rate_billing_table.charge_amount == accessorial_charge.charge_amount
    assert (
        rate_billing_table.sub_total
        == accessorial_charge.charge_amount * rate_billing_table.units
    )
