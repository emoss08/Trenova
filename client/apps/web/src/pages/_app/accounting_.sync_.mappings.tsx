import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { AccountingMappingsPage } from "@/routes/accounting-mapping/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/accounting_/sync_/mappings")({
  beforeLoad: gates(requirePermission(Resource.AccountingIntegration)),
  component: AccountingMappingsPage,
});
