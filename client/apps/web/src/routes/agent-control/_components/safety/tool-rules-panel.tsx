import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import type { AgentEgressClass, AgentToolPolicy } from "@/lib/graphql/agent-safety";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { useMemo, useState } from "react";
import { resourceLabel } from "../agents/tool-catalog";
import { EgressBadges } from "./safety-badges";
import {
  ALL_POLICIES,
  EGRESS_ORDER,
  egressLabel,
  filterPolicies,
  needsLabel,
  readsOutsideLabel,
  resourceOptions,
  tierLabel,
  type PolicyFilter,
} from "./safety-model";

/**
 * Every tool an agent can be given, with the rule the runtime holds it to.
 * The rules are declared beside each tool in code; this is the same list the
 * runtime decides from, so nothing here can promise more than a call gets.
 */
export function ToolRulesPanel({ policies }: { policies: readonly AgentToolPolicy[] }) {
  const t = useT();
  const [filter, setFilter] = useState<PolicyFilter>(ALL_POLICIES);

  const classItems = useMemo(
    () => [
      { value: "all", label: t("All classes") },
      ...EGRESS_ORDER.map((egress) => ({ value: egress, label: egressLabel(t, egress) })),
    ],
    [t],
  );
  const resourceItems = useMemo(
    () => [
      { value: "all", label: t("All resources") },
      ...resourceOptions(policies).map((resource) => ({
        value: resource,
        label: resourceLabel(resource),
      })),
    ],
    [policies, t],
  );
  const visible = useMemo(() => filterPolicies(policies, filter), [filter, policies]);

  return (
    <SectionPanel
      title={t("Tool rules")}
      count={visible.length}
      help={t(
        "Each tool declares who sees its work, the most it may run at, and whether it reads text written outside the organization. Work that reaches a customer, a driver or anyone outside never runs past approval.",
      )}
      action={
        <div className="flex items-center gap-2">
          <Select
            items={classItems}
            value={filter.egress}
            onValueChange={(value) =>
              setFilter((current) => ({
                ...current,
                egress: (value ?? "all") as AgentEgressClass | "all",
              }))
            }
          >
            <SelectTrigger size="sm" className="w-40" aria-label={t("Who sees it")}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {classItems.map((item) => (
                <SelectItem key={item.value} value={item.value}>
                  {item.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Select
            items={resourceItems}
            value={filter.resource}
            onValueChange={(value) =>
              setFilter((current) => ({ ...current, resource: value ?? "all" }))
            }
          >
            <SelectTrigger size="sm" className="w-40" aria-label={t("Resource")}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {resourceItems.map((item) => (
                <SelectItem key={item.value} value={item.value}>
                  {item.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      }
    >
      {visible.length === 0 ? (
        <SectionPanelQuiet>{t("No tool matches these filters.")}</SectionPanelQuiet>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("Tool")}</TableHead>
              <TableHead>{t("Who sees it")}</TableHead>
              <TableHead>{t("Max tier")}</TableHead>
              <TableHead>{t("Needs")}</TableHead>
              <TableHead>{t("Reads outside content")}</TableHead>
              <TableHead>{t("Rationale")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {visible.map((policy) => (
              <ToolRuleRow key={policy.name} policy={policy} />
            ))}
          </TableBody>
        </Table>
      )}
    </SectionPanel>
  );
}

function ToolRuleRow({ policy }: { policy: AgentToolPolicy }) {
  const t = useT();
  const outside = readsOutsideLabel(t, policy);
  const conditional = policy.hasClassify || policy.hasCondition || policy.personalExemption;

  return (
    <TableRow>
      <TableCell className="align-top">
        <div className="flex min-w-0 flex-col">
          <span>{policy.title}</span>
          <span className="text-muted-foreground font-mono text-xs">{policy.name}</span>
        </div>
      </TableCell>
      <TableCell className="align-top">
        <EgressBadges policy={policy} />
      </TableCell>
      <TableCell className="align-top">
        <div className="flex flex-col">
          <span>{tierLabel(t, policy.promotableTier)}</span>
          {conditional ? (
            <Tooltip>
              <TooltipTrigger
                render={
                  <span className="text-muted-foreground w-fit cursor-help text-xs underline decoration-dotted">
                    {t("Depends on the call")}
                  </span>
                }
              />
              <TooltipContent className="max-w-xs">{policy.explanation}</TooltipContent>
            </Tooltip>
          ) : null}
        </div>
      </TableCell>
      <TableCell className="text-muted-foreground align-top">{needsLabel(t, policy)}</TableCell>
      <TableCell className="align-top">
        {outside === null ? <span className="text-muted-foreground">—</span> : outside}
      </TableCell>
      <TableCell className="text-muted-foreground min-w-64 align-top text-xs whitespace-normal">
        {policy.rationale}
      </TableCell>
    </TableRow>
  );
}
