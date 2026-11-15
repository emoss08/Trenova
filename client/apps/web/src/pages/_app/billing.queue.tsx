import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { pageLoader } from "@/lib/route-prefetch";
import { BillingQueuePage, prefetch } from "@/routes/billing-queue/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/billing/queue")({
  beforeLoad: gates(requirePermission(Resource.BillingQueue)),
  component: BillingQueuePage,
  loader: pageLoader(prefetch),
});
