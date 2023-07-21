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

import pytest
from django.utils import timezone

from accounts.tests.factories import UserFactory
from billing.models import BillingQueue
from edi import helpers, exceptions, models
from edi.tests import factories
from order.tests.factories import OrderFactory
from organization.models import BusinessUnit, Organization

pytestmark = pytest.mark.django_db


def test_generate_edi_envelope_headers(
    organization: Organization, business_unit: BusinessUnit
) -> None:
    """Test generation of EDI envelope headers and trailers.

    Args:
        organization (Organization): The organization instance.
        business_unit (): The business unit instance.

    Returns:
        None: This function does not return anything.
    """
    now = timezone.now()
    date = now.strftime("%y%m%d")
    time = now.strftime("%H%M")

    _, _, edi_billing_profile = factories.EDISegmentFactory(
        business_unit=business_unit,
        organization=organization,
    )

    headers = helpers.generate_edi_envelope_headers(
        edi_profile=edi_billing_profile,
        date=date,
        time=time,
    )

    # Split the headers into lines
    lines = headers.split("\n")

    # Assert that the headers start with ISA and GS
    assert lines[0].startswith("ISA*")
    assert lines[1].startswith("GS*")


def test_generate_edi_trailers() -> None:
    """Test generation of EDI envelope trailers.

    Returns:
        None: This function does return anything.
    """

    # This is going to change as it will increment based on the number of transactions
    trailers = helpers.generate_edi_trailers()

    lines = trailers.split("\n")

    assert lines[0].startswith("SE*1")
    assert lines[1].startswith("GE*1")
    assert lines[2].startswith("IEA*1*")


def test_get_nested_attr(
    organization: Organization, business_unit: BusinessUnit
) -> None:
    """Test getting nested attribute.

    Returns:
        None: This function does not return anything.
    """
    order_1 = OrderFactory()
    user = UserFactory()

    order_movements = order_1.movements.all()
    order_movements.update(status="C")

    order_1.status = "C"
    order_1.save()

    billing_item = BillingQueue.objects.create(
        organization=organization,
        business_unit=business_unit,
        order=order_1,
        user=user,
        customer=order_1.customer,
    )

    nest_attr = helpers.get_nested_attr(
        obj=billing_item,
        attr="order.customer.name",
    )

    assert nest_attr == billing_item.order.customer.name
    assert nest_attr == order_1.customer.name


def test_get_nested_attr_exception(
    organization: Organization, business_unit: BusinessUnit
) -> None:
    """Test getting nested attribute exception.

    Returns:
        None: This function does not return anything.
    """
    order_1 = OrderFactory()
    user = UserFactory()

    order_movements = order_1.movements.all()
    order_movements.update(status="C")

    order_1.status = "C"
    order_1.save()

    billing_item = BillingQueue.objects.create(
        organization=organization,
        business_unit=business_unit,
        order=order_1,
        user=user,
        customer=order_1.customer,
    )

    with pytest.raises(exceptions.FieldDoesNotExist) as excinfo:
        helpers.get_nested_attr(
            obj=billing_item,
            attr="order.customer.name1",
        )

    assert (
        excinfo.value.args[0]
        == "Field `order.customer.name1` does not exist on BillingQueue model."
    )


def test_generate_edi_content(
    organization: Organization, business_unit: BusinessUnit
) -> None:
    order_1 = OrderFactory()
    user = UserFactory()

    order_movements = order_1.movements.all()
    order_movements.update(status="C")

    order_1.status = "C"
    order_1.save()

    billing_item = BillingQueue.objects.create(
        organization=organization,
        business_unit=business_unit,
        order=order_1,
        user=user,
        customer=order_1.customer,
    )

    _, fields, edi_billing_profile = factories.EDISegmentFactory(
        business_unit=business_unit,
        organization=organization,
    )

    content = helpers.generate_edi_content(
        billing_item=billing_item, edi_billing_profile=edi_billing_profile
    )

    # Split the content into lines
    lines = content.split("\n")

    # Assert that the content contains the fields.
    assert lines[0].startswith("B3*")
    assert lines[1].startswith("C3*")
    assert lines[2].startswith("N9*")
    assert lines[3].startswith("N1*")
    assert lines[4].startswith("N3*")
    assert lines[5].startswith("N4*")
    assert lines[6].startswith("N7*")
    assert lines[7].startswith("LX*")
    assert lines[8].startswith("L5*")
    assert lines[9].startswith("L0*")
    assert lines[10].startswith("L1*")
    assert lines[11].startswith("L3*")

    # Assert that the content contains the values.
    assert lines[0].endswith("*1")
    assert lines[1].endswith("*USD")
    assert lines[2].endswith("*1")
    assert lines[3].endswith("*1")
    assert lines[4].endswith("*1")
    assert lines[5].endswith("*1")
    assert lines[6].endswith("*1")
    assert lines[7].endswith("*1")
    assert lines[8].endswith("*T")
    assert lines[9].endswith("*TKR")
    assert lines[10].endswith("*MR")
    assert lines[11].endswith("*E")

