--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- resource_permissions has carried nothing but its primary key since RBAC v3, which
-- indexed the user_role_assignments foreign keys and missed this one. Every permission
-- computation loads a role's grants by role_id, so each one sequentially scanned the
-- whole table; the table is sized by roles x resources across every organization, so one
-- tenant adding roles slows the check down for all of them. The INCLUDE columns are the
-- rest of the has-many load, which lets it stay off the heap. Updates to a role's grants
-- lose HOT eligibility in exchange, and those happen only when somebody edits a role.
-- Concurrent builds cannot run inside the migration transaction, hence the non-tx file.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_resource_permissions_role" ON "resource_permissions"("role_id") INCLUDE ("resource", "operations", "data_scope");
