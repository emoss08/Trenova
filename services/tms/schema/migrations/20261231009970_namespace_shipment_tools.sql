-- Shipment, order and import-page tools now carry their namespace (shipment_, order_, intake_), and search_shipments and list_shipments are one tool, shipment_list.
-- Renames an agent tool everywhere a tenant's saved agent configuration names it: the
-- definitions' tool lists, disabled lists, tiers and daily caps, earned trust, tool rule
-- overrides and memories about one tool. A captured eval case replays the tools it held
-- when it was captured, so one that held the old name is quarantined rather than
-- rewritten.
--
-- Two tools merged into one keep a single entry, and the merged entry is never looser
-- than either name was:
--   * tier: the lower of the two (Propose < ActWithApproval < AutoExecute). A name the
--     agent holds with no tier of its own runs at its declared default, which is not
--     known here, so a merge with such a name is held at Propose;
--   * daily cap: the smaller cap, where a cap of 0 or none is no cap;
--   * rule override: the lower max_tier and the stricter reads_external
--     (never < marked < always), where null leaves the declared rule in place;
--   * earned trust: the row with the lower earned tier (none below Propose), then the
--     shorter streak, so a merge never carries the looser name's earned autonomy.
CREATE OR REPLACE FUNCTION "agent_rename_tool"("old_name" text, "new_name" text)
RETURNS void
LANGUAGE plpgsql
AS $fn$
DECLARE
    "tiers" CONSTANT text[] := ARRAY['Propose', 'ActWithApproval', 'AutoExecute'];
    "reads" CONSTANT text[] := ARRAY['never', 'marked', 'always'];
