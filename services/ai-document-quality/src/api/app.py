# Copyright 2023-2025 Eric Moss
# Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
# Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md

"""FastAPI application for document quality assessment service"""

import logging
import os
import uuid
from datetime import datetime
from io import BytesIO
from pathlib import Path
from typing import List, Optional

from fastapi import FastAPI, File, HTTPException, UploadFile, status
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse
from PIL import Image

from .inference import DocumentQualityPredictor
from .document_classification import initialize_classification_service, get_classification_service
from .models import (
    BatchAnalysisResponse,
    DocumentAnalysisRequest,
    DocumentAnalysisResponse,
    ErrorResponse,
    HealthResponse,
    IssueDetection,
    MetricsResponse,
    QualityAssessment,
)

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
)
logger = logging.getLogger(__name__)

# Create FastAPI app
app = FastAPI(
    title="Document Quality Assessment API",
    description="AI-powered document quality assessment for transportation management systems",
    version="2.0.0",
    docs_url="/docs",
    redoc_url="/redoc",
)

# CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=os.getenv("CORS_ORIGINS", "*").split(","),
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Global predictor instance (loaded on startup)
predictor: Optional[DocumentQualityPredictor] = None


@app.on_event("startup")
async def startup_event():
    """Initialize predictor and classification service on startup"""
    global predictor

    model_path = os.getenv("MODEL_PATH", "models/best_model.pth")
    config_path = os.getenv("CONFIG_PATH", "config/best_training.yaml")
    device = os.getenv("DEVICE", "auto")
    classifier_model_path = os.getenv("CLASSIFIER_MODEL_PATH", "models/document_classifier.pth")
    template_bank_path = os.getenv("TEMPLATE_BANK_PATH", "models/customer_templates.pth")

    logger.info("=" * 80)
    logger.info("STARTING DOCUMENT QUALITY ASSESSMENT API")
    logger.info("=" * 80)
    logger.info(f"Model path: {model_path}")
    logger.info(f"Config path: {config_path}")
    logger.info(f"Classifier model path: {classifier_model_path}")
    logger.info(f"Template bank path: {template_bank_path}")
    logger.info(f"Device: {device}")

    try:
        # Initialize quality predictor
        predictor = DocumentQualityPredictor(
            model_path=model_path,
            config_path=config_path if Path(config_path).exists() else None,
            device=device,
            enable_performance_monitoring=True,
        )
        logger.info("✓ Quality predictor initialized successfully")

        # Initialize classification service
        initialize_classification_service(
            model_path=classifier_model_path if Path(classifier_model_path).exists() else None,
            template_bank_path=template_bank_path if Path(template_bank_path).exists() else None,
            device=device
        )
        logger.info("✓ Document classification service initialized successfully")

        logger.info("=" * 80)
    except Exception as e:
        logger.error(f"Failed to initialize services: {str(e)}")
        raise


@app.get("/", tags=["General"])
async def root():
    """Root endpoint"""
    return {
        "service": "Document Quality Assessment API",
        "version": "2.0.0",
        "status": "running",
        "docs": "/docs",
    }


@app.get("/health", response_model=HealthResponse, tags=["Monitoring"])
async def health_check():
    """
    Health check endpoint

    Returns service status, model status, and basic metrics
    """
    if predictor is None:
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="Predictor not initialized",
        )

    return HealthResponse(
        status="healthy",
        version="2.0.0",
        model_loaded=True,
        device=str(predictor.device),
        uptime_seconds=predictor.get_uptime(),
        requests_processed=predictor.request_count,
    )


@app.get("/metrics", response_model=MetricsResponse, tags=["Monitoring"])
async def get_metrics():
    """
    Get detailed performance metrics

    Returns processing times, acceptance rates, and error counts
    """
    if predictor is None:
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="Predictor not initialized",
        )

    metrics = predictor.get_metrics()
    return MetricsResponse(**metrics)


@app.post("/metrics/reset", tags=["Monitoring"])
async def reset_metrics():
    """Reset performance metrics (admin endpoint)"""
    if predictor is None:
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="Predictor not initialized",
        )

    predictor.reset_metrics()
    return {"message": "Metrics reset successfully"}


