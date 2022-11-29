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

import factory

from dispatch import models


class DispatchControlFactory(factory.django.DjangoModelFactory):
    """
    Dispatch control factory
    """

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    regulatory_check = True

    class Meta:
        """
        Metaclass for DispatchControlFactory
        """

        model = models.DispatchControl


class DelayCodeFactory(factory.django.DjangoModelFactory):
    """
    Delay code factory
    """

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    code = "TEST"

    class Meta:
        """
        Metaclass for DelayCodeFactory
        """

        model = models.DelayCode


class FleetCodeFactory(factory.django.DjangoModelFactory):
    """
    Fleet code factory
    """

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    code = "TEST"

    class Meta:
        """
        Metaclass for FleetCodeFactory
        """

        model = models.FleetCode


class CommentTypeFactory(factory.django.DjangoModelFactory):
    """
    Comment type factory
    """

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    name = factory.Faker("word", locale="en_US")

    class Meta:
        """
        Metaclass for CommentTypeFactory
        """

        model = models.CommentType
