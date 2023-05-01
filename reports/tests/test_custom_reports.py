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

import pytest
import uuid

from django.core.exceptions import ValidationError
from pydantic import BaseModel

pytestmark = pytest.mark.django_db


class CustomReportBase(BaseModel):
    """
    Custom report Base Schema
    """

    organization_id: uuid.UUID
    name: str
    table: str


class CustomReportCreate(CustomReportBase):
    """
    Custom report Create Schema
    """

    pass


class CustomReportUpdate(CustomReportBase):
    """
    Custom report Update Schema
    """

    id: uuid.UUID


def test_create_custom_report_schema() -> None:
    """
    Test Custom Report Create Schema
    """
    custom_report_create = CustomReportCreate(
        organization_id=uuid.uuid4(), name="Test", table="test"
    )

    custom_report = custom_report_create.dict()

    assert custom_report is not None
    assert custom_report["organization_id"] is not None
    assert custom_report["name"] == "Test"
    assert custom_report["table"] == "test"


def test_update_custom_report_schema() -> None:
    """
    Test Custom Report Update Schema
    """
    custom_report_update = CustomReportUpdate(
        id=uuid.uuid4(), organization_id=uuid.uuid4(), name="Test", table="test"
    )

    custom_report = custom_report_update.dict()

    assert custom_report is not None
    assert custom_report["id"] is not None
    assert custom_report["organization_id"] is not None
    assert custom_report["name"] == "Test"
    assert custom_report["table"] == "test"


def test_delete_custom_report_schema() -> None:
    """
    Test Custom Report Delete Schema
    """
    custom_reports = [
        CustomReportBase(organization_id=uuid.uuid4(), name="Test", table="test"),
        CustomReportBase(organization_id=uuid.uuid4(), name="Test2", table="test2"),
    ]

    custom_report_store = custom_reports.copy()

    custom_report_store.pop(0)

    assert len(custom_reports) == 2
    assert len(custom_report_store) == 1
    assert custom_reports[0].name == "Test"
    assert custom_report_store[0].name == "Test2"


def test_custom_report_str_representation(custom_report) -> None:
    """
    Test Custom Report String Representation
    """
    assert str(custom_report) == custom_report.name


def test_custom_report_clean_method_with_valid_data(custom_report) -> None:
    """
    Test Custom Report Clean Method with valid data.
    """

    try:
        custom_report.clean()
    except ValidationError:
        pytest.fail("clean method raised ValidationError unexpectedly")


def test_list_custom_report(custom_report) -> None:
    """
    Test Custom Report List
    """
    assert custom_report is not None
