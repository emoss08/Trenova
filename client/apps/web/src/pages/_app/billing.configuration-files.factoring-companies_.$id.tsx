import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { pageLoader } from "@/lib/route-prefetch";
import { FactoringCompanyPage, prefetch } from "@/routes/factoring-company/[id]/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/billing/configuration-files/factoring-companies_/$id")({
  beforeLoad: gates(requirePermission(Resource.FactoringCompany)),
  component: FactoringCompanyPage,
  loader: pageLoader(prefetch),
});
