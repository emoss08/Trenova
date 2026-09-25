import { MappingEditor } from "@/components/accounting-mapping/mapping-editor";
import { MappingRow } from "@/components/accounting-mapping/mapping-row";
import { ReferenceRefreshStatus } from "@/components/accounting-mapping/reference-refresh-status";
import { BillingDetailUnselected, BillingListEmpty } from "@/components/billing/billing-empty";
import { BillingWorkspaceLayout } from "@/components/billing/billing-workspace-layout";
import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { useAccountingMappingActions } from "@/hooks/use-accounting-mapping-actions";
import { useAccountingMappingLabels } from "@/hooks/use-accounting-mapping-labels";
import { useAccountingMappingList } from "@/hooks/use-accounting-mapping-list";
import { useAccountingMappingSummary } from "@/hooks/use-accounting-mapping-summary";
import { useInfiniteScrollSentinel } from "@/hooks/use-infinite-scroll-sentinel";
import { usePermission } from "@/hooks/use-permission";
import {
  ACCOUNTING_MAPPING_TARGET_TYPES,
  accountingSetupPath,
  hasLiveAccountingConnection,
} from "@/lib/accounting-sync";
import type {
  AccountingMappingFilterInput,
  AccountingMappingState,
  AccountingMappingTargetType,
  AccountingSystem,
} from "@trenova/graphql/generated/graphql";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { Input } from "@trenova/shared/components/ui/input";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { SearchIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { Link } from "react-router";

const SYSTEM: AccountingSystem = "QuickBooksOnline";
const PAGE_SIZE = 50;
const SEARCH_DELAY_MS = 250;
const ALL = "all";
const GHOST_ROWS = ["w-2/5", "w-1/3", "w-1/2"] as const;

export function AccountingMappingsPage() {
  const t = useT();
  const labels = useAccountingMappingLabels();
  const { allowed: canUpdate } = usePermission(Resource.AccountingIntegration, Operation.Update);
  const summary = useAccountingMappingSummary(SYSTEM, true);
  const providerName = summary.data?.providerName ?? "";
  const actions = useAccountingMappingActions(SYSTEM, providerName);

  const [targetType, setTargetType] = useState<AccountingMappingTargetType | null>(null);
  const [state, setState] = useState<AccountingMappingState | null>(null);
  const [requiredOnly, setRequiredOnly] = useState(false);
  const [search, setSearch] = useState("");
  const debouncedSearch = useDebounce(search.trim(), SEARCH_DELAY_MS);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [overrides, setOverrides] = useState<Record<string, boolean>>({});

  const filter = useMemo<AccountingMappingFilterInput>(
    () => ({
      targetTypes: targetType ? [targetType] : null,
      states: state ? [state] : null,
      requiredOnly,
      search: debouncedSearch || null,
    }),
    [targetType, state, requiredOnly, debouncedSearch],
  );
  const connection = summary.data?.connection ?? null;
  const live = hasLiveAccountingConnection(connection);
  const list = useAccountingMappingList({
    system: SYSTEM,
    filter,
    pageSize: PAGE_SIZE,
    enabled: live,
  });
  const { hasNextPage, isFetchingNextPage, fetchNextPage } = list;

  const sentinel = useInfiniteScrollSentinel<HTMLDivElement>({
    hasNextPage,
    isFetchingNextPage,
    onLoadMore: () => void fetchNextPage(),
  });

  const checkedIds = list.mappings
    .filter(
      (mapping) => mapping.state === "Proposed" && (overrides[mapping.id] ?? mapping.prechecked),
    )
    .map((mapping) => mapping.id);
  const selected = list.mappings.find((mapping) => mapping.id === selectedId) ?? null;
  const filtered = Boolean(targetType || state || requiredOnly || debouncedSearch);
  const totals = (summary.data?.groups ?? []).reduce(
    (sum, group) => ({
      unmatched: sum.unmatched + group.unmatched,
      proposed: sum.proposed + group.proposed,
      confirmed: sum.confirmed + group.confirmed,
    }),
    { unmatched: 0, proposed: 0, confirmed: 0 },
  );
  const pageHeaderProps = {
    title: t("Accounting mappings"),
    description: t(
      "Which accounting system account, item, customer and vendor each Trenova record is sent as. Only confirmed mappings are used when syncing.",
    ),
  };

  const clearFilters = () => {
    setTargetType(null);
    setState(null);
    setRequiredOnly(false);
    setSearch("");
  };
  const toggleState = (next: AccountingMappingState) => {
    setRequiredOnly(false);
    setState((current) => (current === next ? null : next));
  };

  if (summary.isLoading) {
    return (
      <PageLayout pageHeaderProps={pageHeaderProps}>
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-96 w-full" />
      </PageLayout>
    );
  }

  if (summary.isError || !summary.data) {
    return (
      <PageLayout pageHeaderProps={pageHeaderProps}>
        <Alert size="sm" variant="destructive">
          <AlertDescription className="flex items-center justify-between gap-3">
            <span>{t("The mappings could not be loaded.")}</span>
            <Button
              type="button"
              size="sm"
              variant="outline"
              onClick={() => void summary.refetch()}
            >
              {t("Retry")}
            </Button>
          </AlertDescription>
        </Alert>
      </PageLayout>
    );
  }

  if (!connection || !live) {
    return (
      <PageLayout pageHeaderProps={pageHeaderProps}>
        <EmptySheet
          title={t("{0} is not connected", summary.data.providerName)}
          description={t(
            "Connect it from the integrations page. Trenova reads its records and proposes the matches here.",
          )}
          sketch={
            <div className="flex flex-col gap-2 text-left">
              {GHOST_ROWS.map((width) => (
                <div key={width} className="flex items-center justify-between gap-3">
                  <GhostLine className={width} />
                  <GhostLine className="w-16" />
                </div>
              ))}
            </div>
          }
          action={
            <Button type="button" size="sm" render={<Link to={accountingSetupPath(SYSTEM)} />}>
              {t("Open integrations")}
            </Button>
          }
        />
      </PageLayout>
    );
  }

  return (
    <BillingWorkspaceLayout
      pageHeaderProps={{
        ...pageHeaderProps,
        actions:
          canUpdate && connection.setupStep === "Mappings" ? (
            <Button
              type="button"
              size="sm"
              isLoading={actions.completeSetup.isPending}
              disabled={!summary.data.canCompleteSetup}
              onClick={() => actions.completeSetup.mutate()}
            >
              {t("Finish setup")}
            </Button>
          ) : undefined,
      }}
      toolbar={
        <>
          <KpiStrip aria-label={t("Mapping totals")}>
            <KpiStripItem
              label={t("Required confirmed")}
              value={t("{0} of {1}", summary.data.requiredConfirmed, summary.data.requiredTotal)}
              tone={summary.data.canCompleteSetup ? "success" : "warning"}
              active={requiredOnly}
              onClick={() => {
                setState(null);
                setRequiredOnly((current) => !current);
              }}
            />
            <KpiStripItem
              label={t("Proposed")}
              value={totals.proposed}
              tone="warning"
              active={state === "Proposed"}
              onClick={() => toggleState("Proposed")}
            />
            <KpiStripItem
              label={t("Unmatched")}
              value={totals.unmatched}
              active={state === "Unmatched"}
              onClick={() => toggleState("Unmatched")}
            />
            <KpiStripItem
              label={t("Confirmed")}
              value={totals.confirmed}
              tone="success"
              active={state === "Confirmed"}
              onClick={() => toggleState("Confirmed")}
            />
          </KpiStrip>
          <ReferenceRefreshStatus
            connection={connection}
            providerName={summary.data.providerName}
            canUpdate={canUpdate}
            isRequesting={actions.refreshReference.isPending}
            onRefresh={() => actions.refreshReference.mutate()}
          />
        </>
      }
      sidebar={
        <div className="flex h-full flex-col">
          <div className="flex flex-col gap-1.5 border-b p-2">
            <Input
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder={t("Search mappings")}
              aria-label={t("Search mappings")}
              leftElement={<SearchIcon className="text-foreground-subtle size-3.5" />}
              className="h-7 text-xs"
            />
            <Select
              value={targetType ?? ALL}
              onValueChange={(value) =>
                setTargetType(value === ALL ? null : (value as AccountingMappingTargetType))
              }
            >
              <SelectTrigger className="h-7 text-xs" aria-label={t("Kind")}>
                <SelectValue placeholder={t("Every kind")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ALL}>{t("Every kind")}</SelectItem>
                {ACCOUNTING_MAPPING_TARGET_TYPES.map((type) => (
                  <SelectItem key={type} value={type}>
                    {labels.targetType(type)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {canUpdate ? (
              <Button
                type="button"
                size="sm"
                variant="outline"
                isLoading={actions.confirm.isPending}
                disabled={checkedIds.length === 0}
                onClick={() =>
                  actions.confirm.mutate(checkedIds, { onSuccess: () => setOverrides({}) })
                }
              >
                {t("Confirm {0} checked", checkedIds.length)}
              </Button>
            ) : null}
          </div>
          <ScrollArea className="min-h-0 flex-1">
            {list.isLoading ? (
              <div className="space-y-2 p-2">
                <Skeleton className="h-12 w-full" />
                <Skeleton className="h-12 w-full" />
                <Skeleton className="h-12 w-full" />
              </div>
            ) : list.isError ? (
              <p className="text-danger-foreground p-3 text-sm">
                {t("The mappings could not be loaded.")}
              </p>
            ) : list.mappings.length === 0 ? (
              <BillingListEmpty
                title={filtered ? t("No mappings match") : t("Nothing to map yet")}
                description={
                  filtered
                    ? t("Try another kind or search.")
                    : t("Mappings appear once {0}'s records have been read.", providerName)
                }
                onClearFilters={filtered ? clearFilters : undefined}
              />
            ) : (
              <div className="divide-border-subtle divide-y">
                {list.mappings.map((mapping) => (
                  <MappingRow
                    key={mapping.id}
                    mapping={mapping}
                    selected={mapping.id === selectedId}
                    checked={checkedIds.includes(mapping.id)}
                    canCheck={canUpdate}
                    onSelect={() => setSelectedId(mapping.id)}
                    onCheckedChange={(checked) =>
                      setOverrides((current) => ({ ...current, [mapping.id]: checked }))
                    }
                  />
                ))}
                <div ref={sentinel} className="h-px" />
              </div>
            )}
          </ScrollArea>
        </div>
      }
      detail={
        selected ? (
          <ScrollArea className="h-full">
            <div className="p-4">
              <MappingEditor
                key={selected.id}
                system={SYSTEM}
                providerName={providerName}
                mapping={selected}
                canUpdate={canUpdate}
              />
            </div>
          </ScrollArea>
        ) : (
          <BillingDetailUnselected
            layout="cards"
            title={t("Choose a mapping")}
            description={t(
              "Pick one on the left to see what Trenova proposed, the records it considered, and to choose another.",
            )}
          />
        )
      }
    />
  );
}
