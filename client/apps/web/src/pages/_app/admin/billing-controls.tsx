import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { BillingControlPage } from "@/routes/billing-control/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/billing-controls")({
  beforeLoad: gates(requirePermission(Resource.BillingControl)),
  component: BillingControlPage,
});
