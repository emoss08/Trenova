import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { BusinessCentralCallbackPage } from "@/routes/admin/integrations/business-central-callback/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/integrations_/business-central/callback")({
  beforeLoad: gates(requirePermission(Resource.AccountingIntegration, Operation.Manage)),
  component: BusinessCentralCallbackPage,
});
