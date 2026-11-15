import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { pageLoader } from "@/lib/route-prefetch";
import { LateChargesPage, prefetch } from "@/routes/late-charges/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/accounting_/ar/late-charges")({
  beforeLoad: gates(requirePermission(Resource.AccountsReceivable)),
  component: LateChargesPage,
  loader: pageLoader(prefetch),
});
