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

import factory

from edi import models
from edi.tests.data import test_edi_segments


class EDIBillingProfileFactory(factory.django.DjangoModelFactory):
    """
    Factory for EDIBillingProfile Model.
    """

    class Meta:
        model = "edi.EDIBillingProfile"

    organization = factory.SubFactory("organizations.factories.OrganizationFactory")
    business_unit = factory.SubFactory("organizations.factories.BusinessUnitFactory")
    customer = factory.SubFactory("customer.factories.CustomerFactory")
    edi_enabled = True
    destination = "http://www.example.com"
    edi_format = "X12"
    username = "username"
    password = "password"
    edi_isa_id = factory.Sequence(lambda n: "ISA_ID_%d" % n)
    edi_gs_id = factory.Sequence(lambda n: "GS_ID_%d" % n)
    edi_version = "4010"
    edi_test_mode = True
    edi_functional_ack = True
    edi_ta1_timeout = 100
    edi_997_ack = True
    edi_gs3_receiver_id = factory.Sequence(lambda n: "GS3_RECEIVER_%d" % n)
    edi_gs2_application_sender_id = factory.Sequence(lambda n: "SenderGSId_%d" % n)
    processing_settings = '{"test": "test"}'
    validation_settings = '{"test": "test"}'
    edi_isa_authority = "U"
    edi_isa_security = "01"
    edi_isa_security_info = "password"
    edi_isa_interchange_id_qualifier = "01"
    edi_gs_code = "PO"
    edi_isa_receiver_id = factory.Sequence(lambda n: "ReceiverISAId_%d" % n)
    edi_gs_application_receiver_id = factory.Sequence(lambda n: "RECEIVERAPP1_%d" % n)


class EDISegmentFactory(factory.django.DjangoModelFactory):
    """
    Factory for EDISegment Model.
    """

    class Meta:
        model = "edi.EDISegment"

    organization = factory.SubFactory("organizations.factories.OrganizationFactory")
    business_unit = factory.SubFactory("organizations.factories.BusinessUnitFactory")

    @classmethod
    def _create(cls, model_class, *args, **kwargs):
        for segment in test_edi_segments:
            models.EDISegment.objects.create(
                business_unit=kwargs["business_unit"],
                organization=kwargs["organization"],
                code=segment["code"],
                name=segment["name"],
                parser=segment["parser"],
                sequence=segment["sequence"],
            )

        # Create fields for each segment based on number of %s in parser.
        # Pick random fields from BillingQueue model, and order model.
        for segment in models.EDISegment.objects.all():
            if segment.parser.count("%s") > 0:
                for i in range(segment.parser.count("%s")):
                    models.EDISegmentField.objects.create(
                        business_unit=kwargs["business_unit"],
                        organization=kwargs["organization"],
                        edi_segment=segment,
                        model_field="order.pieces",
                        position=i,
                    )

        edi_billing_profile = EDIBillingProfileFactory(
            organization=kwargs["organization"], business_unit=kwargs["business_unit"]
        )
        segments = models.EDISegment.objects.all()
        fields = models.EDISegmentField.objects.all()

        # Add all segments to edi_billing_profile
        edi_billing_profile.segments.add(*segments)
        return segments, fields, edi_billing_profile
