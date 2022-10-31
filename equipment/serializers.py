# -*- coding: utf-8 -*-
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

from typing import Type

from rest_framework import serializers

from .models import Equipment


class EquipmentSerializer(serializers.ModelSerializer):
    """
    Serializer for Equipment Model.
    """

    class Meta:
        """
        Metaclass for EquipmentSerializer.
        """
        model: Type[Equipment] = Equipment
        fields: tuple[str, ...] = (
            "id",
            "equipment_type",
            "is_active",
            "description",
            "license_plate_number",
            "vin_number",
            "odometer",
            "engine_hours",
            "manufacturer",
            "manufactured_date",
            "model",
            "model_year",
            "state",
            "leased",
            "leased_date",
            "hos_exempt",
            "aux_power_unit_type",
            "fuel_draw_capacity",
            "num_of_axles",
            "transmission_manufacturer",
            "transmission_type",
            "has_berth",
            "has_electronic_engine",
            "highway_use_tax",
            "owner_operated",
            "ifta_qualified",
        )

    def to_representation(self, instance: Equipment) -> dict:
        """
        Serialize Equipment objects to JSON.
        """
        data = super().to_representation(instance)
        data = {k: v if v is not None else "" for k, v in data.items()}
        return data
