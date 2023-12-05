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
from billing.tests.factories import AccessorialChargeFactory
from commodities.factories import CommodityFactory
from customer.factories import CustomerFactory
from dispatch import factories, models
from dispatch.factories import RateBillingTableFactory
from django.core.exceptions import ValidationError
from django.urls import reverse
from django.utils import timezone
from equipment.tests.factories import EquipmentTypeFactory
from organization.models import BusinessUnit, Organization
from rest_framework import status
from rest_framework.response import Response
from rest_framework.test import APIClient
from shipment.tests.factories import ShipmentTypeFactory

pytestmark = pytest.mark.django_db


def test_rate_create(organization: Organization, business_unit: BusinessUnit) -> None:
    customer = CustomerFactory()
    commodity = CommodityFactory()
    shipment_type = ShipmentTypeFactory()
    equipment_type = EquipmentTypeFactory()

    rate = models.Rate.objects.create(
        business_unit=business_unit,
        organization=organization,
        customer=customer,
        effective_date=timezone.now().date(),
        expiration_date=timezone.now().date(),
        commodity=commodity,
        shipment_type=shipment_type,
        equipment_type=equipment_type,
        comments="Test Rate",
    )

    assert rate is not None
    assert rate.organization == organization
    assert rate.rate_number == "R00001"
    assert rate.customer == customer
    assert rate.commodity == commodity
    assert rate.shipment_type == shipment_type
    assert rate.equipment_type == equipment_type
    assert rate.comments == "Test Rate"


def test_rate_update(rate: models.Rate) -> None:
    """
    Test the update method.
    """
    customer = CustomerFactory()
    commodity = CommodityFactory()
    shipment_type = ShipmentTypeFactory()
    equipment_type = EquipmentTypeFactory()

    rate.customer = customer
    rate.commodity = commodity
    rate.shipment_type = shipment_type
    rate.equipment_type = equipment_type
    rate.comments = "Test Rate Update"

    rate.save()

    assert rate is not None
    assert rate.customer == customer
    assert rate.commodity == commodity
    assert rate.shipment_type == shipment_type
    assert rate.equipment_type == equipment_type
    assert rate.comments == "Test Rate Update"


def test_rate_api_get(api_client: APIClient, organization: Organization) -> None:
    """
    Test the get method.
    """

    response = api_client.get(reverse("rates-list"))
    assert response.status_code == status.HTTP_200_OK


def test_rate_api_create(api_client: APIClient, organization: Organization) -> None:
    """
    Test the create method.
    """
    customer = CustomerFactory()
    commodity = CommodityFactory()
    shipment_type = ShipmentTypeFactory()
    equipment_type = EquipmentTypeFactory()

    data = {
        "organization": organization.id,
        "customer": customer.id,
        "effective_date": timezone.now().date(),
        "expiration_date": timezone.now().date(),
        "commodity": commodity.id,
        "shipment_type": shipment_type.id,
        "equipment_type": equipment_type.id,
        "comments": "Test Rate",
    }

    response = api_client.post(reverse("rates-list"), data=data)

    assert response.status_code == status.HTTP_201_CREATED
    assert models.Rate.objects.count() == 1
    assert models.Rate.objects.get().customer.id == data["customer"]


