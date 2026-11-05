-- A portal site says whether its account can change nothing on the portal: the only thing that makes a signed-in browser read only, so an automatic read of a site with a sign-in runs only on one.
ALTER TABLE "agent_portal_sites"
    ADD COLUMN IF NOT EXISTS "view_only_account" BOOLEAN NOT NULL DEFAULT FALSE;
