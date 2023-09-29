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
from decimal import Decimal

from django.conf import settings
from django.core.files.storage import Storage, get_storage_class
from pypdf import PdfMerger

from billing.models import DocumentClassification
from movements.models import Movement
from order import helpers, models, selectors, types


def create_initial_movement(*, order: models.Order) -> None:
    """Create the initial movement for the given order.

    Args:
        order (Order): The order instance.

    Returns:
        None: This function does not return anything.
    """
    Movement.objects.create(
        organization=order.organization,
        business_unit=order.organization.business_unit,
        order=order,
    )


def combine_pdfs_service(*, order: models.Order) -> models.OrderDocumentation:
    """Combine all PDFs in Order Document into one PDF file

    Args:
        order (Order): Order to combine documents from

    Returns:
        OrderDocumentation: created OrderDocumentation
    """

    document_class = DocumentClassification.objects.get(name="CON")
    file_path = f"{settings.MEDIA_ROOT}/{order.id}.pdf"
    merger = PdfMerger()
    storage_class: Storage = get_storage_class()()

    if storage_class.exists(file_path):
        raise FileExistsError(f"File {file_path} already exists")

    for document in order.order_documentation.all():
        merger.append(document.document.path)

    merger.write(file_path)
    merger.close()

    consolidated_document = storage_class.open(file_path, "rb")

    documentation = models.OrderDocumentation.objects.create(
        organization=order.organization,
        order=order,
        document=consolidated_document,
        document_class=document_class,
    )

    storage_class.delete(file_path)

    return documentation


def gather_formula_variables(*, order: models.Order) -> types.FormulaVariables:
    """Gather all the variables needed for the formula

    Args:
        order (Order): The order instance

    Returns:
        FormulaVariables: A dictionary of variables that can be used in a formula.
    """
    return {
        "freight_charge": order.freight_charge_amount,
        "other_charge": order.other_charge_amount,
        "mileage": order.mileage,
        "weight": order.weight,
        "stops": selectors.get_order_stops(order=order).count(),
        "rating_units": order.rating_units,
        "equipment_cost_per_mile": order.equipment_type.cost_per_mile,
        "hazmat_additional_cost": order.hazmat.additional_cost if order.hazmat else 0,
        "temperature_differential": order.temperature_differential,
    }


def calculate_total(*, order: models.Order) -> Decimal:
    """Calculate the sub_total for an order

    Calculate the sub_total for the order if the organization 'OrderControl'
    has auto_total_order as True. If not, this method will be skipped in the
    save method.

    Returns:
        Decimal: The total for the order
    """

    # TODO(WOLFRED): This can be replaced with a dictionary lookup, this seems a bit verbose

    if not order.freight_charge_amount:
        return Decimal(0)

    freight_charge = Decimal(order.freight_charge_amount)
    other_charge = (
        Decimal(order.other_charge_amount) if order.other_charge_amount else Decimal(0)
    )

    # Calculate `FLAT` rating method
    if order.rate_method == models.RatingMethodChoices.FLAT:
        return freight_charge + other_charge

    # Calculate `PER_MILE` rating method
    if order.rate_method == models.RatingMethodChoices.PER_MILE and order.mileage:
        return (freight_charge * Decimal(order.mileage)) + other_charge

    # Calculate `PER_STOP` rating method
    if order.rate_method == models.RatingMethodChoices.PER_STOP:
        order_stops_count = selectors.get_order_stops(order=order).count()
        return (freight_charge * Decimal(order_stops_count)) + other_charge

    # Calculate `PER_POUND` rating method
    if order.rate_method == models.RatingMethodChoices.POUNDS and order.weight > 0:
        return (freight_charge * Decimal(order.weight)) + other_charge

    if order.rate_method == models.RatingMethodChoices.OTHER:
        if not order.formula_template:
            return (freight_charge * Decimal(order.rating_units)) + other_charge

        formula_text = order.formula_template.formula_text
        if helpers.validate_formula(formula=formula_text):
            variables = gather_formula_variables(order=order)
            return Decimal(helpers.evaluate_formula(formula=formula_text, **variables))
    return freight_charge
