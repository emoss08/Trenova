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
import factory
from django.db import models


class UpdateGetOrCreateMetaClass(factory.base.FactoryMetaClass):
    def __new__(mcs, name, bases, attrs):
        cls = super().__new__(mcs, name, bases, attrs)
        if hasattr(cls, "Meta"):
            cls.update_get_or_create()
        return cls


class FactoryMixin(
    factory.django.DjangoModelFactory, metaclass=UpdateGetOrCreateMetaClass
):
    class Meta:
        model = models.Model
        django_get_or_create = ()

    business_unit = factory.SubFactory("organization.factories.BusinessUnitFactory")
    organization = factory.SubFactory("organization.factories.OrganizationFactory")

    @classmethod
    def update_get_or_create(cls):
        django_get_or_create = list(cls.Meta.django_get_or_create)
        django_get_or_create.extend(
            attr_name
            for attr_name, attr_value in cls.__dict__.items()
            if isinstance(attr_value, factory.SubFactory)
        )
        cls.Meta.django_get_or_create = tuple(django_get_or_create)


#
#
# class FactoryMixin(
#     factory.django.DjangoModelFactory, metaclass=UpdateGetOrCreateMetaClass
# ):
#     class Meta:
#         model = models.Model
#         django_get_or_create = ()
#
#     business_unit = factory.SubFactory("organization.factories.BusinessUnitFactory")
#     organization = factory.SubFactory("organization.factories.OrganizationFactory")
#
#     @classmethod
#     def update_get_or_create(cls):
#         django_get_or_create = list(cls.Meta.django_get_or_create)
#         django_get_or_create.extend(
#             attr_name
#             for attr_name, attr_value in cls.__dict__.items()
#             if isinstance(attr_value, factory.SubFactory)
#         )
#         cls.Meta.django_get_or_create = tuple(django_get_or_create)
