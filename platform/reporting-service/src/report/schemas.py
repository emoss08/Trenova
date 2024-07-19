# COPYRIGHT(c) 2024 Trenova
#
# This file is part of Trenova.
#
# The Trenova software is licensed under the Business Source License 1.1. You are granted the right
# to copy, modify, and redistribute the software, but only for non-production use or with a total
# of less than three server instances. Starting from the Change Date (November 16, 2026), the
# software will be made available under version 2 or later of the GNU General Public License.
# If you use the software in violation of this license, your rights under the license will be
# terminated automatically. The software is provided "as is," and the Licensor disclaims all
# warranties and conditions. If you use this license's text or the "Business Source License" name
# and trademark, you must comply with the Licensor's covenants, which include specifying the
# Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
# Grant, and not modifying the license in any other way.


from datetime import datetime
from typing import Optional, TypedDict
from uuid import UUID

from pydantic import BaseModel


class RelationshipType(TypedDict):
    foreignKey: str
    referencedTable: str
    columns: list[str]


class Relationship(BaseModel):
    foreignKey: str
    referencedTable: str
    columns: list[str]

    def to_dict(self) -> RelationshipType:
        return {
            "foreignKey": self.foreignKey,
            "referencedTable": self.referencedTable,
            "columns": self.columns,
        }


class Report(BaseModel):
    tableName: str
    columns: list[str]
    relationships: list[Relationship] | None
    organizationId: str
    businessUnitId: str
    userId: str
    fileFormat: str
    deliveryMethod: str


class TaskSchema(BaseModel):
    id: UUID
    business_unit_id: UUID
    organization_id: UUID
    user_id: UUID
    task_id: UUID
    status: str
    result: Optional[dict]
    payload: Optional[dict]
    created_at: datetime
    updated_at: datetime

    class Config:
        from_attributes = True
