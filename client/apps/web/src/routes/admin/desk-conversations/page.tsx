import { PageLayout } from "@/components/navigation/sidebar-layout";
import { useResetTableSearchParams } from "@/hooks/data-table/use-reset-table-search-params";
import { usePermission } from "@/hooks/use-permission";
import { useSearchState } from "@trenova/shared/hooks/use-search-state";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { Download01Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { lazy, useState } from "react";
import {
  CONVERSATION_SCOPE_PARAM,
  CONVERSATION_VIEW_PARAM,
  conversationScopeParser,
  conversationViewParser,
  scopeKey,
  type ConversationScope,
  type ConversationView,
} from "./_components/conversation-scope";
import { ConversationScopeBar } from "./_components/conversation-scope-bar";
import { ExportDialog } from "./_components/export-dialog";

const ConversationTable = lazy(() => import("./_components/conversation-table"));
const ExportTable = lazy(() => import("./_components/export-table"));

const pageParsers = {
  [CONVERSATION_SCOPE_PARAM]: conversationScopeParser,
  [CONVERSATION_VIEW_PARAM]: conversationViewParser,
};

/**
 * Every person's Desk conversations, for administrators who hold the
 * permission to read them: a list of who talked to which agent and when, each
 * opening read-only, and the transcripts taken out of Trenova one at a time,
 * picked from the list, or for a whole timeframe.
 */
export function DeskConversationsPage() {
  const t = useT();
  const [params, setParams] = useSearchState(pageParsers);
  const scope = params[CONVERSATION_SCOPE_PARAM];
  const view = params[CONVERSATION_VIEW_PARAM];
  const { allowed: canExport } = usePermission(Resource.DeskTranscript, Operation.Export);
  const { allowed: canPickAgent } = usePermission(Resource.AgentDefinition, Operation.Read);
  const [exportOpen, setExportOpen] = useState(false);
  const resetTable = useResetTableSearchParams();

  const changeScope = (next: ConversationScope) => {
    resetTable();
    void setParams({ [CONVERSATION_SCOPE_PARAM]: next });
  };
  const changeView = (next: ConversationView) => {
    resetTable();
    void setParams({ [CONVERSATION_VIEW_PARAM]: next });
  };

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Desk conversations"),
        description: t(
          "Read the Desk conversations of everyone in your organization, and download their transcripts. Every conversation you open or download is recorded in the audit log.",
        ),
        context: (
          <SegmentedControl<ConversationView>
            aria-label={t("Show")}
            value={view}
            onValueChange={changeView}
            items={[
              { value: "conversations", label: t("Conversations") },
              { value: "exports", label: t("Exports") },
            ]}
          />
        ),
        actions:
          canExport && view === "conversations" ? (
            <Button type="button" size="sm" onClick={() => setExportOpen(true)}>
              <Download01Icon className="size-3.5" />
              {t("Export transcripts")}
            </Button>
          ) : null,
      }}
    >
      {view === "exports" ? (
        <DataTableLazyComponent>
          <ExportTable canExport={canExport} />
        </DataTableLazyComponent>
      ) : (
        <div className="flex min-w-0 flex-col gap-3">
          <ConversationScopeBar scope={scope} onChange={changeScope} canPickAgent={canPickAgent} />
          <DataTableLazyComponent>
            <ConversationTable
              key={scopeKey(scope)}
              scope={scope}
              canExport={canExport}
              onScopeChange={changeScope}
              onExportStarted={() => changeView("exports")}
            />
          </DataTableLazyComponent>
        </div>
      )}
      {canExport ? (
        <ExportDialog
          open={exportOpen}
          onOpenChange={setExportOpen}
          scope={scope}
          canPickAgent={canPickAgent}
          onExportStarted={() => changeView("exports")}
        />
      ) : null}
    </PageLayout>
  );
}
