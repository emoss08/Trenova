--keel:notx
--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- Enum values are added outside a transaction because PostgreSQL refuses to
-- use a value added in the same transaction that added it.
ALTER TYPE journal_source_event_enum ADD VALUE IF NOT EXISTS 'FactoringFunded';

--keel:split
ALTER TYPE journal_source_event_enum ADD VALUE IF NOT EXISTS 'FactoringReserveReleased';

--keel:split
ALTER TYPE journal_source_event_enum ADD VALUE IF NOT EXISTS 'FactoringChargedBack';

--keel:split
ALTER TYPE journal_source_event_enum ADD VALUE IF NOT EXISTS 'FactoringChargebackReversed';
