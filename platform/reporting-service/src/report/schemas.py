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
