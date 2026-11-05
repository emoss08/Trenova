-- An upload to a conversation is private to whoever is uploading it, like the document it becomes.
ALTER TABLE "document_upload_sessions"
    ADD COLUMN IF NOT EXISTS "uploaded_by_id" varchar(100);
