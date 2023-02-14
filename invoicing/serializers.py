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

from invoicing import models
from utils.serializers import GenericSerializer


class InvoiceControlSerializer(GenericSerializer):
    """A serializer for the `InvoiceControl` model.

    A serializer class for the InvoiceControl model. This serializer is used
    to convert InvoiceControl model instances into a Python dictionary format
    that can be rendered into a JSON response. It also defined the fields that
    should be included in the serialized representation of the model
    """

    class Meta:
        """
        Metaclass for the InvoiceControlSerializer

        Attributes:
            model (InvoiceControl): The model that the serializer is for.
        """

        model = models.InvoiceControl
