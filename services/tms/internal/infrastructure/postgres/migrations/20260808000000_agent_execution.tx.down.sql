--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md
--
DROP TABLE IF EXISTS "agent_decisions";

--bun:split
DROP TABLE IF EXISTS "agent_exceptions";

--bun:split
DROP TABLE IF EXISTS "agent_proposals";

--bun:split
DROP TABLE IF EXISTS "agent_runs";

--bun:split
DROP TABLE IF EXISTS "agent_controls";

--bun:split
DROP TYPE IF EXISTS "agent_decision_type_enum";

DROP TYPE IF EXISTS "agent_resolution_state_enum";

DROP TYPE IF EXISTS "agent_severity_enum";

DROP TYPE IF EXISTS "agent_exception_category_enum";

DROP TYPE IF EXISTS "agent_autonomy_tier_enum";

DROP TYPE IF EXISTS "agent_proposal_status_enum";

DROP TYPE IF EXISTS "agent_run_status_enum";

DROP TYPE IF EXISTS "agent_subject_type_enum";

DROP TYPE IF EXISTS "agent_type_enum";
