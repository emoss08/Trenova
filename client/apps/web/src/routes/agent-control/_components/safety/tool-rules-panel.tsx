import { SECTION_TABLE_PAGE_SIZE, SectionTable } from "@/components/data-table/section-table";
import { SectionPanel } from "@/components/section-panel";
import { useCursorPages } from "@/hooks/use-cursor-pages";
import {
  ALL_TOOL_POLICIES,
  toolPolicyConnectionInput,
  type AgentEgressClass,
  type AgentToolKind,
  type AgentToolPolicy,
  type ToolPolicyAttendance,
  type ToolPolicyFilter,
} from "@/lib/graphql/agent-safety";
import { queries } from "@/lib/queries";
import { stableStringify } from "@/lib/stable-stringify";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { Input } from "@trenova/shared/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { useT } from "@trenova/shared/i18n/use-t";
import { SearchIcon } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { resourceLabel } from "../agents/tool-catalog";
import { PolicyDetails } from "./policy-details";
import { toolRuleColumns } from "./safety-columns";
import { EGRESS_ORDER, KIND_ORDER, egressLabel, kindLabel, sortResources } from "./safety-model";

/** How long typing settles before the search goes to the server. */
const SEARCH_DEBOUNCE_MS = 250;
/** The rules change only with a deploy, so a page read once is good for a while. */
const POLICY_STALE_MS = 5 * 60_000;

type FilterItem = { value: string; label: string };

const policyRowId = (policy: AgentToolPolicy) => policy.name;
const policyRowLabel = (policy: AgentToolPolicy) => policy.title;
const renderPolicyDetails = (policy: AgentToolPolicy) => <PolicyDetails policy={policy} />;

/**
 * Every tool an agent can be given, with the rule the runtime holds it to, a
 * page at a time. The rules are declared beside each tool in code and paged,
 * searched and filtered on the server, so the screen draws one page however
 * many tools there are.
 */
