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
from django.contrib.contenttypes.models import ContentType

from accounts.tests.factories import UserFactory
from billing.models import BillingQueue
from billing.tests.factories import (
    AccessorialChargeFactory,
    DocumentClassificationFactory,
)
from document_generator import models
from document_generator.services import render_document
from order.tests.factories import AdditionalChargeFactory, OrderFactory
from organization.models import BusinessUnit, Organization

pytestmark = pytest.mark.django_db


# def test_generate_document_with_line_items(organization, business_unit):
#     # Create a sample Invoice model instance
#     order_1 = OrderFactory()
#     user = UserFactory()
#     doc_class = DocumentClassificationFactory()
#
#     order_movements = order_1.movements.all()
#     order_movements.update(status="C")
#
#     order_1.status = "C"
#     order_1.save()
#
#     accessorial_charge = AccessorialChargeFactory()
#
#     AdditionalChargeFactory(
#         order=order_1,
#         accessorial_charge=accessorial_charge,
#     )
#
#     AdditionalChargeFactory(
#         order=order_1,
#         accessorial_charge=accessorial_charge,
#     )
#
#     # Test first invoice
#     invoice = BillingQueue.objects.create(
#         organization=organization,
#         business_unit=business_unit,
#         order=order_1,
#         user=user,
#         customer=order_1.customer,
#     )
#
#     # Create a DocumentTemplate
#     doc_template = models.DocumentTemplate.objects.create(
#         organization=organization,
#         business_unit=business_unit,
#         name="Invoice Template",
#         content="""
#         Invoice Number: {{ invoice_invoice_number }}
#         Total Amount: {{ invoice_total_amount }}
#         Date: {{ invoice_bill_date }}
#
#         {% for line_item in invoice_line_items %}
#         Code: {{ line_item.line_item_code }}
#         Unit: {{ line_item.line_item_unit }}
#         Charge Amount: {{ line_item.line_item_charge_amount }}
#         {% endfor %}
#         """,
#         document_classification=doc_class,
#     )
#
#     # Create TemplateField
#     number_field = models.TemplateField.objects.create(
#         organization=organization,
#         business_unit=business_unit,
#         name="invoice.invoice_number",
#         label="Invoice Number",
#         type="string",
#         template=doc_template,
#     )
#
#     total_field = models.TemplateField.objects.create(
#         name="invoice.total_amount",
#         label="Total Amount",
#         type="money",
#         template=doc_template,
#         organization=organization,
#         business_unit=business_unit,
#     )
#
#     date_field = models.TemplateField.objects.create(
#         name="invoice.bill_date",
#         label="Date Field",
#         type="date",
#         template=doc_template,
#         organization=organization,
#         business_unit=business_unit,
#     )
#
#     # Create DocumentDataBinding for invoice number
#     models.DocumentDataBinding.objects.create(
#         organization=organization,
#         business_unit=business_unit,
#         placeholder="{invoice.invoice_number}",
#         content_type=ContentType.objects.get_for_model(BillingQueue),
#         field_name="invoice_number",
#         template=doc_template,
#         field=number_field,  # Associate with the TemplateField
#     )
#
#     # Create DocumentDataBinding for total amount
#     models.DocumentDataBinding.objects.create(
#         organization=organization,
#         business_unit=business_unit,
#         placeholder="{invoice.total_amount}",
#         content_type=ContentType.objects.get_for_model(BillingQueue),
#         field_name="total_amount",
#         template=doc_template,
#         field=total_field,  # Associate with the TemplateField
#     )
#
#     # Create DocumentDataBinding for total amount
#     models.DocumentDataBinding.objects.create(
#         organization=organization,
#         business_unit=business_unit,
#         placeholder="{invoice.bill_date}",
#         content_type=ContentType.objects.get_for_model(BillingQueue),
#         field_name="bill_date",
#         template=doc_template,
#         field=date_field,  # Associate with the TemplateField
#     )
#
#     # Create DocumentDataBinding for additional charges (Foreign key to the invoice.order model)
#     line_items_binding = models.DocumentDataBinding.objects.create(
#         organization=organization,
#         business_unit=business_unit,
#         placeholder="{invoice_line_items}",
#         content_type=ContentType.objects.get_for_model(invoice.order),
#         field_name="order.additional_charges",
#         template=doc_template,
#         is_list=True,
#     )
#
#     # Create DocumentTableColumnBinding for each field of the line item
#     charge_amount_field = models.TemplateField.objects.create(
#         name="line_item.charge_amount",
#         label="Charge Amount",
#         type="number",
#         template=doc_template,
#         organization=organization,
#         business_unit=business_unit,
#     )
#
#     unit_field = models.TemplateField.objects.create(
#         name="line_item.unit",
#         label="Unit",
#         type="number",
#         template=doc_template,
#         organization=organization,
#         business_unit=business_unit,
#     )
#
#     code_field = models.TemplateField.objects.create(
#         name="line_item.additional_charges.code",
#         label="Code",
#         type="string",
#         template=doc_template,
#         organization=organization,
#         business_unit=business_unit,
#     )
#
#     models.DocumentTableColumnBinding.objects.create(
#         organization=organization,
#         business_unit=business_unit,
#         table_binding=line_items_binding,
#         column_name="{line_item.charge_amount}",
#         field_name="charge_amount",
#         field=charge_amount_field,
#     )
#
#     models.DocumentTableColumnBinding.objects.create(
#         organization=organization,
#         business_unit=business_unit,
#         table_binding=line_items_binding,
#         column_name="{line_item.code}",
#         field_name="accessorial_charge__code",  # Updated this line
#         field=code_field,
#     )
#
#     models.DocumentTableColumnBinding.objects.create(
#         organization=organization,
#         business_unit=business_unit,
#         table_binding=line_items_binding,
#         column_name="{line_item.unit}",
#         field_name="unit",
#         field=unit_field,
#     )
#
#     print(render_document(doc_template, invoice))


