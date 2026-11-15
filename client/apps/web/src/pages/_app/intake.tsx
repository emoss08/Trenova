import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { pageLoader } from "@/lib/route-prefetch";
import { IntakePage, prefetch } from "@/routes/intake/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/intake")({
  beforeLoad: gates(requirePermission(Resource.CaptureBatch, Operation.Read)),
  component: IntakePage,
  loader: pageLoader(prefetch),
});
