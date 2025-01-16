-- Trenova - (c) 2024 Eric Moss
-- Licensed under the Business Source License 1.1 (BSL 1.1)
--
-- You may use this software for non-production purposes only.
-- For full license text, see the LICENSE file in the project root.
--
-- This software will be licensed under GPLv2 or later on 2026-11-16.
-- For alternative licensing options, email: eric@trenova.app

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TYPE gender_enum AS ENUM ('Male', 'Female', 'Other');