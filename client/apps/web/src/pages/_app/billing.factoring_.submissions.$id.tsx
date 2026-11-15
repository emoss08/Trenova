import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { FactoringSubmissionPage } from "@/routes/factoring-submission/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/billing/factoring_/submissions/$id")({
  beforeLoad: gates(requirePermission(Resource.FactoringSubmission)),
  component: FactoringSubmissionPage,
});
