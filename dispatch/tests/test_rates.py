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
from django.utils import timezone
from pydantic import BaseModel

from dispatch import factories

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

    assert str(rate) == "R00001"


def test_list(rate) -> None:
    """
    Test the list method.
    """
    assert rate is not None


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
