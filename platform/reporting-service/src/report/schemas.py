# Copyright (c) 2024 Trenova Technologies, LLC
#
# Licensed under the Business Source License 1.1 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://trenova.app/pricing/
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#
# Key Terms:
# - Non-production use only
# - Change Date: 2026-11-16
# - Change License: GNU General Public License v2 or later
#
# For full license text, see the LICENSE file in the root directory.

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
