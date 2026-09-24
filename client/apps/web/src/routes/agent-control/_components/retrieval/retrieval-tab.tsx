import { usePermission } from "@/hooks/use-permission";
import type { AIRetrievalSourceType } from "@/lib/graphql/ai-retrieval";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { CircleAlertIcon } from "lucide-react";
import { lazy, useCallback, useRef, useState } from "react";
import { useAIControlNavigation } from "../../use-ai-control-navigation";
import { ReindexDialog } from "./reindex-dialog";
import { RetrievalFigures } from "./retrieval-figures";
import { RETRIEVAL_STALE_MS, retrievalRefetchInterval } from "./retrieval-model";
import { RetrievalNotice } from "./retrieval-notice";
import { RetrievalSettingsPanel } from "./retrieval-settings";
import { RetrievalSources } from "./retrieval-sources";

const FailedEntriesTable = lazy(() => import("./failed-entries-table"));

/**
 * Whether agents find memories, documents and inbound email by meaning, and
 * what that costs. The notice says what stops it and how to fix that, the
 * figures and the sources say how far the index has come, the settings say
 * what is indexed and what it may spend, and the table lists what failed.
 */
export default function RetrievalTab({ onOpenProviders }: { onOpenProviders: () => void }) {
  const t = useT();
  const navigate = useAIControlNavigation();
  const { allowed: canUpdate } = usePermission(Resource.AIProvider, Operation.Update);
  const settingsRef = useRef<HTMLDivElement>(null);
  const [reindexing, setReindexing] = useState<AIRetrievalSourceType | null>(null);

  const status = useQuery({
    ...queries.aiRetrieval.status(),
    staleTime: RETRIEVAL_STALE_MS,
    refetchInterval: (query) => retrievalRefetchInterval(query.state.data),
  });

  const openSettings = useCallback(() => {
    const target = settingsRef.current;
    if (!target) {
      return;
    }
    target.scrollIntoView({ block: "start" });
    target.focus({ preventScroll: true });
  }, []);

  const showFailures = useCallback(
    (sourceType: AIRetrievalSourceType) =>
      navigate({ tab: "retrieval", retrievalSource: sourceType }),
    [navigate],
  );
  const closeReindex = useCallback(() => setReindexing(null), []);

  if (status.isError) {
    return (
      <Alert variant="destructive" size="sm">
        <CircleAlertIcon />
        <AlertDescription>
          {t("Where search by meaning stands could not be loaded. Try again shortly.")}
        </AlertDescription>
      </Alert>
    );
  }

  if (!status.data) {
    return (
      <div className="flex min-w-0 flex-col gap-4" aria-busy>
        <Skeleton className="h-16" />
        <div className="grid gap-4 lg:grid-cols-2">
          <Skeleton className="h-64" />
          <Skeleton className="h-64" />
        </div>
      </div>
    );
  }

  const data = status.data;

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <RetrievalNotice
        availability={data.availability}
        onOpenProviders={onOpenProviders}
        onOpenSettings={openSettings}
      />
      <RetrievalFigures status={data} />
      <div className="grid min-w-0 items-start gap-4 lg:grid-cols-2">
        <RetrievalSources
          status={data}
          canUpdate={canUpdate}
          onReindex={setReindexing}
          onShowFailures={showFailures}
        />
        <div
          ref={settingsRef}
          tabIndex={-1}
          className="ui-focus-ring min-w-0 rounded-lg outline-none"
        >
          <RetrievalSettingsPanel settings={data.settings} canUpdate={canUpdate} />
        </div>
      </div>
      <DataTableLazyComponent>
        <FailedEntriesTable />
      </DataTableLazyComponent>
      <ReindexDialog sourceType={reindexing} paused={data.settings.paused} onClose={closeReindex} />
    </div>
  );
}
