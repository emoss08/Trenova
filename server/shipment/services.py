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
from shipment import helpers, models, selectors, types


def create_initial_movement(*, shipment: models.Shipment) -> None:
    """Create the initial movement for the given shipment.

    Args:
        shipment (Shipment): The shipment instance.

    Returns:
        None: This function does not return anything.
    """
    Movement.objects.create(
        organization=shipment.organization,
        business_unit=shipment.organization.business_unit,
        shipment=shipment,
    )


def combine_pdfs_service(*, shipment: models.Shipment) -> models.ShipmentDocumentation:
    """Combine all PDFs in shipment Document into one PDF file

    Args:
        shipment (Shipment): shipment to combine documents from

    Returns:
        ShipmentDocumentation: created ShipmentDocumentation
    """

    document_class = DocumentClassification.objects.get(name="CON")
    file_path = f"{settings.MEDIA_ROOT}/{shipment.id}.pdf"
    merger = PdfMerger()
    storage_class: Storage = get_storage_class()()

    if storage_class.exists(file_path):
        raise FileExistsError(f"File {file_path} already exists")

    for document in shipment.shipment_documentation.all():
        merger.append(document.document.path)

    merger.write(file_path)
    merger.close()

    consolidated_document = storage_class.open(file_path, "rb")

    documentation = models.ShipmentDocumentation.objects.create(
        organization=shipment.organization,
        shipment=shipment,
        document=consolidated_document,
        document_class=document_class,
    )

    storage_class.delete(file_path)

    return documentation


def gather_formula_variables(*, shipment: models.Shipment) -> types.FormulaVariables:
    """Gather all the variables needed for the formula

    Args:
        shipment (Shipment): The shipment instance

    Returns:
        FormulaVariables: A dictionary of variables that can be used in a formula.
    """
    return {
        "freight_charge": shipment.freight_charge_amount,
        "other_charge": shipment.other_charge_amount,
        "mileage": shipment.mileage,
        "weight": shipment.weight,
        "stops": selectors.get_shipment_stops(shipment=shipment).count(),
        "rating_units": shipment.rating_units,
        "equipment_cost_per_mile": shipment.equipment_type.cost_per_mile,
        "hazmat_additional_cost": shipment.hazmat.additional_cost
        if shipment.hazmat
        else 0,
        "temperature_differential": shipment.temperature_differential,
    }


def calculate_total(*, shipment: models.Shipment) -> Decimal:
    """Calculate the sub_total for an order

    Calculate the sub_total for the shipment if the organization 'ShipmentControl'
    has auto_total_shipment as True. If not, this method will be skipped in the
    save method.

    Returns:
        Decimal: The total for the order
    """

    # TODO(WOLFRED): This can be replaced with a dictionary lookup, this seems a bit verbose

    if not shipment.freight_charge_amount:
        return Decimal(0)

    freight_charge = Decimal(shipment.freight_charge_amount)
    other_charge = (
        Decimal(shipment.other_charge_amount)
        if shipment.other_charge_amount
        else Decimal(0)
    )

    # Calculate `FLAT` rating method
    if shipment.rate_method == models.RatingMethodChoices.FLAT:
        return freight_charge + other_charge

    # Calculate `PER_MILE` rating method
    if shipment.rate_method == models.RatingMethodChoices.PER_MILE and shipment.mileage:
        return (freight_charge * Decimal(shipment.mileage)) + other_charge

    # Calculate `PER_STOP` rating method
    if shipment.rate_method == models.RatingMethodChoices.PER_STOP:
        shipment_stops_count = selectors.get_shipment_stops(shipment=shipment).count()
        return (freight_charge * Decimal(shipment_stops_count)) + other_charge

    # Calculate `PER_POUND` rating method
    if (
        shipment.rate_method == models.RatingMethodChoices.POUNDS
        and shipment.weight > 0
    ):
        return (freight_charge * Decimal(shipment.weight)) + other_charge

    if shipment.rate_method == models.RatingMethodChoices.OTHER:
        if not shipment.formula_template:
            return (freight_charge * Decimal(shipment.rating_units)) + other_charge

        formula_text = shipment.formula_template.formula_text
        if helpers.validate_formula(formula=formula_text):
            variables = gather_formula_variables(shipment=shipment)
            return Decimal(helpers.evaluate_formula(formula=formula_text, **variables))
    return freight_charge


def handle_voided_shipment(shipment: models.Shipment) -> None:
    """If a shipment has the status of voided. Void all stops and movements."""

    shipment.status = models.StatusChoices.VOIDED
    shipment.ship_date = None
    shipment.transferred_to_billing = False
    shipment.billed = False
    shipment.billing_transfer_date = None

    # Void all related Movements and Stops
    shipment.movements.update(
        primary_worker=None,
        secondary_worker=None,
        tractor=None,
        status=models.StatusChoices.VOIDED,
    )
    stops = selectors.get_shipment_stops(shipment=shipment)
    stops.update(
        status=models.StatusChoices.VOIDED, arrival_time=None, departure_time=None
    )
