--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

DROP TABLE IF EXISTS "worker_checklist_items";

--bun:split
DROP TABLE IF EXISTS "worker_checklists";

--bun:split
DROP TABLE IF EXISTS "worker_checklist_template_items";

--bun:split
DROP TABLE IF EXISTS "worker_checklist_templates";

--bun:split
DROP TYPE IF EXISTS "worker_checklist_item_status_enum";

--bun:split
DROP TYPE IF EXISTS "worker_checklist_status_enum";

--bun:split
DROP TYPE IF EXISTS "worker_checklist_owner_enum";

--bun:split
DROP TYPE IF EXISTS "worker_checklist_item_kind_enum";

--bun:split
DROP TYPE IF EXISTS "worker_checklist_trigger_enum";

--bun:split
DROP TYPE IF EXISTS "worker_checklist_kind_enum";
