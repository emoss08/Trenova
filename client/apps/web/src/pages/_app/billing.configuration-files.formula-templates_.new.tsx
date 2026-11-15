import { createFileRoute } from "@tanstack/react-router";
import { requirePermission } from "@/lib/route-permission";
import { pageLoader } from "@/lib/route-prefetch";
import { FormulaStudioCreatePage, prefetch } from "@/routes/formula-template/new/page";
import { gates } from "@trenova/shared/lib/router/gates";
import { Operation, Resource } from "@trenova/shared/types/permission";

// The studio is a room: the module rail stays, and its rate trace takes
// the panel's place and always stands, the panel folded in its favour.
export const Route = createFileRoute("/_app/billing/configuration-files/formula-templates_/new")({
  beforeLoad: gates(requirePermission(Resource.FormulaTemplate, Operation.Create)),
  staticData: { frame: "room", roomRail: "stands" },
  component: FormulaStudioCreatePage,
  loader: pageLoader(prefetch),
});
