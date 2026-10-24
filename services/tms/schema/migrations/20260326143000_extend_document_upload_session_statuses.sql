ALTER TYPE "document_upload_session_status_enum" ADD VALUE IF NOT EXISTS 'Uploaded';
ALTER TYPE "document_upload_session_status_enum" ADD VALUE IF NOT EXISTS 'Verifying';
ALTER TYPE "document_upload_session_status_enum" ADD VALUE IF NOT EXISTS 'Finalizing';
ALTER TYPE "document_upload_session_status_enum" ADD VALUE IF NOT EXISTS 'Available';
ALTER TYPE "document_upload_session_status_enum" ADD VALUE IF NOT EXISTS 'Quarantined';
