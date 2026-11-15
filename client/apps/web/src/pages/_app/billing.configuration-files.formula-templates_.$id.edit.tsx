import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { pageLoader } from "@/lib/route-prefetch";
import { FormulaStudioEditPage, prefetch } from "@/routes/formula-template/[id]/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Resource } from "@trenova/shared/types/permission";

export const Route = createFileRoute(
  "/_app/billing/configuration-files/formula-templates_/$id/edit",
)({
  beforeLoad: gates(requirePermission(Resource.FormulaTemplate)),
  staticData: { frame: "room", roomRail: "stands" },
  component: FormulaStudioEditPage,
  loader: pageLoader(prefetch),
});
