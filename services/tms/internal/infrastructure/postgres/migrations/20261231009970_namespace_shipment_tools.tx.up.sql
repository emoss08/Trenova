-- Shipment, order and import-page tools now carry their namespace (shipment_, order_, intake_), and search_shipments and list_shipments are one tool, shipment_list.
-- Renames an agent tool everywhere a tenant's saved agent configuration names it: the
-- definitions' tool lists, disabled lists and tiers, earned trust, tool rule overrides
-- and memories about one tool. Two tools merged into one keep a single entry; where both
-- names already have a row keyed by tool, the old one is dropped. A captured eval case
-- replays the tools it held when it was captured, so one that held the old name is
-- quarantined rather than rewritten.
CREATE OR REPLACE FUNCTION "agent_rename_tool"("old_name" text, "new_name" text)
RETURNS void
LANGUAGE plpgsql
AS $fn$
BEGIN
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

    UPDATE "agent_definitions"
    SET "tool_tiers" = ("tool_tiers" - "old_name")
        || jsonb_build_object("new_name", "tool_tiers" -> "old_name")
    WHERE "tool_tiers" ? "old_name"
      AND NOT ("tool_tiers" ? "new_name");

    UPDATE "agent_definitions"
    SET "tool_tiers" = "tool_tiers" - "old_name"
    WHERE "tool_tiers" ? "old_name";

    DELETE FROM "agent_tool_trust" AS "stale"
    WHERE "stale"."tool_name" = "old_name"
      AND EXISTS (
          SELECT 1 FROM "agent_tool_trust" AS "kept"
          WHERE "kept"."organization_id" = "stale"."organization_id"
            AND "kept"."business_unit_id" = "stale"."business_unit_id"
            AND "kept"."agent_definition_id" = "stale"."agent_definition_id"
            AND "kept"."tool_name" = "new_name"
      );

    UPDATE "agent_tool_trust" SET "tool_name" = "new_name" WHERE "tool_name" = "old_name";

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

--bun:split
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
