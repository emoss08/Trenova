from datetime import datetime
from typing import Dict, List, Optional

from pydantic import BaseModel, Field, field_validator


class QualityAssessment(BaseModel):
    score: float = Field(..., ge=0.0, le=1.0, description="Overall quality score (0-1)")
    quality_class: str = Field(..., description="Quality classification")
    quality_class_index: int = Field(
        ..., ge=0, le=4, description="Quality class index (0-4)"
    )
    is_acceptable: bool = Field(
        ..., description="Whether document meets quality threshold"
    )
    confidence: float = Field(
        ..., ge=0.0, le=1.0, description="Model confidence in prediction"
    )

    class Config:
        json_schema_extra = {
            "example": {
                "score": 0.85,
                "quality_class": "Good",
                "quality_class_index": 1,
                "is_acceptable": True,
                "confidence": 0.92,
            }
        }


class IssueDetection(BaseModel):
    issue_type: str = Field(..., description="Type of issue detected")
    probability: float = Field(
        ..., ge=0.0, le=1.0, description="Probability of issue (0-1)"
    )
    severity: str = Field(..., description="Issue severity (minor, moderate, critical)")

    class Config:
        json_schema_extra = {
            "example": {
                "issue_type": "blur",
                "probability": 0.75,
                "severity": "critical",
            }
        }


class DocumentAnalysisRequest(BaseModel):
    threshold: Optional[float] = Field(
        default=0.5,
        ge=0.0,
        le=1.0,
        description="Quality threshold for accept/reject decision",
    )
    return_visualization: bool = Field(
        default=False, description="Whether to return Grad-CAM visualization"
    )
    include_issues: bool = Field(
        default=True, description="Whether to include issue detection"
    )

    @field_validator("threshold")
    @classmethod
    def validate_threshold(cls, v):
        if not 0.0 <= v <= 1.0:
            raise ValueError("Threshold must be between 0 and 1")
        return v

    class Config:
        json_schema_extra = {
            "example": {
                "threshold": 0.5,
                "return_visualization": False,
                "include_issues": True,
            }
        }


class DocumentAnalysisResponse(BaseModel):
    request_id: str = Field(..., description="Unique request identifier")
    timestamp: datetime = Field(..., description="Analysis timestamp")
    quality: QualityAssessment = Field(..., description="Quality assessment results")
    issues: List[IssueDetection] = Field(default=[], description="Detected issues")
    recommendations: List[str] = Field(
        default=[], description="Recommendations for improvement"
    )
    processing_time_ms: float = Field(
        ..., description="Processing time in milliseconds"
    )
    visualization_url: Optional[str] = Field(
        default=None, description="URL to Grad-CAM visualization (if requested)"
    )

    class Config:
        json_schema_extra = {
            "example": {
                "request_id": "req_abc123",
                "timestamp": "2025-09-30T14:30:22Z",
                "quality": {
                    "score": 0.85,
                    "quality_class": "Good",
                    "quality_class_index": 1,
                    "is_acceptable": True,
                    "confidence": 0.92,
                },
                "issues": [
                    {
                        "issue_type": "slight_blur",
                        "probability": 0.35,
                        "severity": "minor",
                    }
                ],
                "recommendations": [
                    "Document quality is acceptable",
                    "Consider better focus for optimal quality",
                ],
                "processing_time_ms": 145.5,
                "visualization_url": None,
            }
        }


class BatchAnalysisRequest(BaseModel):
    threshold: Optional[float] = Field(
        default=0.5, ge=0.0, le=1.0, description="Quality threshold"
    )
    include_issues: bool = Field(default=True, description="Include issue detection")

    class Config:
        json_schema_extra = {
            "example": {
                "threshold": 0.5,
                "include_issues": True,
            }
        }


class BatchAnalysisResponse(BaseModel):
    request_id: str = Field(..., description="Unique batch request identifier")
    timestamp: datetime = Field(..., description="Batch analysis timestamp")
    total_documents: int = Field(..., description="Total documents analyzed")
    results: List[DocumentAnalysisResponse] = Field(
        ..., description="Individual results"
    )
    summary: Dict[str, any] = Field(..., description="Batch summary statistics")
    total_processing_time_ms: float = Field(..., description="Total processing time")

    class Config:
        json_schema_extra = {
            "example": {
                "request_id": "batch_xyz789",
                "timestamp": "2025-09-30T14:30:22Z",
                "total_documents": 10,
                "results": [],
                "summary": {
                    "acceptable": 8,
                    "rejected": 2,
                    "average_quality_score": 0.75,
                },
                "total_processing_time_ms": 1250.0,
            }
        }


class HealthResponse(BaseModel):
    status: str = Field(..., description="Service status")
    version: str = Field(..., description="Service version")
    model_loaded: bool = Field(..., description="Whether model is loaded")
    device: str = Field(..., description="Device being used (cpu/cuda)")
    uptime_seconds: float = Field(..., description="Service uptime in seconds")
    requests_processed: int = Field(..., description="Total requests processed")

    class Config:
        json_schema_extra = {
            "example": {
                "status": "healthy",
                "version": "2.0.0",
                "model_loaded": True,
                "device": "cuda",
                "uptime_seconds": 3600.5,
                "requests_processed": 1234,
            }
        }


class MetricsResponse(BaseModel):
    total_requests: int = Field(..., description="Total requests processed")
    average_processing_time_ms: float = Field(
        ..., description="Average processing time"
    )
    p50_processing_time_ms: float = Field(..., description="P50 processing time")
    p95_processing_time_ms: float = Field(..., description="P95 processing time")
    p99_processing_time_ms: float = Field(..., description="P99 processing time")
    acceptance_rate: float = Field(
        ..., ge=0.0, le=1.0, description="Document acceptance rate"
    )
    rejection_rate: float = Field(
        ..., ge=0.0, le=1.0, description="Document rejection rate"
    )
    average_quality_score: float = Field(..., description="Average quality score")
    errors: int = Field(..., description="Total errors encountered")

    class Config:
        json_schema_extra = {
            "example": {
                "total_requests": 1000,
                "average_processing_time_ms": 150.5,
                "p50_processing_time_ms": 145.0,
                "p95_processing_time_ms": 250.0,
                "p99_processing_time_ms": 400.0,
                "acceptance_rate": 0.85,
                "rejection_rate": 0.15,
                "average_quality_score": 0.73,
                "errors": 5,
            }
        }


class ErrorResponse(BaseModel):
    error: str = Field(..., description="Error type")
    message: str = Field(..., description="Error message")
    request_id: Optional[str] = Field(
        default=None, description="Request ID if available"
    )
    timestamp: datetime = Field(..., description="Error timestamp")

    class Config:
        json_schema_extra = {
            "example": {
                "error": "InvalidImage",
                "message": "Unable to process image file",
                "request_id": "req_abc123",
                "timestamp": "2025-09-30T14:30:22Z",
            }
        }
