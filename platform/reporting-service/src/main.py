from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from database import database_available
from routers import report_router

origins = [
    "https://localhost:3000",
    "http://localhost:3000",
]

app = FastAPI()

app.add_middleware(
    CORSMiddleware,
    allow_origins=origins,
    allow_credentials=True,
    allow_methods=["POST"],
    allow_headers=["*"],
)

# Ping the database
if not database_available():
    raise Exception("Database is not available")

app.include_router(report_router.router, prefix="/report")
