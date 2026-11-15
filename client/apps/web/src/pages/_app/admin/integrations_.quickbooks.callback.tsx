import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { QuickBooksCallbackPage } from "@/routes/admin/integrations/quickbooks-callback/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/integrations_/quickbooks/callback")({
  beforeLoad: gates(requirePermission(Resource.AccountingIntegration, Operation.Manage)),
  component: QuickBooksCallbackPage,
});
