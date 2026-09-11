--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

DROP TABLE IF EXISTS "osha_annual_summaries";
--bun:split
DROP TABLE IF EXISTS "worker_injuries";
--bun:split
DROP TYPE IF EXISTS "osha_summary_status_enum";
--bun:split
DROP TYPE IF EXISTS "workers_comp_claim_status_enum";
--bun:split
DROP TYPE IF EXISTS "injury_case_status_enum";
--bun:split
DROP TYPE IF EXISTS "injury_treatment_enum";
--bun:split
DROP TYPE IF EXISTS "osha_illness_type_enum";
--bun:split
DROP TYPE IF EXISTS "osha_case_classification_enum";
