--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

DROP TABLE IF EXISTS "worker_credentials";

--bun:split
DROP TABLE IF EXISTS "worker_credential_types";

--bun:split
DROP TYPE IF EXISTS "worker_credential_status_enum";

--bun:split
DROP TYPE IF EXISTS "worker_credential_category_enum";
