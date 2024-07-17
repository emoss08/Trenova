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

import logging
import os
from uuid import uuid4

from sqlalchemy import JSON, Column, DateTime, String, create_engine
from sqlalchemy.dialects.postgresql import UUID
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy.orm import sessionmaker
from sqlalchemy.sql import func, text

SQLALCHEMY_DATABASE_URL = os.environ.get(
    "DATABASE_URL", "postgresql://postgres:postgres@localhost:5432/trenova_go_db"
)

engine = create_engine(SQLALCHEMY_DATABASE_URL)
SessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)


# Setup logging
file_handler = logging.FileHandler(filename="sql.log")
handlers = [file_handler]

logging.basicConfig(
    format="[%(asctime)s] {%(filename)s:%(lineno)d} %(levelname)s - %(message)s",
    handlers=handlers,  # type:ignore
)

logger = logging.getLogger("sqlalchemy.engine")
logger.setLevel(logging.INFO)

Base = declarative_base()


def database_available():
    statement = text("SELECT 1")
    try:
        with engine.connect() as conn:
            conn.execute(statement=statement)
    except Exception as e:
        print(f"An error occurred: {e}")
        return False
    return True


# Define the Task model
class TaskStatus(Base):
    __tablename__ = "user_tasks"

    id = Column(UUID(as_uuid=True), primary_key=True, default=uuid4)
    business_unit_id = Column(UUID(as_uuid=True), nullable=False)
    organization_id = Column(UUID(as_uuid=True), nullable=False)
    user_id = Column(UUID(as_uuid=True), nullable=False)
    task_id = Column(UUID(as_uuid=True), nullable=False, unique=True)
    status = Column(String, nullable=False)
    result = Column(String)
    payload = Column(JSON)
    error = Column(JSON)
    created_at = Column(DateTime(timezone=True), server_default=func.now())
    updated_at = Column(
        DateTime(timezone=True), server_default=func.now(), onupdate=func.now()
    )
