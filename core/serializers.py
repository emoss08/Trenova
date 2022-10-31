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

from typing import Any

from rest_framework import serializers


class DynamicModelSerializer(serializers.ModelSerializer):
    """
    A ModelSerializer that takes an additional `fields` argument that
    controls which fields should be displayed.
    """

    def get_default_field_names(
        self, declared_fields, model_info
    ) -> tuple | list | ...:
        """
        Return the default set of field names that should be used for the
        serializer.

        Args:
            declared_fields ():
            model_info ():

        Returns:
            tuple | list | ...: The default set of field names
        """
        field_names: list | tuple | ... = super().get_field_names(
            declared_fields, model_info
        )
        if self.dynamic_fields is not None:
            allowed = set(self.dynamic_fields)
            excluded_field_names: set = set(field_names) - allowed
            field_names = tuple(x for x in field_names if x not in excluded_field_names)
        return field_names

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        """This is the constructor for the DynamicModelSerializer class.

        Args:
            *args: Variable length argument list
            **kwargs: Arbitrary keyword arguments

        Returns:
            None
        """
        self.dynamic_fields = kwargs.pop("fields", None)
        self.read_only_fields = kwargs.pop("read_only_fields", [])
        super().__init__(*args, **kwargs)
