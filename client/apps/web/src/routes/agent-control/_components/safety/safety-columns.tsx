import type {
  AgentToolAutonomy,
  AgentToolPolicy,
  AgentToolSafety,
} from "@/lib/graphql/agent-safety";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { useT } from "@trenova/shared/i18n/use-t";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { AnswerBadge, EgressBadges, HeldByChips } from "./safety-badges";
import { needsLabel, readsOutsideLabel, tierLabel } from "./safety-model";

function ToolName({ title, name }: { title: string; name: string }) {
  return (
    <div className="flex min-w-0 flex-col">
      <span className="truncate">{title}</span>
      <span className="text-muted-foreground truncate font-mono text-xs">{name}</span>
    </div>
  );
}

function MaxTierCell({ policy }: { policy: AgentToolPolicy }) {
  const t = useT();
  const conditional = policy.hasClassify || policy.hasCondition || policy.personalExemption;

  return (
    <div className="flex flex-col">
      <span>{tierLabel(t, policy.promotableTier)}</span>
      {conditional ? (
        <span className="text-muted-foreground text-xs">{t("Depends on the call")}</span>
      ) : null}
    </div>
  );
}

function AutonomyCell({ autonomy }: { autonomy: AgentToolAutonomy }) {
  const t = useT();

  return (
    <div className="flex flex-col items-start gap-0.5">
      <AnswerBadge autonomy={autonomy} />
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
  );
}

/** Every tool's rule, one line a tool; the rationale opens underneath. */
export function toolRuleColumns(t: TranslateFn): ColumnDef<AgentToolPolicy>[] {
  return [
    {
      id: "tool",
      header: t("Tool"),
      cell: ({ row }) => <ToolName title={row.original.title} name={row.original.name} />,
      meta: { cellClassName: "max-w-72" },
    },
    {
      id: "egress",
      header: t("Who sees it"),
      cell: ({ row }) => <EgressBadges policy={row.original} />,
    },
    {
      id: "maxTier",
      header: t("Max tier"),
      cell: ({ row }) => <MaxTierCell policy={row.original} />,
    },
    {
      id: "needs",
      header: t("Needs"),
      cell: ({ row }) => (
        <span className="text-muted-foreground">{needsLabel(t, row.original)}</span>
      ),
    },
    {
      id: "readsOutside",
      header: t("Reads outside content"),
      cell: ({ row }) =>
        readsOutsideLabel(t, row.original) ?? <span className="text-muted-foreground">—</span>,
    },
  ];
}

/** What one agent makes of each tool it holds, before and after outside text. */
export function agentToolColumns(t: TranslateFn): ColumnDef<AgentToolSafety>[] {
  return [
    {
      id: "tool",
      header: t("Tool"),
      cell: ({ row }) => (
        <ToolName title={row.original.policy.title} name={row.original.policyName} />
      ),
      meta: { cellClassName: "max-w-72" },
    },
    {
      id: "egress",
      header: t("Who sees it"),
      cell: ({ row }) => <EgressBadges policy={row.original.policy} />,
    },
    {
      id: "clean",
      header: t("Before outside text"),
      cell: ({ row }) => <AutonomyCell autonomy={row.original.clean} />,
    },
    {
      id: "tainted",
      header: t("After outside text"),
      cell: ({ row }) => <AutonomyCell autonomy={row.original.tainted} />,
    },
    {
      id: "heldBy",
      header: t("Held by"),
      cell: ({ row }) => (
        <HeldByChips
          heldBy={[...new Set([...row.original.clean.heldBy, ...row.original.tainted.heldBy])]}
        />
      ),
    },
  ];
}
