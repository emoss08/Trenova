import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { pageLoader } from "@/lib/route-prefetch";
import { SupportAccessPage, prefetch } from "@/routes/admin/support-access/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/support-access")({
  beforeLoad: gates(requirePermission(Resource.SupportAccess, Operation.Read)),
  component: SupportAccessPage,
  loader: pageLoader(prefetch),
});