def test_generate_edi_content_value_returns_empty_string(
        organization: Organization, business_unit: BusinessUnit
) -> None:
    """Test generate_edi_content value returns an empty string if value is ``None``

    Args:
        organization (Organization): The organization instance.
        business_unit (BusinessUnit): The business unit instance.

    Returns:
        None: This function does not return anything.
    """
    order_1 = OrderFactory()
    user = UserFactory()

    order_movements = order_1.movements.all()
    order_movements.update(status="C")

    order_1.status = "C"
    order_1.save()

    billing_item = BillingQueue.objects.create(
        organization=organization,
        business_unit=business_unit,
        order=order_1,
        user=user,
        customer=order_1.customer,
    )

    _, fields, edi_billing_profile = factories.EDISegmentFactory(
        business_unit=business_unit,
        organization=organization,
    )

    fields.update(model_field="order.commodity")

    content = helpers.generate_edi_content(
        billing_item=billing_item, edi_billing_profile=edi_billing_profile
    )

    # Split the content into lines
    lines = content.split("\n")

    # Assert that the content contains the fields.
    assert lines[0] == "B3*B**********"

def test_generate_edi_content_parser_error(
    organization: Organization, business_unit: BusinessUnit
) -> None:
    """Test Generate EDI content throws parser error if placeholders are not found, but passed.

    Args:
        organization (Organization): The organization instance.
        business_unit (BusinessUnit): The business unit instance.

    Returns:
        None: This function does not return anything.
    """
    order_1 = OrderFactory()
    user = UserFactory()

    order_movements = order_1.movements.all()
    order_movements.update(status="C")

    order_1.status = "C"
    order_1.save()

    billing_item = BillingQueue.objects.create(
        organization=organization,
        business_unit=business_unit,
        order=order_1,
        user=user,
        customer=order_1.customer,
    )

    _, fields, edi_billing_profile = factories.EDISegmentFactory(
        business_unit=business_unit,
        organization=organization,
    )
    fields.delete()

    with pytest.raises(exceptions.EDIParserError) as excinfo:
        helpers.generate_edi_content(
            billing_item=billing_item, edi_billing_profile=edi_billing_profile
        )

    assert (
        excinfo.value.args[0]
        == "Number of placeholders in parser does not match number of values for segment `B3`"
    )


def test_generate_edi_document(
    organization: Organization, business_unit: BusinessUnit
) -> None:
    order_1 = OrderFactory()
    user = UserFactory()

    order_movements = order_1.movements.all()
    order_movements.update(status="C")

    order_1.status = "C"
    order_1.save()

    billing_item = BillingQueue.objects.create(
        organization=organization,
        business_unit=business_unit,
        order=order_1,
        user=user,
        customer=order_1.customer,
    )

    _, _, edi_billing_profile = factories.EDISegmentFactory(
        business_unit=business_unit,
        organization=organization,
    )

    document = helpers.generate_edi_document(
        billing_queue_item=billing_item,
        edi_profile=edi_billing_profile,
    )

    # Split the document into lines
    lines = document.split("\n")

    # Assert that the document starts with ISA and ends with IEA
    assert lines[0].startswith("ISA*")
    assert lines[-1].startswith("IEA*")

    # Assert that the document has GS followed by GE
    assert "GS*" in lines[1]
    assert "GE*" in lines[-2]

    # Assert that the document has ST followed by SE
    st_index = [i for i, s in enumerate(lines) if "ST*" in s][0]
    se_index = [i for i, s in enumerate(lines) if "SE*" in s][0]
    assert st_index < se_index

    # Assert that BIG and N3 segments are in the document
    assert "N3*" in document