BEGIN
    UPDATE "agent_definitions"
    SET "tool_tiers" = ("tool_tiers" - "old_name") || jsonb_build_object(
        "new_name",
        CASE
            WHEN "tool_tiers" ? "old_name" AND "tool_tiers" ? "new_name" THEN
                CASE
                    WHEN coalesce(array_position("tiers", "tool_tiers" ->> "old_name"), 99)
                        < coalesce(array_position("tiers", "tool_tiers" ->> "new_name"), 99)
                    THEN "tool_tiers" -> "old_name"
                    ELSE "tool_tiers" -> "new_name"
                END
            WHEN "tool_tiers" ? "old_name"
                AND "new_name" = ANY(coalesce("tool_names", '{}'::text[]))
            THEN to_jsonb('Propose'::text)
            WHEN "tool_tiers" ? "old_name" THEN "tool_tiers" -> "old_name"
            ELSE to_jsonb('Propose'::text)
        END
    )
    WHERE "tool_tiers" ? "old_name"
       OR ("tool_tiers" ? "new_name" AND "old_name" = ANY(coalesce("tool_names", '{}'::text[])));

    UPDATE "agent_definitions"
    SET "tool_daily_limits" = ("tool_daily_limits" - "old_name") || jsonb_build_object(
        "new_name",
        CASE
            WHEN NOT ("tool_daily_limits" ? "new_name")
                OR coalesce(("tool_daily_limits" ->> "new_name")::numeric, 0) <= 0
            THEN "tool_daily_limits" -> "old_name"
            WHEN coalesce(("tool_daily_limits" ->> "old_name")::numeric, 0) <= 0
            THEN "tool_daily_limits" -> "new_name"
            ELSE to_jsonb(least(
                ("tool_daily_limits" ->> "old_name")::numeric,
                ("tool_daily_limits" ->> "new_name")::numeric
            ))
        END
    )
    WHERE "tool_daily_limits" ? "old_name";

    UPDATE "agent_definitions"
    SET "tool_names" = ARRAY(
        SELECT "name"
        FROM unnest(array_replace("tool_names", "old_name", "new_name"))
            WITH ORDINALITY AS "entry"("name", "position")
        GROUP BY "name"
        ORDER BY min("position")
    )
    WHERE "old_name" = ANY("tool_names");

    UPDATE "agent_definitions"
    SET "disabled_tool_names" = ARRAY(
        SELECT "name"
        FROM unnest(array_replace("disabled_tool_names", "old_name", "new_name"))
            WITH ORDINALITY AS "entry"("name", "position")
        GROUP BY "name"
        ORDER BY min("position")
    )
    WHERE "old_name" = ANY("disabled_tool_names");

    DELETE FROM "agent_tool_trust" AS "looser"
    USING "agent_tool_trust" AS "twin"
    WHERE "twin"."organization_id" = "looser"."organization_id"
      AND "twin"."business_unit_id" = "looser"."business_unit_id"
      AND "twin"."agent_definition_id" = "looser"."agent_definition_id"
      AND (
          ("looser"."tool_name" = "old_name" AND "twin"."tool_name" = "new_name")
          OR ("looser"."tool_name" = "new_name" AND "twin"."tool_name" = "old_name")
      )
      AND (
          coalesce(array_position("tiers", "looser"."earned_tier"::text), 0),
          "looser"."streak",
          ("looser"."tool_name" = "old_name")::int
      ) > (
          coalesce(array_position("tiers", "twin"."earned_tier"::text), 0),
          "twin"."streak",
          ("twin"."tool_name" = "old_name")::int
      );

    UPDATE "agent_tool_trust" SET "tool_name" = "new_name" WHERE "tool_name" = "old_name";

    UPDATE "agent_tool_rule_overrides" AS "kept"
    SET "max_tier" = CASE
            WHEN "stale"."max_tier" IS NULL THEN "kept"."max_tier"
            WHEN "kept"."max_tier" IS NULL THEN "stale"."max_tier"
            WHEN coalesce(array_position("tiers", "stale"."max_tier"::text), 99)
                < coalesce(array_position("tiers", "kept"."max_tier"::text), 99)
            THEN "stale"."max_tier"
            ELSE "kept"."max_tier"
        END,
        "reads_external" = CASE
            WHEN "stale"."reads_external" IS NULL THEN "kept"."reads_external"
            WHEN "kept"."reads_external" IS NULL THEN "stale"."reads_external"
            WHEN coalesce(array_position("reads", "stale"."reads_external"::text), 0)
                > coalesce(array_position("reads", "kept"."reads_external"::text), 0)
            THEN "stale"."reads_external"
            ELSE "kept"."reads_external"
        END,
        "reason" = CASE WHEN "kept"."reason" = '' THEN "stale"."reason" ELSE "kept"."reason" END,
        "version" = "kept"."version" + 1,
        "updated_at" = extract(epoch FROM current_timestamp)::bigint
    FROM "agent_tool_rule_overrides" AS "stale"
    WHERE "kept"."tool_name" = "new_name"
      AND "stale"."tool_name" = "old_name"
      AND "stale"."organization_id" = "kept"."organization_id"
      AND "stale"."business_unit_id" = "kept"."business_unit_id";

    DELETE FROM "agent_tool_rule_overrides" AS "stale"
    WHERE "stale"."tool_name" = "old_name"
      AND EXISTS (
          SELECT 1 FROM "agent_tool_rule_overrides" AS "kept"
          WHERE "kept"."organization_id" = "stale"."organization_id"
            AND "kept"."business_unit_id" = "stale"."business_unit_id"
            AND "kept"."tool_name" = "new_name"
      );

    UPDATE "agent_tool_rule_overrides" SET "tool_name" = "new_name"
    WHERE "tool_name" = "old_name";

    UPDATE "agent_memories" SET "tool_name" = "new_name" WHERE "tool_name" = "old_name";

    UPDATE "agent_eval_cases" SET "status" = 'Quarantined'
    WHERE "old_name" = ANY("held_tools") AND "status" <> 'Retired';
END;
$fn$;

