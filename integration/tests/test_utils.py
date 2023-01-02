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

import pytest

from integration.models import Integration
from integration.utils import IntegrationBase


class TestIntegrationBase:
    """
    Test the IntegrationBase utility class.
    """

    def test_check_method(self):
        """
        Test the _check method.
        """
        integration_base = IntegrationBase()
        integration_base.headers = {"test": "test"}
        integration_base.model = Integration
        integration_base._check()
        assert integration_base.headers == {"test": "test"}
        assert integration_base.model == Integration

    def test_check_with_invalid_data(self):
        """
        Test the _check method with invalid data.
        """
        integration_base = IntegrationBase()
        integration_base.headers = "test"
        integration_base.model = "test"
        with pytest.raises(TypeError):
            integration_base._check()