@app.post("/analyze", response_model=DocumentAnalysisResponse, tags=["Analysis"])
async def analyze_document(
    file: UploadFile = File(..., description="Document image file"),
    threshold: float = 0.5,
    include_issues: bool = True,
):
    """
    Analyze a single document image

    Upload a document image and receive quality assessment results including:
    - Overall quality score (0-1)
    - Quality classification (High, Good, Moderate, Poor, Very Poor)
    - Accept/reject decision based on threshold
    - Detected quality issues
    - Recommendations for improvement

    Args:
        file: Image file (JPEG, PNG, etc.)
        threshold: Quality threshold for accept/reject decision (0-1)
        include_issues: Whether to include detailed issue detection

    Returns:
        DocumentAnalysisResponse with quality assessment results
    """
    if predictor is None:
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="Predictor not initialized",
        )

    request_id = f"req_{uuid.uuid4().hex[:12]}"
    timestamp = datetime.utcnow()

    try:
        # Validate file type
        if not file.content_type.startswith("image/"):
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail=f"Invalid file type: {file.content_type}. Expected image file.",
            )

        # Read image
        contents = await file.read()
        image = Image.open(BytesIO(contents))

        # Predict
        result = predictor.predict(
            image=image, threshold=threshold, include_issues=include_issues
        )

        # Build response
        quality = QualityAssessment(
            score=result["quality_score"],
            quality_class=result["quality_class"],
            quality_class_index=result["quality_class_index"],
            is_acceptable=result["is_acceptable"],
            confidence=result["confidence"],
        )

        issues = []
        if include_issues and "issues" in result:
            issues = [IssueDetection(**issue) for issue in result["issues"]]

        recommendations = result.get("recommendations", [])

        return DocumentAnalysisResponse(
            request_id=request_id,
            timestamp=timestamp,
            quality=quality,
            issues=issues,
            recommendations=recommendations,
            processing_time_ms=result["processing_time_ms"],
            visualization_url=None,
        )

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error analyzing document: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error processing document: {str(e)}",
        )


@app.post("/analyze/batch", response_model=BatchAnalysisResponse, tags=["Analysis"])
async def analyze_batch(
    files: List[UploadFile] = File(..., description="Multiple document image files"),
    threshold: float = 0.5,
    include_issues: bool = True,
):
    """
    Analyze multiple document images in batch

    Upload multiple document images and receive quality assessment for each.
    Batch processing is more efficient than individual requests.

    Args:
        files: List of image files
        threshold: Quality threshold for accept/reject decision
        include_issues: Whether to include detailed issue detection

    Returns:
        BatchAnalysisResponse with results for all documents and summary statistics
    """
    if predictor is None:
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="Predictor not initialized",
        )

    if len(files) == 0:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="No files provided",
        )

    if len(files) > 100:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Maximum 100 files per batch request",
        )

    batch_id = f"batch_{uuid.uuid4().hex[:12]}"
    timestamp = datetime.utcnow()
    batch_start_time = datetime.now()

    results = []
    images = []
    file_names = []

    try:
        # Load all images
        for file in files:
            if not file.content_type.startswith("image/"):
                logger.warning(f"Skipping non-image file: {file.filename}")
                continue

            contents = await file.read()
            image = Image.open(BytesIO(contents))
            images.append(image)
            file_names.append(file.filename)

        if not images:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="No valid image files found",
            )

        # Batch prediction
        predictions = predictor.predict_batch(
            images=images, threshold=threshold, include_issues=include_issues
        )

        # Build individual responses
        for idx, (prediction, filename) in enumerate(zip(predictions, file_names)):
            request_id = f"{batch_id}_{idx}"

            quality = QualityAssessment(
                score=prediction["quality_score"],
                quality_class=prediction["quality_class"],
                quality_class_index=prediction["quality_class_index"],
                is_acceptable=prediction["is_acceptable"],
                confidence=prediction["confidence"],
            )

            issues = []
            if include_issues and "issues" in prediction:
                issues = [IssueDetection(**issue) for issue in prediction["issues"]]

            recommendations = prediction.get("recommendations", [])

            result = DocumentAnalysisResponse(
                request_id=request_id,
                timestamp=timestamp,
                quality=quality,
                issues=issues,
                recommendations=recommendations,
                processing_time_ms=0.0,  # Individual times not tracked in batch
                visualization_url=None,
            )
            results.append(result)

        # Calculate summary statistics
        total_acceptable = sum(1 for r in results if r.quality.is_acceptable)
        total_rejected = len(results) - total_acceptable
        avg_quality_score = sum(r.quality.score for r in results) / len(results)

        summary = {
            "acceptable": total_acceptable,
            "rejected": total_rejected,
            "average_quality_score": round(avg_quality_score, 3),
            "quality_distribution": {},
        }

        # Quality class distribution
        for class_name in predictor.CLASS_NAMES:
            count = sum(1 for r in results if r.quality.quality_class == class_name)
            summary["quality_distribution"][class_name] = count

        # Calculate total processing time
        batch_end_time = datetime.now()
        total_processing_time = (batch_end_time - batch_start_time).total_seconds() * 1000

        return BatchAnalysisResponse(
            request_id=batch_id,
            timestamp=timestamp,
            total_documents=len(results),
            results=results,
            summary=summary,
            total_processing_time_ms=total_processing_time,
        )

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error in batch analysis: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error processing batch: {str(e)}",
        )


