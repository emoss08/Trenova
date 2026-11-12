-- Triage has one address, /desk/triage. A message's Watchtower item kept the
-- link it was raised with, under the inbox's old address, which no route
-- answers any more. Rewrite them, so an item raised before this opens its
-- message as one raised after does.
UPDATE "watchtower_items"
SET "path" = '/desk/triage' || substr("path", length('/inbox') + 1)
WHERE "path" = '/inbox' OR "path" LIKE '/inbox?%';
