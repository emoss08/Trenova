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

from accounts.tests.factories import UserFactory
from billing.models import BillingQueue
from edi import helpers, models
from order.tests.factories import OrderFactory

pytestmark = pytest.mark.django_db


def test_generate_edi_document(organization, business_unit) -> None:
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

    # Create EDI Segment
    segment = models.EDISegment.objects.create(
        business_unit=business_unit,
        organization=organization,
        code="BIG",
        name="Beginning Segment for Invoice",
        parser="BIG*%s*%s**%s*%s",
        sequence=1,
    )

    segment_field_1 = models.EDISegmentField.objects.create(
        business_unit=business_unit,
        organization=organization,
        edi_segment=segment,
        model_field="invoice_number",
        position=2,
    )
    segment_field_2 = models.EDISegmentField.objects.create(
        business_unit=business_unit,
        organization=organization,
        edi_segment=segment,
        model_field="mileage",
        position=3,
    )

    segment_field_3 = models.EDISegmentField.objects.create(
        business_unit=business_unit,
        organization=organization,
        edi_segment=segment,
        model_field="bill_date",
        format="%Y%m%d",  # if the field is a date
        position=4,
    )

    segment_field_4 = models.EDISegmentField.objects.create(
        business_unit=business_unit,
        organization=organization,
        edi_segment=segment,
        model_field="weight",
        position=5,
    )

    # Create second EDI Segment
    segment2 = models.EDISegment.objects.create(
        business_unit=business_unit,
        organization=organization,
        code="N3",
        name="Beginning Segment for Invoice",
        parser="N3*%s",
        sequence=2,
    )

    models.EDISegmentField.objects.create(
        business_unit=business_unit,
        organization=organization,
        edi_segment=segment2,
        model_field="order.origin_location.city",
        format="%Y%m%d",  # if the field is a date
        position=1,
    )

    edi_billing_profile = models.EDIBillingProfile.objects.create(
        business_unit=business_unit,
        organization=organization,
        customer=billing_item.customer,
        edi_enabled=True,
        edi_format="X12",
        destination="http://www.example.com",
        username="test_username",
        password="test_password",
        edi_isa_id="SenderISAId",  # typically a DUNS number or another identifier assigned to you
        edi_gs_id="SenderGSId",  # usually the same as the ISA ID, but not always
        edi_version="4010",  # version of the X12 standards you're using (4010 in this case)
        edi_test_mode=True,
        edi_functional_ack=True,
        edi_ta1_timeout=100,
        edi_997_ack=True,
        edi_gs3_receiver_id="ReceiverGSId",  # typically the receiver's DUNS number or other ID
        edi_gs2_application_sender_id="SenderGSId",  # usually the same as the GS ID
        processing_settings='{"test": "test"}',
        validation_settings='{"test": "test"}',
        edi_isa_authority="U",  # "U" for U.S. Department of Transportation, "X" for Accredited Standards Committee X12
        edi_isa_security="01",  # "01" for password
        edi_isa_security_info="Password",  # the actual password
        edi_isa_interchange_id_qualifier="01",  # "01" for DUNS (Dun & Bradstreet), "14" for D-U-N-S+4 (Dun & Bradstreet)
        edi_gs_code="PO",  # functional identifier code, "PO"  for purchase order
        edi_isa_receiver_id="ReceiverISAId",  # typically the receiver's DUNS number or other ID
        edi_gs_application_receiver_id="RECEIVERAPP1",  # usually the same as the GS ID
    )

    segment_st = models.EDISegment.objects.create(
        business_unit=business_unit,
        organization=organization,
        code="ST",
        name="Transaction Set Header",
        parser="ST*%s*%s",
        sequence=1,
    )

    models.EDISegmentField.objects.create(
        business_unit=business_unit,
        organization=organization,
        edi_segment=segment_st,
        model_field="invoice_number",
        position=1,
    )
    models.EDISegmentField.objects.create(
        business_unit=business_unit,
        organization=organization,
        edi_segment=segment_st,
        model_field="order.pro_number",
        position=2,
    )

    segment_se = models.EDISegment.objects.create(
        business_unit=business_unit,
        organization=organization,
        code="SE",
        name="Transaction Set Header",
        parser="SE*%s*%s",
        sequence=1,
    )

    models.EDISegmentField.objects.create(
        business_unit=business_unit,
        organization=organization,
        edi_segment=segment_se,
        model_field="invoice_number",
        position=1,
    )
    models.EDISegmentField.objects.create(
        business_unit=business_unit,
        organization=organization,
        edi_segment=segment_se,
        model_field="order.pro_number",
        position=2,
    )


    edi_billing_profile.segments.add(segment)
    edi_billing_profile.segments.add(segment2)
    edi_billing_profile.segments.add(segment_st)
    edi_billing_profile.segments.add(segment_se)

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
    st_index = [i for i, s in enumerate(lines) if 'ST*' in s][0]
    se_index = [i for i, s in enumerate(lines) if 'SE*' in s][0]
    assert st_index < se_index

    # Assert that BIG and N3 segments are in the document
    assert "BIG*" in document
    assert "N3*" in document
