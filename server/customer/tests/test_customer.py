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
from collections.abc import Generator
from typing import Any

import pytest
from django.core.exceptions import ValidationError
from rest_framework.test import APIClient

from billing.tests.factories import DocumentClassificationFactory
from customer import factories, models
from location.factories import LocationFactory
from organization.models import BusinessUnit, Organization

pytestmark = pytest.mark.django_db

# TODO(Wolfred): I didn't realize I had literally no tests for this.... Let's do that add some point.


@pytest.fixture
def customer_contact() -> Generator[Any, Any, None]:
    """
    Customer contact fixture
    """
    yield factories.CustomerContactFactory()


def test_customer_creation(customer) -> None:
    """
    Test customer creation
    """
    assert customer is not None


def test_customer_update(customer) -> None:
    """
    Test customer update
    """
    customer.name = "New name"
    customer.save()
    assert customer.name == "New name"


def test_generate_customer_code(
    organization: Organization, business_unit: BusinessUnit
) -> None:
    """Test when inserting a customer, that a code is generated for them.

    Args:
        organization(Organization): Organization Object.
        business_unit(BusinessUnit): Business Unit Object.

    Returns:
        None: This function does not return anything.
    """
    customer = models.Customer.objects.create(
        organization=organization,
        business_unit=business_unit,
        name="Intel Corporation",
    )

    assert customer.code == "INTEL0001"


def test_create_customer_with_details(api_client: APIClient) -> None:
    document_classification = DocumentClassificationFactory()
    location = LocationFactory()

    response = api_client.post(
        "/api/customers/",
        {
            "name": "Intel Corporation",
            "code": "INTEL0001",
            "address_line_1": "123 Fake Street",
            "address_line_2": "Unit 1",
            "city": "Fake City",
            "state": "NC",
            "zip_code": "12345",
            "customer_contacts": [
                {
                    "name": "Test Contact",
                    "email": "test@monta.io",
                    "title": "Test Title",
                    "phone_number": "123-456-7890",
                    "is_payable_contact": True,
                }
            ],
            "email_profile": {
                "subject": "Test Subject",
                "comment": "Test Comment",
                "from_address": "test@monta.io",
                "blind_copy": "test2@monta.io, test2@monta.io",
            },
            "rule_profile": {
                "name": "Test Rule Profile",
                "document_class": [document_classification.id],
            },
            "delivery_slots": [
                {
                    "start_time": "20:37:33",
                    "end_time": "21:37:33",
                    "day_of_week": 0,
                    "location": location.id,
                }
            ],
        },
        format="json",
    )

    assert response.status_code == 201
    assert response.data is not None
    assert response.data["name"] == "Intel Corporation"
    assert response.data["code"] == "INTEL0001"
    assert response.data["address_line_1"] == "123 Fake Street"
    assert response.data["address_line_2"] == "Unit 1"
    assert response.data["city"] == "Fake City"
    assert response.data["state"] == "NC"
    assert response.data["zip_code"] == "12345"
    assert response.data["email_profile"]["subject"] == "Test Subject"
    assert response.data["email_profile"]["comment"] == "Test Comment"
    assert response.data["email_profile"]["from_address"] == "test@monta.io"
    assert (
        response.data["email_profile"]["blind_copy"] == "test2@monta.io, test2@monta.io"
    )
    assert response.data["rule_profile"]["name"] == "Test Rule Profile"
    assert response.data["rule_profile"]["document_class"] == [
        document_classification.id
    ]
    assert response.data["delivery_slots"][0]["start_time"] == "20:37:33"
    assert response.data["delivery_slots"][0]["end_time"] == "21:37:33"
    assert response.data["delivery_slots"][0]["day_of_week"] == 0
    assert response.data["delivery_slots"][0]["location"] == location.id


def test_edit_customer_with_details(
    api_client: APIClient, customer: models.Customer
) -> None:
    document_classification = DocumentClassificationFactory()
    location = LocationFactory()

    response = api_client.put(
        f"/api/customers/{customer.id}/",
        {
            "name": "Intel Corporation",
            "address_line_1": "123 Fake Street",
            "address_line_2": "Unit 1",
            "city": "Fake City",
            "state": "NC",
            "zip_code": "12345",
            "customer_contacts": [
                {
                    "name": "Test Contact",
                    "email": "test@monta.io",
                    "title": "Test Title",
                    "phone_number": "123-456-7890",
                    "is_payable_contact": True,
                }
            ],
            "email_profile": {
                "subject": "Vimeo Customer Support",
                "comment": "Do Not Email",
                "from_address": "test@vimeo.com",
                "blind_copy": "test2@vimeo.com, test2@vimeo.com",
            },
            "rule_profile": {
                "name": "Vimeo Rule Profile",
                "document_class": [document_classification.id],
            },
            "delivery_slots": [
                {
                    "start_time": "20:37:33",
                    "end_time": "21:37:33",
                    "day_of_week": 0,
                    "location": location.id,
                }
            ],
        },
        format="json",
    )

    assert response.status_code == 200
    assert response.data is not None
    assert response.data["name"] == "Intel Corporation"
    assert response.data["address_line_1"] == "123 Fake Street"
    assert response.data["address_line_2"] == "Unit 1"
    assert response.data["city"] == "Fake City"
    assert response.data["state"] == "NC"
    assert response.data["zip_code"] == "12345"
    assert response.data["email_profile"]["subject"] == "Vimeo Customer Support"
    assert response.data["email_profile"]["comment"] == "Do Not Email"
    assert response.data["email_profile"]["from_address"] == "test@vimeo.com"
    assert (
        response.data["email_profile"]["blind_copy"]
        == "test2@vimeo.com, test2@vimeo.com"
    )
    assert response.data["rule_profile"]["name"] == "Vimeo Rule Profile"
    assert response.data["rule_profile"]["document_class"] == [
        document_classification.id
    ]
    assert response.data["delivery_slots"][0]["start_time"] == "20:37:33"
    assert response.data["delivery_slots"][0]["end_time"] == "21:37:33"
    assert response.data["delivery_slots"][0]["day_of_week"] == 0
    assert response.data["delivery_slots"][0]["location"] == location.id


def test_validate_blind_copy_emails(customer: models.Customer) -> None:
    """Test ValidationError is thrown when email in blind copy is not valid.

    Args:
        customer(models.Customer): Customer object.

    Returns:
        None: This function does not return anything.
    """

    with pytest.raises(ValidationError) as excinfo:
        factories.CustomerEmailProfileFactory(customer=customer, blind_copy="Test2")

    assert excinfo.value.message_dict["blind_copy"] == [
        "Test2 is not a valid email address. Please try again."
    ]


def test_customer_contact_creation(customer_contact: models.CustomerContact) -> None:
    """
    Test customer contact creation
    """
    assert customer_contact is not None


def test_customer_contact_update(customer_contact: models.CustomerContact) -> None:
    """
    Test customer contact update
    """
    customer_contact.name = "New name"
    customer_contact.save()
    assert customer_contact.name == "New name"


def test_customer_contact_payable_has_no_email(
    customer_contact: models.CustomerContact,
) -> None:
    """
    Test customer contact payable has no email
    """

    with pytest.raises(
        ValidationError, match="Payable contact must have an email address"
    ):
        customer_contact.email = ""
        customer_contact.full_clean()
