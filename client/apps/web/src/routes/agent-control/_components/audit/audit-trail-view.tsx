import { DataTable } from "@/components/data-table/data-table";
import { searchParamsParser } from "@/hooks/data-table/use-data-table-state";
import { usePermission } from "@/hooks/use-permission";
import {
  AI_AUDIT_EVENT_LIST_KEY,
  aiAuditEventTableGraphQLConfig,
  type AIAuditEventRow,
} from "@/lib/graphql/ai-audit";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptyTable } from "@trenova/shared/components/ui/empty-table";
import { useT } from "@trenova/shared/i18n/use-t";
import type { DataTableEmptyStateRenderProps } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { FileDownIcon } from "lucide-react";
import { useQueryStates } from "nuqs";
import { useCallback, useMemo, useState } from "react";
import { getAuditEventColumns } from "./audit-event-columns";
import { AuditEventPanel } from "./audit-event-panel";
import {
  AUDIT_SCOPE_PARAM,
  auditScopeFilters,
  auditScopeParser,
  type AuditTableState,
  type AuditTrailScope,
} from "./audit-model";
import { AuditScopeBar } from "./audit-scope-bar";
import { ChainStatusStrip } from "./chain-status";
import { ExportTrailDialog } from "./export-trail-dialog";

/** The trail is written by a projector every minute; the table reads again on that beat. */
const TRAIL_REFRESH_MS = 60_000;

const EMPTY_COLUMNS = [
  { label: "When" },
  { label: "Agent" },
  { label: "What" },
  { label: "Outcome" },
  { label: "Person" },
  { label: "Record" },
] as const;

const trailParsers = {
  ...searchParamsParser,
  [AUDIT_SCOPE_PARAM]: auditScopeParser,
};

function TrailEmpty({ hasActiveFilters, onClearFilters }: DataTableEmptyStateRenderProps) {
  const t = useT();

  return (
    <EmptyTable
      className="py-10"
      title={hasActiveFilters ? t("Nothing matches") : t("Nothing on the trail in this range")}
      description={
        hasActiveFilters
          ? t("No event fits the search and filters. Widen them, or clear them to see every one.")
          : t(
              "Agent work reaches the trail within a minute of happening. Widen the range, pick another agent or person, or include evaluations.",
            )
      }
      columns={EMPTY_COLUMNS}
      onClearFilters={hasActiveFilters ? onClearFilters : undefined}
    />
  );
}

/**
 * The signed trail of what agents did: the chain's status above, the range,
 * agent and person it is narrowed to beside it, and every event in the
 * standard table below, each opening in full.
 */
export default function AuditTrailView({ onOpenExports }: { onOpenExports: () => void }) {
  const t = useT();
  const [params, setParams] = useQueryStates(trailParsers);
  const scope = params[AUDIT_SCOPE_PARAM];
  const { allowed: canExport } = usePermission(Resource.AIAuditTrail, Operation.Export);
  const { allowed: canPickAgent } = usePermission(Resource.AgentDefinition, Operation.Read);
  const [exportOpen, setExportOpen] = useState(false);

  const columns = useMemo(() => getAuditEventColumns(t), [t]);
  const scopeFilters = useMemo(() => auditScopeFilters(scope), [scope]);
  const table = useMemo<AuditTableState>(
    () => ({
      query: params.query,
      fieldFilters: params.fieldFilters,
      filterGroups: params.filterGroups,
      sort: params.sort,
    }),
    [params.fieldFilters, params.filterGroups, params.query, params.sort],
  );

  const changeScope = useCallback(
    (next: AuditTrailScope) => {
      void setParams({ [AUDIT_SCOPE_PARAM]: next, pageIndex: 1 });
    },
    [setParams],
  );

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <ChainStatusStrip />
      <AuditScopeBar
        scope={scope}
        onChange={changeScope}
        canPickAgent={canPickAgent}
        actions={
          canExport ? (
            <Button type="button" variant="outline" size="sm" onClick={() => setExportOpen(true)}>
              <FileDownIcon className="size-3.5" />
              {t("Export trail…")}
            </Button>
          ) : null
        }
      />
      <DataTable<AIAuditEventRow>
        name="AI Audit Event"
        queryKey={AI_AUDIT_EVENT_LIST_KEY}
        graphql={aiAuditEventTableGraphQLConfig}
        resource={Resource.AIAuditTrail}
        columns={columns}
        scopeFilters={scopeFilters}
        enableExport={false}
        enableCreateAction={false}
        enableReadOnlyPanel
        TablePanel={AuditEventPanel}
        refetchIntervalMs={TRAIL_REFRESH_MS}
        initialColumnVisibility={{ toolName: false, entityType: false, seq: false }}
        renderEmptyState={(state) => <TrailEmpty {...state} />}
      />
      {canExport ? (
        <ExportTrailDialog
          open={exportOpen}
          onOpenChange={setExportOpen}
          scope={scope}
          table={table}
          onOpenExports={onOpenExports}
        />
      ) : null}
    </div>
  );
}
