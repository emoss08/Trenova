import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { RateMatrixPage } from "@/routes/rate-matrix/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute("/_app/billing/configuration-files/rate-matrices")({
  beforeLoad: gates(requirePermission(Resource.RateMatrix)),
  component: RateMatrixPage,
});
