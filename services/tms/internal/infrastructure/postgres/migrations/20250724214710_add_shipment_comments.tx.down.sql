--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--
DROP TABLE IF EXISTS "shipment_comments";

DROP INDEX IF EXISTS "idx_shipment_comments_shipment";

DROP INDEX IF EXISTS "idx_shipment_comments_business_unit";

DROP INDEX IF EXISTS "idx_shipment_comments_organization";

DROP INDEX IF EXISTS "idx_shipment_comments_created_updated";

DROP INDEX IF EXISTS "idx_shipment_comments_org_bu_user";

--bun:split
-- Drop shipment comment mentions table and indexes
DROP TABLE IF EXISTS "shipment_comment_mentions";

DROP INDEX IF EXISTS "idx_shipment_comment_mentions_comment";

DROP INDEX IF EXISTS "idx_shipment_comment_mentions_user";

DROP INDEX IF EXISTS "idx_shipment_comment_mentions_organization";

