--keel:notx
-- A person who loses sign-in loses their Capture devices in every organization, so the lookup is by
-- person across tenants, which idx_capture_devices_user cannot answer because it leads with the
-- tenant columns.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_capture_devices_user_status"
    ON "capture_devices"("user_id", "status");
