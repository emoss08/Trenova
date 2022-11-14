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

from rest_framework.routers import DefaultRouter

from equipment.api import (
    EquipmentMaintenancePlanViewSet, EquipmentManufacturerViewSet,
    EquipmentTypeViewSet, EquipmentViewSet,
)

router = DefaultRouter()

# Equipment Urls
router.register(r"equipment", EquipmentViewSet, basename="equipment")
router.register(
    r"equipment-manufacturer",
    EquipmentManufacturerViewSet,
    basename="equipment-manufacturer",
)
router.register(
    r"equipment-type",
    EquipmentTypeViewSet,
    basename="equipment-type",
)
router.register(
    r"equipment-maintenance-plan",
    EquipmentMaintenancePlanViewSet,
    basename="equipment-maintenance-plan",
)

urlpatterns = router.urls