def test_rate_api_create_with_tables(
    api_client: APIClient, organization: Organization
) -> None:
    """
    Test the create method.
    """
    customer = CustomerFactory()
    commodity = CommodityFactory()
    shipment_type = ShipmentTypeFactory()
    equipment_type = EquipmentTypeFactory()
    accessorial_charge = AccessorialChargeFactory()

    response = api_client.post(
        "/api/rates/",
        data={
            "organization": organization.id,
            "customer": customer.id,
            "effective_date": timezone.now().date(),
            "expiration_date": timezone.now().date(),
            "commodity": commodity.id,
            "shipment_type": shipment_type.id,
            "equipment_type": equipment_type.id,
            "comments": "Test Rate 01",
            "rate_billing_tables": [
                {
                    "accessorial_charge": accessorial_charge.id,
                    "description": "Test Rate Billing Table",
                    "unit": 1,
                    "charge_amount": 100.00,
                    "sub_total": 100.00,
                }
            ],
        },
        format="json",
    )

    assert response.status_code == status.HTTP_201_CREATED
    assert models.Rate.objects.count() == 1
    assert models.Rate.objects.get().customer.id == customer.id
    assert (
        response.data["rate_billing_tables"][0]["accessorial_charge"]
        == accessorial_charge.id
    )
    assert (
        response.data["rate_billing_tables"][0]["description"]
        == "Test Rate Billing Table"
    )
    assert response.data["rate_billing_tables"][0]["unit"] == 1
    assert (
        decimal.Decimal(response.data["rate_billing_tables"][0]["charge_amount"])
        == 100.00
    )
    assert (
        decimal.Decimal(response.data["rate_billing_tables"][0]["sub_total"]) == 100.00
    )


def test_rate_api_update(api_client: APIClient, rate: models.Rate) -> None:
    """
    Test the update method.
    """
    customer = CustomerFactory()
    commodity = CommodityFactory()
    shipment_type = ShipmentTypeFactory()
    equipment_type = EquipmentTypeFactory()
    accessorial_charge = AccessorialChargeFactory()

    data = {
        "customer": customer.id,
        "commodity": commodity.id,
        "shipment_type": shipment_type.id,
        "equipment_type": equipment_type.id,
        "comments": "Test Rate Update",
        "rate_billing_tables": [
            {
                "accessorial_charge": accessorial_charge.id,
                "description": "Test Rate Billing Table",
                "unit": 1,
                "charge_amount": 100.00,
                "sub_total": 100.00,
            },
            {
                "accessorial_charge": accessorial_charge.id,
                "description": "Test Rate Billing Table 2",
                "unit": 100,
                "charge_amount": 100.00,
                "sub_total": 100.00,
            },
        ],
    }

    response = api_client.put(
        f"/api/rates/{rate.id}/",
        data=data,
        format="json",
    )
    
    assert response.status_code == status.HTTP_200_OK
    assert (
        response.data["rate_billing_tables"][0]["description"]
        == "Test Rate Billing Table 2"
    )
    assert response.data["rate_billing_tables"][0]["unit"] == 100
    assert (
        response.data["rate_billing_tables"][1]["description"]
        == "Test Rate Billing Table"
    )
    assert response.data["rate_billing_tables"][1]["unit"] == 1


def test_rate_api_delete(api_client: APIClient, rate_api: Response) -> None:
    """
    Test the delete method.
    """

    response = api_client.delete(
        reverse("rates-detail", kwargs={"pk": rate_api.data["id"]})
    )
    assert response.status_code == status.HTTP_204_NO_CONTENT
    assert response.data is None
    assert models.Rate.objects.count() == 0


def test_expiration_cannot_be_greater_than_effective_date(rate: models.Rate) -> None:
    """
    Test that the expiration date cannot be greater than the effective date.
    """
    rate.expiration_date = rate.effective_date - datetime.timedelta(days=1)

    with pytest.raises(ValidationError) as excinfo:
        rate.full_clean()

    assert excinfo.value.message_dict["expiration_date"] == [
        "Expiration Date must be after Effective Date. Please correct and try again."
    ]


def test_set_rate_number_before_create_hook(rate: models.Rate) -> None:
    """
    Test the set_rate_number_before_create_hook method.
    """
    assert rate.rate_number is not None
    assert rate.rate_number == "R00001"


def test_set_rate_number_increment_hook(rate: models.Rate) -> None:
    """
    Test the set_rate_number_increment_hook method.
    """
    rate2 = factories.RateFactory()

    assert rate.rate_number is not None
    assert rate.rate_number == "R00001"
    assert rate2.rate_number is not None
    assert rate2.rate_number == "R00002"