@app.exception_handler(HTTPException)
async def http_exception_handler(request, exc):
    """Custom HTTP exception handler"""
    return JSONResponse(
        status_code=exc.status_code,
        content=ErrorResponse(
            error=exc.detail.split(":")[0] if ":" in exc.detail else "HTTPError",
            message=exc.detail,
            request_id=None,
            timestamp=datetime.utcnow(),
        ).model_dump(),
    )


@app.exception_handler(Exception)
async def general_exception_handler(request, exc):
    """General exception handler"""
    logger.error(f"Unhandled exception: {str(exc)}")
    return JSONResponse(
        status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
        content=ErrorResponse(
            error="InternalServerError",
            message=str(exc),
            request_id=None,
            timestamp=datetime.utcnow(),
        ).model_dump(),
    )


# ============================================================================
# Document Classification Endpoints
# ============================================================================

@app.post("/classify", tags=["Document Classification"], response_model=dict)
async def classify_document(
    file: UploadFile = File(...),
    customer_id: Optional[str] = None,
    top_k: int = 3,
    confidence_threshold: float = 0.6
):
    """
    Classify a document image to determine its type.

    - **file**: Document image file
    - **customer_id**: Optional customer ID for template matching
    - **top_k**: Number of top predictions to return
    - **confidence_threshold**: Minimum confidence threshold

    Returns document type predictions with confidence scores.
    """
    try:
        service = get_classification_service()

        # Read and validate image
        contents = await file.read()
        try:
            image = Image.open(BytesIO(contents))
        except Exception as e:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail=f"Invalid image file: {str(e)}"
            )

        # Classify document
        result = service.classify_document(
            image=image,
            customer_id=customer_id,
            top_k=top_k,
            confidence_threshold=confidence_threshold
        )

        return result

    except Exception as e:
        logger.error(f"Error classifying document: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=str(e)
        )