def test_generate_document_with_styles(organization, business_unit) -> None:
    doc_theme = models.DocumentTheme.objects.create(
        organization=organization,
        business_unit=business_unit,
        name="Default Theme",
        css="""
            body { font-family: Arial, sans-serif; }
            table { border-collapse: collapse; width: 100%; }
            table, th, td { border: 1px solid black; }
            th, td { padding: 8px 12px; }
            .logo img {
                max-width: 200px;
                margin: 10px;
            }
            .invoice-header {
                text-align: center;
                margin-bottom: 20px;
            }
        """,
    )
    order_1 = OrderFactory()
    user = UserFactory()
    doc_class = DocumentClassificationFactory()

    order_movements = order_1.movements.all()
    order_movements.update(status="C")

    order_1.status = "C"
    order_1.save()

    accessorial_charge = AccessorialChargeFactory()

    AdditionalChargeFactory(
        order=order_1,
        accessorial_charge=accessorial_charge,
    )

    AdditionalChargeFactory(
        order=order_1,
        accessorial_charge=accessorial_charge,
    )

    # Test first invoice
    invoice = BillingQueue.objects.create(
        organization=organization,
        business_unit=business_unit,
        order=order_1,
        user=user,
        customer=order_1.customer,
        bill_type="INVOICE",
    )

    # Create a DocumentTemplate
    doc_template = models.DocumentTemplate.objects.create(
        organization=organization,
        business_unit=business_unit,
        name="Invoice Template",
        content="""
        <div class="invoice-header">
        {% if invoice_organization.logo %}
            <div class="logo">
                <img src="{{ invoice_organization.logo.url }}" />
            </div>
        {% else %}
            <h1>{{ invoice_organization }}</h1>
        {% endif %}
            <p>Invoice Number: {{ invoice_invoice_number }}</p>
            <p>Total Amount: {{ invoice_total_amount }}</p>
            <p>Date: {{ invoice_bill_date }}</p>
            {% if invoice_bill_type == "INVOICE" %}
            <p>Bill Type: Invoice</p>
            {% elif invoice_bill_type == "CREDIT" %}
            <p>Bill Type: Credit</p>
            {% elif invoice_bill_type == "DEBIT" %}
            <p>Bill Type: Debit</p>
            {% endif %}
        </div>

        <table class="line-items">
            <thead>
                <tr>
                    <th>Code</th>
                    <th>Unit</th>
                    <th>Charge Amount</th>
                </tr>
            </thead>
            <tbody>
            {% for line_item in invoice_line_items %}
                <tr>
                    <td>{{ line_item.line_item_code }}</td>
                    <td>{{ line_item.line_item_unit }}</td>
                    <td>{{ line_item.line_item_charge_amount }}</td>
                </tr>
            {% endfor %}
            </tbody>
        </table>
        """,
        document_classification=doc_class,
        theme=doc_theme,
    )

    models.DocTemplateCustomization.objects.create(
        organization=organization,
        business_unit=business_unit,
        doc_template=doc_template,
        css_selector=".invoice-footer",
        property_name="color",
        property_value="blue",
    )

    models.DocTemplateCustomization.objects.create(
        organization=organization,
        business_unit=business_unit,
        doc_template=doc_template,
        css_selector=".line-items",
        property_name="background-color",
        property_value="#f2f2f2",  # A light gray background
    )

    # Create TemplateField
    number_field = models.TemplateField.objects.create(
        organization=organization,
        business_unit=business_unit,
        name="invoice.invoice_number",
        label="Invoice Number",
        type="string",
        template=doc_template,
    )

    total_field = models.TemplateField.objects.create(
        name="invoice.total_amount",
        label="Total Amount",
        type="money",
        template=doc_template,
        organization=organization,
        business_unit=business_unit,
    )

    date_field = models.TemplateField.objects.create(
        name="invoice.bill_date",
        label="Date Field",
        type="date",
        template=doc_template,
        organization=organization,
        business_unit=business_unit,
    )
    bill_type_field = models.TemplateField.objects.create(
        name="invoice.bill_type",
        label="Bill Type",
        type="string",
        template=doc_template,
        organization=organization,
        business_unit=business_unit,
    )
    organization_field = models.TemplateField.objects.create(
        name="invoice.organization",
        label="Organization",
        type="string",
        template=doc_template,
        organization=organization,
        business_unit=business_unit,
    )

    models.DocumentDataBinding.objects.create(
        organization=organization,
        business_unit=business_unit,
        placeholder="{invoice.invoice_number}",
        content_type=ContentType.objects.get_for_model(BillingQueue),
        field_name="invoice_number",
        template=doc_template,
        field=number_field,
    )

    models.DocumentDataBinding.objects.create(
        organization=organization,
        business_unit=business_unit,
        placeholder="{invoice.organization}",
        content_type=ContentType.objects.get_for_model(BillingQueue),
        field_name="organization",
        template=doc_template,
        field=organization_field,
    )

    models.DocumentDataBinding.objects.create(
        organization=organization,
        business_unit=business_unit,
        placeholder="{invoice.total_amount}",
        content_type=ContentType.objects.get_for_model(BillingQueue),
        field_name="total_amount",
        template=doc_template,
        field=total_field,
    )

    models.DocumentDataBinding.objects.create(
        organization=organization,
        business_unit=business_unit,
        placeholder="{invoice.bill_date}",
        content_type=ContentType.objects.get_for_model(BillingQueue),
        field_name="bill_date",
        template=doc_template,
        field=date_field,
    )

    models.DocumentDataBinding.objects.create(
        organization=organization,
        business_unit=business_unit,
        placeholder="{invoice.bill_type}",
        content_type=ContentType.objects.get_for_model(BillingQueue),
        field_name="bill_type",
        template=doc_template,
        field=bill_type_field,  # Associate with the TemplateField
    )

    # Create DocumentDataBinding for additional charges (Foreign key to the invoice.order model)
    line_items_binding = models.DocumentDataBinding.objects.create(
        organization=organization,
        business_unit=business_unit,
        placeholder="{invoice_line_items}",
        content_type=ContentType.objects.get_for_model(invoice.order),
        field_name="order.additional_charges",
        template=doc_template,
        is_list=True,
    )

    # Create DocumentTableColumnBinding for each field of the line item
    charge_amount_field = models.TemplateField.objects.create(
        name="line_item.charge_amount",
        label="Charge Amount",
        type="number",
        template=doc_template,
        organization=organization,
        business_unit=business_unit,
    )

    unit_field = models.TemplateField.objects.create(
        name="line_item.unit",
        label="Unit",
        type="number",
        template=doc_template,
        organization=organization,
        business_unit=business_unit,
    )

    code_field = models.TemplateField.objects.create(
        name="line_item.additional_charges.code",
        label="Code",
        type="string",
        template=doc_template,
        organization=organization,
        business_unit=business_unit,
    )

    # Table column bindings
    models.DocumentTableColumnBinding.objects.create(
        organization=organization,
        business_unit=business_unit,
        table_binding=line_items_binding,
        column_name="{line_item.charge_amount}",
        field_name="charge_amount",
        field=charge_amount_field,
    )

    models.DocumentTableColumnBinding.objects.create(
        organization=organization,
        business_unit=business_unit,
        table_binding=line_items_binding,
        column_name="{line_item.code}",
        field_name="accessorial_charge__code",  # Updated this line
        field=code_field,
    )

    models.DocumentTableColumnBinding.objects.create(
        organization=organization,
        business_unit=business_unit,
        table_binding=line_items_binding,
        column_name="{line_item.unit}",
        field_name="unit",
        field=unit_field,
    )

    print(render_document(template=doc_template, instance=invoice))


def test_save_template_version(
    organization: Organization, business_unit: BusinessUnit
) -> None:
    """Test saving new template version signal.

    Args:
        organization(Organization): Organization object.
        business_unit(BusinessUnit): Business Unit object.

    Returns:
        None: This function does not return anything.
    """
    doc_class = DocumentClassificationFactory()
    template = models.DocumentTemplate.objects.create(
        organization=organization,
        business_unit=business_unit,
        name="Invoice Template",
        content="""
        Invoice Number: {{ invoice_invoice_number }}
        Total Amount: {{ invoice_total_amount }}
        Date: {{ invoice_bill_date }}
        """,
        document_classification=doc_class,
    )

    print(template)

    print(template.current_version)
