import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { JournalReversalsPage } from "@/routes/journal-reversal/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/accounting_/journal-reversals")({
  beforeLoad: gates(requirePermission(Resource.JournalReversal)),
  component: JournalReversalsPage,
});