@app.post("/classify/batch", tags=["Document Classification"], response_model=dict)
async def classify_documents_batch(
    files: List[UploadFile] = File(...),
    customer_ids: Optional[str] = None,
    top_k: int = 3,
    confidence_threshold: float = 0.6
):
    """
    Classify multiple document images in batch.

    - **files**: List of document image files
    - **customer_ids**: Optional comma-separated customer IDs (one per file)
    - **top_k**: Number of top predictions per document
    - **confidence_threshold**: Minimum confidence threshold

    Returns classification results for all documents.
    """
    try:
        service = get_classification_service()

        # Parse customer IDs if provided
        customer_id_list = None
        if customer_ids:
            customer_id_list = customer_ids.split(",")
            if len(customer_id_list) != len(files):
                raise HTTPException(
                    status_code=status.HTTP_400_BAD_REQUEST,
                    detail="Number of customer_ids must match number of files"
                )

        # Read all images
        images = []
        for file in files:
            contents = await file.read()
            try:
                image = Image.open(BytesIO(contents))
                images.append(image)
            except Exception as e:
                raise HTTPException(
                    status_code=status.HTTP_400_BAD_REQUEST,
                    detail=f"Invalid image file {file.filename}: {str(e)}"
                )

        # Classify batch
        results = service.classify_batch(
            images=images,
            customer_ids=customer_id_list,
            top_k=top_k,
            confidence_threshold=confidence_threshold
        )

        return {
            "results": results,
            "total_processed": len(results)
        }

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error classifying batch: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=str(e)
        )


@app.post("/templates/learn", tags=["Customer Templates"], response_model=dict)
async def learn_customer_template(
    file: UploadFile = File(...),
    customer_id: str = None,
    document_type: str = None,
    template_id: Optional[str] = None
):
    """
    Learn a new customer template from a document image.

    - **file**: Document image file
    - **customer_id**: Customer identifier (required)
    - **document_type**: Base document type (BOL, INVOICE, etc.) (required)
    - **template_id**: Optional template identifier

    Adds this document to the customer's template bank for future matching.
    """
    if not customer_id:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="customer_id is required"
        )

    if not document_type:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="document_type is required"
        )

    try:
        service = get_classification_service()

        # Read and validate image
        contents = await file.read()
        try:
            image = Image.open(BytesIO(contents))
        except Exception as e:
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail=f"Invalid image file: {str(e)}"
            )

        # Learn template
        result = service.learn_customer_template(
            image=image,
            customer_id=customer_id,
            document_type=document_type.upper(),
            template_id=template_id,
            metadata={
                "filename": file.filename,
                "learned_at": datetime.utcnow().isoformat()
            }
        )

        return result

    except ValueError as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=str(e)
        )
    except Exception as e:
        logger.error(f"Error learning template: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=str(e)
        )


@app.get("/templates/customer/{customer_id}", tags=["Customer Templates"], response_model=dict)
async def get_customer_templates(customer_id: str):
    """
    Get information about a customer's document templates.

    - **customer_id**: Customer identifier

    Returns template counts and document types for the customer.
    """
    try:
        service = get_classification_service()
        return service.get_customer_templates(customer_id)
    except Exception as e:
        logger.error(f"Error getting customer templates: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=str(e)
        )


@app.get("/templates/customers", tags=["Customer Templates"], response_model=dict)
async def get_all_customers():
    """
    Get information about all customers with templates.

    Returns list of all customers and their template information.
    """
    try:
        service = get_classification_service()
        customers = service.get_all_customers()
        return {
            "customers": customers,
            "total_customers": len(customers)
        }
    except Exception as e:
        logger.error(f"Error getting customers: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=str(e)
        )


@app.get("/classify/document-types", tags=["Document Classification"], response_model=dict)
async def get_supported_document_types():
    """
    Get list of supported base document types.

    Returns dictionary of document type codes and descriptions.
    """
    try:
        service = get_classification_service()
        return {
            "document_types": service.get_supported_document_types(),
            "total_types": len(service.get_supported_document_types())
        }
    except Exception as e:
        logger.error(f"Error getting document types: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=str(e)
        )


@app.get("/classify/metrics", tags=["Document Classification"], response_model=dict)
async def get_classification_metrics():
    """
    Get document classification service metrics.

    Returns classification counts, template counts, and performance metrics.
    """
    try:
        service = get_classification_service()
        return service.get_metrics()
    except Exception as e:
        logger.error(f"Error getting metrics: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=str(e)
        )


if __name__ == "__main__":
    import uvicorn

    port = int(os.getenv("PORT", "8000"))
    host = os.getenv("HOST", "0.0.0.0")

    uvicorn.run(
        "src.api.app:app",
        host=host,
        port=port,
        reload=os.getenv("RELOAD", "false").lower() == "true",
        log_level="info",
    )
