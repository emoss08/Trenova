-- An exception that is someone's own is now told by its own title, and the panel says why the
-- person got it, so the old lead-ins ("Exception assigned to you: …") only repeat that.
UPDATE "notifications"
SET "title" = regexp_replace(
        "title",
        '^(Exception assigned to you|Overdue exception escalated|New comment on your exception|Snoozed exception is back): ',
        ''
    )
WHERE "subject_key" LIKE 'watchtower:item:%';

-- A notification clears itself when the work it points at is finished. These were left open
-- for exceptions that had already been resolved.
UPDATE "notifications" n
SET "dismissed_at" = EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "resolved_at" = EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "resolution" = 'resolved',
    "read_at" = coalesce(n."read_at", EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint)
FROM "watchtower_items" wi
WHERE n."dismissed_at" IS NULL
  AND n."organization_id" = wi."organization_id"
  AND n."subject_key" = 'watchtower:item:' || wi."id"
  AND wi."status" = 'Resolved';
