import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { JournalReviewPage } from "@/routes/journal-review/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/accounting_/journals-to-post")({
  beforeLoad: gates(requirePermission(Resource.JournalEntry)),
  component: JournalReviewPage,
});
