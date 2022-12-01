"""
COPYRIGHT 2022 MONTA

This file is part of Monta.

Monta is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

Monta is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with Monta.  If not, see <https://www.gnu.org/licenses/>.
"""


class FuelSurchargeService:
    """Fuel Surcharge

    This class generates fuel surcharge based on the following:
    - User input of fuel price range from and to (e.g. 1.00 - 1.50)
    - User input of fuel price increment (e.g. 0.01)
    - Base Charge (e.g. 0.50)
    - Fuel Surcharge Increment (e.g. 0.01)
    - Fuel Method (e.g. Percentage)

    Attributes:
        fuel_price_from (float): The fuel price range from.
        fuel_price_to (float): The fuel price range to.
        fuel_price_increment (float): The fuel price increment.
        base_charge (float): The base charge.
        fuel_surcharge_increment (float): The fuel surcharge increment.
        fuel_method (str): The fuel method.
    """

    def __init__(
        self,
        fuel_price_from: float,
        fuel_price_to: float,
        fuel_price_increment: float,
        base_charge: float,
        fuel_surcharge_increment: float,
        fuel_method: str,
    ) -> None:
        """Initialize Fuel Surcharge

        Args:
            fuel_price_from (float): The fuel price range from.
            fuel_price_to (float): The fuel price range to.
            fuel_price_increment (float): The fuel price increment.
            base_charge (float): The base charge.
            fuel_surcharge_increment (float): The fuel surcharge increment.
            fuel_method (str): The fuel method.
        """
        self.fuel_price_from = fuel_price_from
        self.fuel_price_to = fuel_price_to
        self.fuel_price_increment = fuel_price_increment
        self.base_charge = base_charge
        self.fuel_surcharge_increment = fuel_surcharge_increment
        self.fuel_method = fuel_method

    def generate_fuel_surcharge(self) -> dict[float, float]:
        """Generate Fuel Surcharge

        Generate fuel surcharge based on the following:
        - User input of fuel price range from and to (e.g. 1.00 - 1.50)
        - User input of fuel price increment (e.g. 0.01)
        - Base Charge (e.g. 0.50)
        - Fuel Surcharge Increment (e.g. 0.01)
        - Fuel Method (e.g. Percentage)

        Returns:
            dict: The fuel surcharge.
        """
        fuel_surcharge = {}
        fuel_price = self.fuel_price_from
        while fuel_price <= self.fuel_price_to:
            fuel_surcharge[fuel_price] = self.base_charge
            fuel_price += self.fuel_price_increment
        return fuel_surcharge
