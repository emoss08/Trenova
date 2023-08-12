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
import re
from functools import reduce
from typing import Any

from django.core.exceptions import FieldDoesNotExist
from django.db.models import Field, Model
from django.db.models.fields.related import ForeignKey
from django.utils import timezone

from billing.models import BillingQueue
from edi import exceptions, models


def generate_edi_envelope_headers(
    *, edi_profile: models.EDIBillingProfile, date: str, time: str
) -> str:
    """Generate EDI X12 ISA and GS envelope headers and trailers.

    Args:
        edi_profile (models.EDIBillingProfile): The EDI profile instance.
        date (str): The current date in 'YYMMDD' format.
        time (str): The current time in 'HHMM' format.

    Returns:
        str: The EDI envelope headers and trailers.
    """

    # Interchange Control Header (ISA)
    isa_header = (
        "ISA*{auth_info_qualifier}*{auth_info}*"
        "{security_info_qualifier}*{security_info}*"
        "{interchange_id_qualifier_sender}*{interchange_sender_id}*"
        "{interchange_id_qualifier_receiver}*{interchange_receiver_id}*"
        "{date}*{time}*U*00401*{control_number}*{ack_requested}*{usage_indicator}*>"
    ).format(
        auth_info_qualifier=edi_profile.edi_isa_authority.zfill(2),
        auth_info=edi_profile.edi_isa_id.ljust(10),
        security_info_qualifier=edi_profile.edi_isa_security.zfill(2),
        security_info=edi_profile.edi_isa_security_info.ljust(10),
        interchange_id_qualifier_sender=edi_profile.edi_isa_interchange_id_qualifier.zfill(
            2
        ),
        interchange_sender_id=edi_profile.edi_gs_id.ljust(15),
        interchange_id_qualifier_receiver=edi_profile.edi_isa_interchange_id_qualifier.zfill(
            2
        ),
        interchange_receiver_id=edi_profile.edi_isa_receiver_id.ljust(15),
        date=date,
        time=time,
        control_number="000000001",
        ack_requested="1" if edi_profile.edi_functional_ack else "0",
        usage_indicator="T" if edi_profile.edi_test_mode else "P",
    )

    # Functional Group Header (GS)
    gs_header = (
        "GS*{functional_identifier_code}*{application_senders_code}*"
        "{application_receivers_code}*{date}*{time}*{group_control_number}*"
        "X*{responsible_agency_code}*{version_release_industry_id_code}"
    ).format(
        functional_identifier_code=edi_profile.edi_gs_code,
        application_senders_code=edi_profile.edi_gs_id,
        application_receivers_code=edi_profile.edi_gs_application_receiver_id,
        date=date,
        time=time,
        group_control_number="1",
        responsible_agency_code="004010",
        version_release_industry_id_code=edi_profile.edi_version,
    )

    # Transaction Set Header (ST)
    st_header = (
        "ST*{transaction_set_identifier_code}*{transaction_set_control_number}"
    ).format(
        transaction_set_identifier_code="210",
        transaction_set_control_number="1",  # TODO(Wolfred): This should increment by 1 for each transaction set
    )

    return f"{isa_header}\n{gs_header}\n{st_header}"


def generate_edi_trailers() -> str:
    """Generates EDI interchange and functional group trailer segments.

    Returns:
        str: The generated EDI trailers as a string.

    Details:
        Generates the 2 standard EDI trailers:

        - Functional Group Trailer (GE)
          Indicates the end of a functional group.
          Hardcoded to 'GE*1*1' (1 functional group with 1 document).

        - Interchange Control Trailer (IEA)
          Indicates the end of the interchange.
          Hardcoded to 'IEA*1*000000001' (1 interchange with control # 000000001).

        The trailers are joined with a newline and returned as a string.
    """

    # SE Transaction Set Trailer (SE)
    # TODO(Wolfred): First value should be the number of segments in the transaction set
    # TODO(Wolfred): Second Number should be the transaction set control number
    # Which in the ST header is currently 1 ,but should be the same value as transaction_set_control_number
    se_trailer = "SE*1*1"

    # Functional Group Trailer (GE)
    ge_trailer = "GE*1*1"

    # Interchange Control Trailer (IEA)
    iea_trailer = "IEA*1*000000001"

    return f"{se_trailer}\n{ge_trailer}\n{iea_trailer}"


def get_nested_attr(*, obj: BillingQueue, attr: str) -> Any:
    """Get a nested attribute from an object

    Args:
        obj (BillingQueue): BillingQueue object
        attr (str): Attribute to get from the object

    Returns:
        Any: The value of the attribute
    """
    try:
        obj = reduce(getattr, attr.split("."), obj)
    except AttributeError as e:
        raise exceptions.EDIInvalidFieldException(
            f"Field `{attr}` does not exist on BillingQueue model."
        ) from e

    return obj