SELECT "agent_rename_tool"('accept_all_confident', 'intake_accept_confident');
SELECT "agent_rename_tool"('accept_field', 'intake_accept_field');
SELECT "agent_rename_tool"('add_order_charge', 'order_add_charge');
SELECT "agent_rename_tool"('add_shipment_charge', 'shipment_add_charge');
SELECT "agent_rename_tool"('add_shipment_comment', 'shipment_add_comment');
SELECT "agent_rename_tool"('attach_document_to_shipment', 'shipment_attach_document');
SELECT "agent_rename_tool"('attach_order_shipments', 'order_attach_shipments');
SELECT "agent_rename_tool"('cancel_order', 'order_cancel');
SELECT "agent_rename_tool"('cancel_shipment', 'shipment_cancel');
SELECT "agent_rename_tool"('close_order', 'order_close');
SELECT "agent_rename_tool"('create_order', 'order_create');
SELECT "agent_rename_tool"('create_recurring_shipment', 'shipment_create_recurring');
SELECT "agent_rename_tool"('create_shipment', 'shipment_create');
SELECT "agent_rename_tool"('delete_shipment_comment', 'shipment_delete_comment');
SELECT "agent_rename_tool"('detach_order_shipment', 'order_detach_shipment');
SELECT "agent_rename_tool"('duplicate_shipment', 'shipment_duplicate');
SELECT "agent_rename_tool"('edit_shipment_comment', 'shipment_update_comment');
SELECT "agent_rename_tool"('find_shipment_records', 'shipment_match_records');
SELECT "agent_rename_tool"('generate_recurring_shipment', 'shipment_generate_recurring');
SELECT "agent_rename_tool"('get_order', 'order_get');
SELECT "agent_rename_tool"('get_shipment', 'shipment_get');
SELECT "agent_rename_tool"('get_shipment_draft', 'intake_get_draft');
SELECT "agent_rename_tool"('get_shipment_tracking', 'shipment_get_tracking');
SELECT "agent_rename_tool"('list_orders', 'order_list');
SELECT "agent_rename_tool"('list_recurring_shipments', 'shipment_list_recurring');
SELECT "agent_rename_tool"('list_shipment_permits', 'shipment_list_permits');
SELECT "agent_rename_tool"('list_shipments', 'shipment_list');
SELECT "agent_rename_tool"('pin_shipment_comment', 'shipment_pin_comment');
SELECT "agent_rename_tool"('place_shipment_hold', 'shipment_place_hold');
SELECT "agent_rename_tool"('recalculate_shipment_distance', 'shipment_recalculate_distance');
SELECT "agent_rename_tool"('record_shipment_permit', 'shipment_record_permit');
SELECT "agent_rename_tool"('release_shipment_hold', 'shipment_release_hold');
SELECT "agent_rename_tool"('remove_order_charge', 'order_remove_charge');
SELECT "agent_rename_tool"('request_missing_docs', 'shipment_request_documents');
SELECT "agent_rename_tool"('rerate_shipment', 'shipment_rerate');
SELECT "agent_rename_tool"('reschedule_stop', 'shipment_reschedule_stop');
SELECT "agent_rename_tool"('resolve_shipment_comment', 'shipment_resolve_comment');
SELECT "agent_rename_tool"('revoke_customer_tracking_link', 'shipment_revoke_tracking_link');
SELECT "agent_rename_tool"('search_shipments', 'shipment_list');
SELECT "agent_rename_tool"('set_field_value', 'intake_set_field');
SELECT "agent_rename_tool"('set_order_charge_allocations', 'order_set_charge_allocations');
SELECT "agent_rename_tool"('set_recurring_shipment_status', 'shipment_set_recurring_status');
SELECT "agent_rename_tool"('set_required_field', 'intake_set_required_field');
SELECT "agent_rename_tool"('set_stop_location', 'intake_set_stop_location');
SELECT "agent_rename_tool"('set_stop_schedule', 'intake_set_stop_schedule');
SELECT "agent_rename_tool"('share_customer_tracking_link', 'shipment_share_tracking_link');
SELECT "agent_rename_tool"('transfer_shipment_ownership', 'shipment_transfer_ownership');
SELECT "agent_rename_tool"('uncancel_shipment', 'shipment_uncancel');
SELECT "agent_rename_tool"('unpin_shipment_comment', 'shipment_unpin_comment');
SELECT "agent_rename_tool"('update_order', 'order_update');
SELECT "agent_rename_tool"('update_order_charge', 'order_update_charge');
SELECT "agent_rename_tool"('update_recurring_shipment', 'shipment_update_recurring');
SELECT "agent_rename_tool"('update_shipment', 'shipment_update');
SELECT "agent_rename_tool"('update_shipment_hold', 'shipment_update_hold');
SELECT "agent_rename_tool"('update_shipment_permit', 'shipment_update_permit');
