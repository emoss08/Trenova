--keel:notx
-- Customer status emails find their candidates through one index range per way a shipment can owe
-- an email: open shipments through the open index, recently changed ones through the updated
-- index, and recently delivered ones and shipments with a recently filed POD through these. The
-- delivered index holds only delivered shipments; the documents index holds the current, live
-- versions a resource's recent documents are read from, by kind of resource and filing time.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_shipments_delivered" ON "shipments"("organization_id", "business_unit_id", "actual_delivery_date")
WHERE
    "actual_delivery_date" IS NOT NULL;

--keel:split
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_documents_current_filed" ON "documents"("organization_id", "business_unit_id", "resource_type", "created_at") INCLUDE ("resource_id", "document_type_id")
WHERE
    "is_current_version" IS TRUE AND "status" NOT IN ('Rejected', 'Archived');
