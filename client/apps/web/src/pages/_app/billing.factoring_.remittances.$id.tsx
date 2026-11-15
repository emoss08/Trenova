import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { pageLoader } from "@/lib/route-prefetch";
import { FactoringRemittancePage, prefetch } from "@/routes/factoring-remittance/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/billing/factoring_/remittances/$id")({
  beforeLoad: gates(requirePermission(Resource.FactoringRemittance)),
  component: FactoringRemittancePage,
  loader: pageLoader(prefetch),
});
