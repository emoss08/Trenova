--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--
DROP TABLE IF EXISTS "shipment_commodities";

DROP INDEX IF EXISTS "idx_shipment_commodities_shipment";

DROP INDEX IF EXISTS "idx_shipment_commodities_commodity";

DROP INDEX IF EXISTS "idx_shipment_commodities_business_unit";

DROP INDEX IF EXISTS "idx_shipment_commodities_organization";

DROP INDEX IF EXISTS "idx_shipment_commodities_created_updated";

