import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { CollectionsPage } from "@/routes/collections/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/accounting_/ar/collections")({
  beforeLoad: gates(requirePermission(Resource.AccountsReceivable)),
  component: CollectionsPage,
});
