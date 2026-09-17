import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { usePermission } from "@/hooks/use-permission";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { ActivityIcon, InboxIcon, TriangleAlertIcon } from "lucide-react";
import { parseAsStringLiteral, useQueryState } from "nuqs";
import { lazy, useMemo } from "react";

const AgentRunTable = lazy(() => import("./agent-run-table"));
const AgentProposalTable = lazy(() => import("./agent-proposal-table"));
const AgentExceptionTable = lazy(() => import("./agent-exception-table"));

const views = ["runs", "proposals", "exceptions"] as const;
type ActivityView = (typeof views)[number];

const activityViewParser = parseAsStringLiteral(views)
  .withOptions({ history: "replace", shallow: true })
  .withDefault("runs");

export default function ActivityTab() {
  const t = useT();
  const [view, setView] = useQueryState("activity", activityViewParser);

  const { allowed: canReadProposals } = usePermission(Resource.AgentProposal, Operation.Read);
  const { allowed: canReadExceptions } = usePermission(Resource.AgentException, Operation.Read);

  const items = useMemo(
    () =>
      [
        { value: "runs" as const, label: t("Runs"), icon: ActivityIcon },
        canReadProposals
          ? { value: "proposals" as const, label: t("Proposals"), icon: InboxIcon }
          : null,
        canReadExceptions
          ? { value: "exceptions" as const, label: t("Exceptions"), icon: TriangleAlertIcon }
          : null,
      ].filter((item) => item !== null),
    [canReadExceptions, canReadProposals, t],
  );

  const activeView: ActivityView = items.some((item) => item.value === view) ? view : "runs";

  return (
    <section className="flex flex-col gap-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className="text-base font-semibold">{t("Activity")}</h2>
          <p className="text-muted-foreground max-w-prose text-sm">
            {t(
              "Every run an agent made, the changes it proposed, and the cases it could not resolve on its own.",
            )}
          </p>
        </div>
        <SegmentedControl
          items={items}
          value={activeView}
          onValueChange={(value) => void setView(value)}
          aria-label={t("Activity view")}
        />
      </div>

      <DataTableLazyComponent>
        {activeView === "runs" && <AgentRunTable />}
        {activeView === "proposals" && <AgentProposalTable />}
        {activeView === "exceptions" && <AgentExceptionTable />}
      </DataTableLazyComponent>
    </section>
  );
}
