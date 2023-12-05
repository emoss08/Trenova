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
from collections.abc import Callable
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
        "hazmat_additional_cost": shipment.hazardous_material.additional_cost
        if shipment.hazardous_material
        else 0,
        "temperature_differential": shipment.temperature_differential,
    }


def calculate_total(*, shipment: models.Shipment) -> Decimal:
    """Calculate the total cost of a shipment based on a given rate method.

    This function will take in one of the predefined `RatingMethodChoices` from the
    `models` module and use the information to calculate the total cost. In the case where
    a `RatingMethodChoice` is not found, it will default to calculating based on the
    `freight_charge` alone.

    Args:
        shipment (models.Shipment): Shipment instance for which to calculate total cost.

    Returns:
        Decimal: Total cost calculated based on the rate method.
    """

    # Convert to Decimal once
    freight_charge = Decimal(shipment.freight_charge_amount or 0)
    other_charge = Decimal(shipment.other_charge_amount or 0)

    # Helper function to calculate based on a multiplier
    def calculate_with_multiplier(multiplier: Decimal) -> Decimal:
        return (freight_charge * multiplier) + other_charge

    # Mapping of rate methods to their calculation strategies
    rate_method_calculations: dict[str, Callable[[], Decimal]] = {
        models.RatingMethodChoices.FLAT: lambda: freight_charge + other_charge,
        models.RatingMethodChoices.PER_MILE: lambda: calculate_with_multiplier(
            Decimal(shipment.mileage or 0)
        ),
        models.RatingMethodChoices.PER_STOP: lambda: calculate_with_multiplier(
            Decimal(selectors.get_shipment_stops(shipment=shipment).count())
        ),
        models.RatingMethodChoices.POUNDS: lambda: calculate_with_multiplier(
            Decimal(shipment.weight or 0)
        ),
        models.RatingMethodChoices.OTHER: lambda: calculate_other_method(
            shipment, freight_charge, other_charge
        ),
    }

    # Perform calculation based on the rate method
    return rate_method_calculations.get(shipment.rate_method, lambda: freight_charge)()


def calculate_other_method(
    shipment: models.Shipment, freight_charge: Decimal, other_charge: Decimal
) -> Decimal:
    """Calculate the total cost for a shipment using an alternate method.

    This function is called within the larger calculate_total function specifically for
    'OTHER' rate methods.

    Args:
        shipment (object): Shipment data that includes the formula template.
        freight_charge (Decimal): Cost incurred for the transportation of goods.
        other_charge (Decimal): Additional cost not categorized under freight charge.

    Returns:
        Decimal: Total cost calculated based on the formula template or freight_charge.
    """
    if not shipment.formula_template:
        return (freight_charge * Decimal(shipment.rating_units)) + other_charge

    formula_text = shipment.formula_template.formula_text
    if helpers.validate_formula(formula=formula_text):
        variables = gather_formula_variables(shipment=shipment)
        return Decimal(helpers.evaluate_formula(formula=formula_text, **variables))
    return freight_charge


def handle_voided_shipment(shipment: models.Shipment) -> None:
    """Handles a shipment that was voided.

    This function sets the shipment status to 'VOIDED', nullifies the ship date,
    sets `transferred_to_billing` and `billed` fields to False, nullifies the `billing_transfer_date`,
    and update status of all related movements and stops.

    Args:
        shipment (models.Shipment): Shipment instance that was voided.

    Returns:
        None: This function does not return anything.
    """
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
