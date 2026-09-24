import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import type {
  AgentToolAutonomy,
  AgentToolRuleRow,
  AgentToolSafetyRow,
} from "@/lib/graphql/agent-safety";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { PolicyDetails } from "./policy-details";
import { AnswerBadge, EgressBadges, HeldByChips } from "./safety-badges";
import { tierLabel } from "./safety-model";

/** One tool's whole rule, read-only: the table shows it a line at a time. */
export function ToolRulePanel({ open, onOpenChange, row }: DataTablePanelProps<AgentToolRuleRow>) {
  const t = useT();

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={row?.title ?? t("Tool rule")}
      description={row?.name}
      size="lg"
    >
      {row ? (
        <div className="flex flex-col gap-4">
          <EgressBadges egress={row.egress} />
          <PolicyDetails policy={row} />
        </div>
      ) : null}
    </DataTablePanelContainer>
  );
}

/** What one agent makes of one tool it holds, and the rule the answers come from. */
export function AgentToolPanel({
  open,
  onOpenChange,
  row,
}: DataTablePanelProps<AgentToolSafetyRow>) {
  const t = useT();

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={row?.policy.title ?? t("Held tool")}
      description={row ? t("Held by {0}", row.agentName) : undefined}
      size="lg"
    >
      {row ? (
        <div className="flex flex-col gap-4">
          <EgressBadges egress={row.policy.egress} />
          <DescriptionList layout="stacked" columns={2}>
            <AutonomyItem label={t("Before outside text")} autonomy={row.clean} />
            <AutonomyItem label={t("After outside text")} autonomy={row.tainted} />
          </DescriptionList>
          <PolicyDetails policy={row.policy} />
        </div>
      ) : null}
    </DataTablePanelContainer>
  );
}

function AutonomyItem({ label, autonomy }: { label: string; autonomy: AgentToolAutonomy }) {
  const t = useT();

  return (
    <DescriptionItem label={label}>
      <div className="flex flex-col items-start gap-1.5">
        <div className="flex items-center gap-2">
          <AnswerBadge answer={autonomy.answer} />
          <span className="text-muted-foreground text-xs">
            {t("Up to {0}", tierLabel(t, autonomy.tier))}
          </span>
        </div>
        <HeldByChips heldBy={autonomy.heldBy} />
        {autonomy.earned ? (
          <span className="text-muted-foreground text-xs">{t("Tier earned")}</span>
        ) : null}
        {autonomy.approvalsToNext != null ? (
          <span className="text-muted-foreground text-xs">
            {t(
              "{0, plural, one {# clean approval to the next tier} other {# clean approvals to the next tier}}",
              autonomy.approvalsToNext,
            )}
          </span>
        ) : null}
      </div>
    </DescriptionItem>
  );
}
