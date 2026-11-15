import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { SourceDrillDownPage } from "@/routes/journal-entry/_components/source-drill-down-page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/accounting_/journal-entries/source/$type/$sourceId")({
  beforeLoad: gates(requirePermission(Resource.JournalEntry)),
  component: SourceDrillDownPage,
});
