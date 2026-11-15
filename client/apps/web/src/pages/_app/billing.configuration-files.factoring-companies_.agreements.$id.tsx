import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { FactoringAgreementRedirectPage } from "@/routes/factoring-company/record-redirect-page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute(
  "/_app/billing/configuration-files/factoring-companies_/agreements/$id",
)({
  beforeLoad: gates(requirePermission(Resource.FactoringAgreement)),
  component: FactoringAgreementRedirectPage,
});
