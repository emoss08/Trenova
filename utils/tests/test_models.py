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

from django.core.checks import Error
from django.db.models import TextChoices

from utils import models

def test_choice_field_with_no_choice_attr():
    """
    Test Error is thrown when Choice has no choices attribute
    """
    model = models.ChoiceField(
        name="test",
        max_length=10,
    )
    errors = model.check()
    assert len(errors) == 1
    for error in errors:
        assert isinstance(error, Error)
        assert error.msg == "ChoiceField must define a `choice` attribute."

def test_choice_field_with_no_max_length():
    """
    Test Error is thrown when Choice has no max_length attribute
    """
    model = models.ChoiceField(
        name="test",
        choices=[],
    )
    errors = model.check()
    assert len(errors) == 1
    for error in errors:
        assert isinstance(error, Error)
        assert error.id == "fields.E120"
        assert error.msg == "CharFields must define a 'max_length' attribute."

def test_choice_model_created_with_max_length():
    """
    Test Choice model is created
    """
    class TestChoices(TextChoices):
        """
        Status choices for Order model
        """

        PREPAID = "TEST_CHOICES", "Test Choices"
        OTHER = "OTHER", "Other"

    model = models.ChoiceField(
        name="test",
        choices=TestChoices.choices,
    )
    assert model.name == "test"
    assert model.choices == TestChoices.choices
    assert model.max_length == max(len(choice[0]) for choice in TestChoices.choices)