def generate_edi_content(
    *, billing_item: BillingQueue, edi_billing_profile: models.EDIBillingProfile
) -> str:
    # Initialize the segments' list
    segments = []

    # Get the EDI segments defined in the profile ordered by sequence
    edi_segments = edi_billing_profile.segments.order_by("sequence")

    # Cache for compiled regex patterns
    regex_cache = {}

    for edi_segment in edi_segments:
        # Validate the number of placeholders matches the number of values
        if edi_segment.parser.count("%s") != edi_segment.fields.count():
            raise exceptions.EDIParserError(
                f"Number of placeholders in parser does not match number of fields for segment `{edi_segment.code}`"
            )

        # Get the defined fields
        fields = edi_segment.fields.all()

        # Initialize the values' list
        values = []
        for field in fields:
            # Lookup the value for each field from the billing queue object
            value = get_nested_attr(obj=billing_item, attr=field.model_field) or ""

            # Convert value to string if not a string
            if not isinstance(value, str):
                value = str(value)

            # Compile regex if not in cache
            if field.validation_regex not in regex_cache:
                regex_cache[field.validation_regex] = re.compile(field.validation_regex)

            m = re.match(regex_cache[field.validation_regex], value)

            # Check if field has `validation_regex` defined
            if field.validation_regex and not m:
                raise exceptions.EDIFieldValidationError(
                    f"Value `{value}` for field `{field.model_field}` does not match regex `{field.validation_regex}`"
                )

            values.append(value)

        # Use the segment parser string to format the values
        segment = edi_segment.parser % tuple(values)

        # Append the formatted segment string
        segments.append(segment)

    # Join the segment strings with newlines and return
    return "\n".join(segments)


def generate_edi_document(
    *, billing_queue_item: BillingQueue, edi_profile: models.EDIBillingProfile
) -> str:
    """Generate an EDI document for a BillingQueue item based on the given EDI profile.

    Args:
        billing_queue_item (models.BillingQueue): The BillingQueue item instance.
        edi_profile (models.EDIBillingProfile): The EDI profile instance.

    Returns:
        str: The EDI document string.
    """

    # Get the current date and time
    now = timezone.now()
    date = now.strftime("%y%m%d")
    time = now.strftime("%H%M")

    # Generate the envelope headers
    envelope = generate_edi_envelope_headers(
        edi_profile=edi_profile, date=date, time=time
    )

    # Generate the document content
    content = generate_edi_content(
        billing_item=billing_queue_item, edi_billing_profile=edi_profile
    )

    # Generate the envelope trailers
    trailers = generate_edi_trailers()

    return f"{envelope}\n{content}\n{trailers}"


def _get_actual_field(
    *, model: type[Model], fields_chain: list[str]
) -> Field[Any, Any]:
    """Recursively retrieves the actual field in a model given a fields chain list.

    This function navigates through a model's fields using the fields_chain list.
    It starts from the model represented by 'model' argument, and goes deeper into related models
    (ForeignKey fields) if necessary and if they exist in the fields_chain list.

    Raises an easy-to-understand exception if a field from the fields_chain does not exist on its corresponding model.

    Args:
        model (Model): An instance of Django's Model. Is the starting point for field lookup.
        fields_chain (list[str]): A list of field names from 'model' and its related models. Must
            represent a valid chain of fields starting from 'model'.

    Returns:
        Field: The actual field object in a model derived from the fields_chain list.

    Raises:
        InvalidFieldException: A custom Exception. Raised when a field does not exist on its corresponding model.

    Notes:
        There is no reason to use this function outside of this module. It is only used by the 'get_actual_field' function.

    Examples:
        >>> from accounts.models import User
        >>> actual_field = _get_actual_field(User, ['user_profile', 'date_of_birth'])
    """
    try:
        current_field = model._meta.get_field(fields_chain[0])
    except FieldDoesNotExist as e:
        raise exceptions.EDIInvalidFieldException(
            f"Field '{fields_chain[0]}' does not exist on the {model.__name__} model."
        ) from e

    # If this is the last field in the chain, or it's not a ForeignKey, return it
    if len(fields_chain) == 1 or not isinstance(current_field, ForeignKey):
        return current_field  # type: ignore

    # Otherwise, continue with the next model
    return _get_actual_field(
        model=current_field.related_model, fields_chain=fields_chain[1:]
    )


def validate_data_type(*, data_type: str, model_field: str) -> tuple[bool, str]:
    """Validates that the internal type of a field corresponds to a given data type.

    This function extracts the actual field from the 'BillingQueue' model (or a model related to it)
    using a list of field names represented as a string separated by periods ('.'). It then verifies that
    the internal type of the derived field is equal to the provided 'data_type'.

    Args:
        data_type (str): A string representing the expected data type of the field.
            Corresponds to the internal type used by Django's ORM.
        model_field (str): A string of field names from 'BillingQueue' model (or a model related to it),
            separated by periods ('.'). Must represent a valid chain of fields starting from 'BillingQueue'.

    Returns:
        bool: True if the internal type of the field is equal to the provided 'data_type';
              otherwise, False.

    Examples:
        >>> is_valid = validate_data_type('CharField', 'customer.name')
    """

    # Split the fields chain into a list
    fields_chain = model_field.split(".")

    # Get the actual field from the BillingQueue model
    actual_field = _get_actual_field(model=BillingQueue, fields_chain=fields_chain)

    return (
        actual_field.get_internal_type() == data_type,
        actual_field.get_internal_type(),
    )
