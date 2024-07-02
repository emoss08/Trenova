import datetime
import logging
import os
from http import server
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