export function ToolRulesPanel() {
  const t = useT();
  const [searchInput, setSearchInput] = useState("");
  const [filters, setFilters] = useState<Omit<ToolPolicyFilter, "search">>(ALL_TOOL_POLICIES);
  const [pageSize, setPageSize] = useState<number>(SECTION_TABLE_PAGE_SIZE);
  const search = useDebounce(searchInput.trim(), SEARCH_DEBOUNCE_MS);
  const filter = useMemo<ToolPolicyFilter>(() => ({ ...filters, search }), [filters, search]);

  const scopeKey = useMemo(
    () =>
      stableStringify({
        input: toolPolicyConnectionInput(filter, { first: pageSize, after: null }),
      }),
    [filter, pageSize],
  );
  const pages = useCursorPages(scopeKey);
  const { pageIndex, recordPage } = pages;

  const policyQuery = useQuery({
    ...queries.agentSafety.policyPage(filter, {
      first: pageSize,
      after: pages.after,
      includeTotalCount: pageIndex === 0,
    }),
    placeholderData: keepPreviousData,
    staleTime: POLICY_STALE_MS,
  });
  const summaryQuery = useQuery({ ...queries.agentSafety.summary(), staleTime: 60_000 });

  const page = policyQuery.data;
  const landed = page && !policyQuery.isPlaceholderData ? page : undefined;
  useEffect(() => {
    if (!landed) {
      return;
    }
    recordPage({
      pageIndex,
      endCursor: landed.endCursor,
      hasNextPage: landed.hasNextPage,
      totalCount: landed.totalCount,
    });
  }, [landed, pageIndex, recordPage]);

  const columns = useMemo(() => toolRuleColumns(t), [t]);

  const classItems = useMemo<FilterItem[]>(
    () => [
      { value: "all", label: t("All classes") },
      ...EGRESS_ORDER.map((egress) => ({ value: egress, label: egressLabel(t, egress) })),
    ],
    [t],
  );
  const resources = summaryQuery.data?.resources;
  const resourceItems = useMemo<FilterItem[]>(
    () => [
      { value: "all", label: t("All resources") },
      ...sortResources(resources ?? []).map((resource) => ({
        value: resource,
        label: resourceLabel(resource),
      })),
    ],
    [resources, t],
  );
  const kindItems = useMemo<FilterItem[]>(
    () => [
      { value: "all", label: t("All kinds") },
      ...KIND_ORDER.map((kind) => ({ value: kind, label: kindLabel(t, kind) })),
    ],
    [t],
  );
  const attendanceItems = useMemo<FilterItem[]>(
    () => [
      { value: "all", label: t("With or without a person") },
      { value: "alone", label: t("Runs without a person") },
    ],
    [t],
  );

  const setFilter = useCallback(
    <K extends keyof Omit<ToolPolicyFilter, "search">>(
      key: K,
      value: Omit<ToolPolicyFilter, "search">[K],
    ) => {
      setFilters((current) => ({ ...current, [key]: value }));
    },
    [],
  );

  const filtered =
    search !== "" ||
    filters.egress !== "all" ||
    filters.resource !== "all" ||
    filters.kind !== "all" ||
    filters.attendance !== "all";

  return (
    <SectionPanel
      title={t("Tool rules")}
      count={pages.totalCount ?? undefined}
      help={t(
        "Each tool declares who sees its work, the most it may run at, and whether it reads text written outside the organization. Work that reaches a customer, a driver or anyone outside never runs past approval.",
      )}
    >
      <div className="border-border flex flex-wrap items-center gap-2 border-b px-3 py-2">
        <Input
          type="search"
          value={searchInput}
          onChange={(event) => setSearchInput(event.target.value)}
          placeholder={t("Search tools")}
          aria-label={t("Search tools")}
          maxLength={200}
          inputContainerClassName="w-52"
          leftElement={<SearchIcon className="text-muted-foreground size-3.5 shrink-0" />}
        />
        <FilterSelect
          label={t("Who sees it")}
          items={classItems}
          value={filters.egress}
          onChange={(value) => setFilter("egress", value as AgentEgressClass | "all")}
        />
        <FilterSelect
          label={t("Resource")}
          items={resourceItems}
          value={filters.resource}
          onChange={(value) => setFilter("resource", value)}
        />
        <FilterSelect
          label={t("Kind")}
          items={kindItems}
          value={filters.kind}
          onChange={(value) => setFilter("kind", value as AgentToolKind | "all")}
        />
        <FilterSelect
          label={t("Runs without a person")}
          items={attendanceItems}
          value={filters.attendance}
          onChange={(value) => setFilter("attendance", value as ToolPolicyAttendance)}
          className="w-52"
        />
      </div>
      <SectionTable
        label={t("Tool rules")}
        columns={columns}
        rows={page?.items}
        getRowId={policyRowId}
        rowLabel={policyRowLabel}
        renderDetails={renderPolicyDetails}
        isLoading={policyQuery.isPending}
        isRefreshing={policyQuery.isPlaceholderData || searchInput.trim() !== search}
        error={policyQuery.isError ? t("Tool rules could not be loaded.") : null}
        onRetry={() => void policyQuery.refetch()}
        empty={filtered ? t("No tool matches these filters.") : t("No tool rules yet.")}
        pagination={{
          mode: "cursor",
          pageIndex,
          pageSize,
          hasNextPage: page?.hasNextPage ?? false,
          totalCount: pages.totalCount,
          onPageChange: pages.goToPage,
          onPageSizeChange: setPageSize,
        }}
      />
    </SectionPanel>
  );
}

type FilterSelectProps = {
  label: string;
  items: FilterItem[];
  value: string;
  onChange: (value: string) => void;
  className?: string;
};

function FilterSelect({ label, items, value, onChange, className = "w-40" }: FilterSelectProps) {
  return (
    <Select items={items} value={value} onValueChange={(next) => onChange(next ?? "all")}>
      <SelectTrigger size="sm" className={className} aria-label={label}>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {items.map((item) => (
          <SelectItem key={item.value} value={item.value}>
            {item.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
