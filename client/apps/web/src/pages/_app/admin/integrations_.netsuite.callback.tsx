import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { NetSuiteCallbackPage } from "@/routes/admin/integrations/netsuite-callback/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/integrations_/netsuite/callback")({
  beforeLoad: gates(requirePermission(Resource.AccountingIntegration, Operation.Manage)),
  component: NetSuiteCallbackPage,
});
