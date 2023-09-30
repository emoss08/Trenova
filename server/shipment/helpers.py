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

import tokenize
import typing
from io import BytesIO

from sympy import SympifyError, sympify

FORMULA_ALLOWED_VARIABLES = [
    "freight_charge",
    "other_charge",
    "mileage",
    "weight",
    "stops",
    "rating_units",
    "equipment_cost_per_mile",
    "hazmat_additional_cost",
    "temperature_differential",
]


def extract_variable_from_formula(*, formula: str) -> typing.List[str]:
    """Extract the variables from a given mathematical formula.

    This function tokenizes the input formula using Python's tokenize module. It
    extracts all tokens of the type 'NAME' (which often represent variable names in a formula)
    and returns them as a set.

    Args:
        formula (str): A string representing a mathematical formula from which variables are to be extracted.

    Returns:
        typing.List[str]: A set of strings representing the variable names found in the formula.
    """
    tokens = tokenize.tokenize(BytesIO(formula.encode("utf-8")).readline)
    return [token.string for token in tokens if token.type == tokenize.NAME]


def validate_formula(*, formula: str) -> bool:
    """Validate the input formula using sympify from sympy.

    This function takes a mathematical formula as a string, attempts to sympify it using
    sympy's sympify function, and returns True if sympify is successful (meaning the formula is
    valid). If sympify raises a SympifyError, the function catches the exception and returns
    False, indicating the formula is not valid.

    Args:
        formula (str): A string representing the mathematical formula to be validated.

    Returns:
        bool: True if the formula is valid, False otherwise.

    Raises:
        Does not raise any exceptions but catches SympifyError exceptions raised by sympify.
    """
    try:
        sympify(formula)
        return True
    except SympifyError:
        return False


def evaluate_formula(*, formula: str, **kwargs: typing.Any) -> float:
    """Evaluate a given mathematical formula with the provided variables.

    This function takes a mathematical formula as a string and a variable number of
    keyword arguments representing the symbols in the formula and their corresponding
    values. The formula is then evaluated by first converting it into a sympy expression
    using sympify. A check is performed to ensure that all the symbols in the expression
    are among the keys provided as keyword arguments. If the check passes, the symbols
    in the expression are then substituted with their corresponding values and the
    resultant float value of the expression is returned. If the check fails, a ValueError
    is raised.

    Args:
        formula (str): A string representing a mathematical formula to be evaluated.
        **kwargs (typing.Any): Arbitrary keyword arguments. The keys represent the symbols
            (variables) in the formula, the values are used to substitute these symbols in the expression.

    Returns:
        float: The resultant float value after evaluating the expression.

    Raises:
        ValueError: If there are any symbols in the formula that are not provided as keyword arguments.
    """
    expression = sympify(formula)

    # Ensure only allowed symbols are in the formula
    allowed_symbols = set(kwargs.keys())
    formula_symbols = {str(symbol) for symbol in expression.free_symbols}
    if not formula_symbols.issubset(allowed_symbols):
        raise ValueError("Invalid formula")

    return float(expression.subs(kwargs))
