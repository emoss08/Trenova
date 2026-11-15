import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { pageLoader } from "@/lib/route-prefetch";
import { InboundMailboxesPage, prefetch } from "@/routes/admin/inbound-mailboxes/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/admin/inbound-mailboxes")({
  beforeLoad: gates(requirePermission(Resource.InboundMailbox, Operation.Read)),
  component: InboundMailboxesPage,
  loader: pageLoader(prefetch),
});
