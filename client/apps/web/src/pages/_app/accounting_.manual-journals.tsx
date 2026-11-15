import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { ManualJournalsPage } from "@/routes/manual-journal/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/accounting_/manual-journals")({
  beforeLoad: gates(requirePermission(Resource.ManualJournal)),
  component: ManualJournalsPage,
});
