import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { pageLoader } from "@/lib/route-prefetch";
import { DeskTriagePage } from "@/routes/desk/triage-page";
import { prefetch } from "@/routes/inbox/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_desk/desk/triage")({
  beforeLoad: gates(requirePermission(Resource.InboundMessage, Operation.Read)),
  component: DeskTriagePage,
  loader: pageLoader(prefetch),
});
