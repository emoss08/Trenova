import type {
  AgentToolAutonomy,
  AgentToolPolicy,
  AgentToolRuleRow,
  AgentToolSafetyRow,
} from "@/lib/graphql/agent-safety";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { useT } from "@trenova/shared/i18n/use-t";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import { AnswerBadge, EgressBadges, HeldByChips } from "./safety-badges";
import {
  answerChoices,
  egressChoices,
  externalReadChoices,
  heldByChoices,
  heldByOf,
  kindChoices,
  kindLabel,
  needsLabel,
  readsOutsideLabel,
  resourceChoices,
  tierChoices,
  tierLabel,
} from "./safety-model";

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
      <AnswerBadge answer={autonomy.answer} />
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

function Muted({ children }: { children: React.ReactNode }) {
  return <span className="text-muted-foreground">{children}</span>;
}

/**
 * Every tool's rule, one line a tool. The rationale and the record condition
 * are long, so they open in the row's panel rather than stretching the row.
 */
export function getToolRuleColumns(
  t: TranslateFn,
  resources: readonly string[],
): ColumnDef<AgentToolRuleRow>[] {
  return [
    {
      accessorKey: "title",
      header: t("Tool"),
      cell: ({ row }) => <ToolName title={row.original.title} name={row.original.name} />,
      size: 260,
      meta: {
        label: t("Tool"),
        apiField: "title",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
    },
    {
      accessorKey: "name",
      header: t("Name"),
      cell: ({ row }) => (
        <span className="text-muted-foreground font-mono text-xs">{row.original.name}</span>
      ),
      size: 200,
      meta: {
        label: t("Name"),
        apiField: "name",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
    },
    {
      accessorKey: "egress",
      header: t("Who sees it"),
      cell: ({ row }) => <EgressBadges egress={row.original.egress} />,
      size: 200,
      meta: {
        label: t("Who sees it"),
        apiField: "egress",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: egressChoices(t),
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "promotableTier",
      header: t("Max tier"),
      cell: ({ row }) => <MaxTierCell policy={row.original} />,
      size: 150,
      meta: {
        label: t("Max tier"),
        apiField: "maxTier",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: tierChoices(t),
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "needs",
      header: t("Needs"),
      cell: ({ row }) => <Muted>{needsLabel(t, row.original)}</Muted>,
      size: 200,
      meta: {
        label: t("Needs"),
        apiField: "resource",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: resourceChoices(resources),
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "kind",
      header: t("Kind"),
      cell: ({ row }) => kindLabel(t, row.original.kind),
      size: 110,
      meta: {
        label: t("Kind"),
        apiField: "kind",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: kindChoices(t),
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "readsExternal",
      header: t("Reads outside content"),
      cell: ({ row }) => readsOutsideLabel(t, row.original) ?? <Muted>—</Muted>,
      size: 220,
      meta: {
        label: t("Reads outside content"),
        apiField: "readsExternal",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: externalReadChoices(t),
        defaultFilterOperator: "eq",
      },
    },
    {
      accessorKey: "runsWithoutPerson",
      header: t("Runs without a person"),
      cell: ({ row }) =>
        row.original.runsWithoutPerson ? t("On at least one agent") : <Muted>—</Muted>,
      size: 180,
      meta: {
        label: t("Runs without a person"),
        apiField: "runsWithoutPerson",
        filterable: true,
        sortable: true,
        filterType: "boolean",
      },
    },
  ];
}

/** What the compared agents make of each tool they hold, before and after outside text. */
export function getAgentToolColumns(
  t: TranslateFn,
  agents: readonly { id: string; name: string }[],
  resources: readonly string[],
): ColumnDef<AgentToolSafetyRow>[] {
  return [
    {
      accessorKey: "agentName",
      header: t("Agent"),
      cell: ({ row }) => <span className="truncate">{row.original.agentName}</span>,
      size: 180,
      meta: {
        label: t("Agent"),
        apiField: "agentId",
        filterable: true,
        sortable: false,
        filterType: "select",
        filterOptions: agents.map((agent) => ({ value: agent.id, label: agent.name })),
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "tool",
      accessorFn: (row) => row.policy.title,
      header: t("Tool"),
      cell: ({ row }) => (
        <ToolName title={row.original.policy.title} name={row.original.policyName} />
      ),
      size: 260,
      meta: {
        label: t("Tool"),
        apiField: "title",
        filterable: true,
        sortable: true,
        filterType: "text",
      },
    },
    {
      id: "egress",
      accessorFn: (row) => row.policy.egress,
      header: t("Who sees it"),
      cell: ({ row }) => <EgressBadges egress={row.original.policy.egress} />,
      size: 200,
      meta: {
        label: t("Who sees it"),
        apiField: "egress",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: egressChoices(t),
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "clean",
      accessorFn: (row) => row.clean.answer,
      header: t("Before outside text"),
      cell: ({ row }) => <AutonomyCell autonomy={row.original.clean} />,
      size: 200,
      meta: {
        label: t("Before outside text"),
        apiField: "clean",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: answerChoices(t),
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "tainted",
      accessorFn: (row) => row.tainted.answer,
      header: t("After outside text"),
      cell: ({ row }) => <AutonomyCell autonomy={row.original.tainted} />,
      size: 200,
      meta: {
        label: t("After outside text"),
        apiField: "tainted",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: answerChoices(t),
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "heldBy",
      accessorFn: (row) => heldByOf(row),
      header: t("Held by"),
      cell: ({ row }) => <HeldByChips heldBy={heldByOf(row.original)} />,
      size: 240,
      meta: {
        label: t("Held by"),
        apiField: "heldBy",
        filterable: true,
        sortable: false,
        filterType: "select",
        filterOptions: heldByChoices(t),
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "maxTier",
      accessorFn: (row) => row.policy.promotableTier,
      header: t("Max tier"),
      cell: ({ row }) => <MaxTierCell policy={row.original.policy} />,
      size: 150,
      meta: {
        label: t("Max tier"),
        apiField: "maxTier",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: tierChoices(t),
        defaultFilterOperator: "eq",
      },
    },
    {
      id: "needs",
      accessorFn: (row) => row.policy.needs?.resource ?? "",
      header: t("Needs"),
      cell: ({ row }) => <Muted>{needsLabel(t, row.original.policy)}</Muted>,
      size: 200,
      meta: {
        label: t("Needs"),
        apiField: "resource",
        filterable: true,
        sortable: true,
        filterType: "select",
        filterOptions: resourceChoices(resources),
        defaultFilterOperator: "eq",
      },
    },
  ];
}
