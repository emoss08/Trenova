import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { FactoringCompaniesPage } from "@/routes/factoring-company/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/billing/configuration-files/factoring-companies")({
  beforeLoad: gates(requirePermission(Resource.FactoringCompany)),
  component: FactoringCompaniesPage,
});
