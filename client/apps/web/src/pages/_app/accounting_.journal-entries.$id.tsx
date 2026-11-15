import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { JournalEntryDetailPage } from "@/routes/journal-entry/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/accounting_/journal-entries/$id")({
  beforeLoad: gates(requirePermission(Resource.JournalEntry)),
  component: JournalEntryDetailPage,
});
