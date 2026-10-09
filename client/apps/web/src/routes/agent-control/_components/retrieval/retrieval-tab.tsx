import { usePermission } from "@/hooks/use-permission";
import type { AIRetrievalSourceType } from "@/lib/graphql/ai-retrieval";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQueryState } from "nuqs";
import { lazy, useCallback } from "react";
import { REINDEX_PARAM, reindexParser } from "../../ai-control-tabs";
import { useAIControlNavigation } from "../../use-ai-control-navigation";
import { formatUsd } from "@/lib/ai-usage-format";
import { Ic } from "../kit/ic";
import { NovaSummary } from "../nova/nova-summary";
import { ReindexDialog } from "./reindex-dialog";
import { RetrievalFigures } from "./retrieval-figures";
import { RetrievalHero } from "./retrieval-hero";
import {
  RETRIEVAL_STALE_MS,
  SOURCE_LABEL,
  SOURCE_SETTING,
  retrievalRefetchInterval,
} from "./retrieval-model";
import { RetrievalNotice } from "./retrieval-notice";
import { RetrievalSettingsPanel } from "./retrieval-settings";
import { RetrievalSources } from "./retrieval-sources";
import { useRetrievalPatch } from "./use-retrieval-patch";

const FailedEntriesTable = lazy(() => import("./failed-entries-table"));

const FAILED_SECTION_ID = "rfail";

/**
 * Whether agents find memories, documents and inbound email by meaning, and what that
 * costs. Nova says where the index stands with the one control that changes it, the
 * figures and the sources say how far it has come, the settings say what indexing may
 * spend, and the table lists what failed.
 */
export default function RetrievalTab({ onOpenProviders }: { onOpenProviders: () => void }) {
  const t = useT();
  const navigate = useAIControlNavigation();
  const { allowed: canUpdate } = usePermission(Resource.AIProvider, Operation.Update);
  const [reindexing, setReindexing] = useQueryState(REINDEX_PARAM, reindexParser);
  const patch = useRetrievalPatch();

  const status = useQuery({
    ...queries.aiRetrieval.status(),
    staleTime: RETRIEVAL_STALE_MS,
    refetchInterval: (query) => retrievalRefetchInterval(query.state.data),
  });
  const providers = useQuery(queries.aiProvider.list());
  const routedTo =
    [...(providers.data ?? [])]
      .sort((a, b) => a.priority - b.priority)
      .find((provider) => provider.enabled && provider.tasks.includes("Embedding"))?.name ?? null;

  const showFailed = useCallback(() => {
    document
      .getElementById(FAILED_SECTION_ID)
      ?.scrollIntoView({ behavior: "smooth", block: "start" });
  }, []);
  const showFailures = useCallback(
    (sourceType: AIRetrievalSourceType) =>
      navigate({ tab: "retrieval", retrievalSource: sourceType }),
    [navigate],
  );
  const openReindex = useCallback(
    (sourceType: AIRetrievalSourceType) => void setReindexing(sourceType),
    [setReindexing],
  );
  const closeReindex = useCallback(() => void setReindexing(null), [setReindexing]);

  if (status.isError) {
    return (
      <div className="tabp">
        <div className="bnr d" role="alert">
          <Ic n="alert" s={14} />
          <span>{t("Where search by meaning stands could not be loaded. Try again shortly.")}</span>
        </div>
      </div>
    );
  }

  if (!status.data) {
    return (
      <div className="tabp">
        <NovaSummary context={t("Retrieval")} segments={undefined} loading onTarget={() => {}} />
      </div>
    );
  }

  const data = status.data;
  const pause = (paused: boolean) =>
    patch.mutate({
      patch: { paused },
      done: paused ? t("Indexing paused") : t("Indexing resumed"),
    });

  return (
    <div className="tabp">
      <RetrievalHero
        status={data}
        routedTo={routedTo}
        canUpdate={canUpdate}
        busy={patch.isPending}
        onPause={pause}
        onRoute={onOpenProviders}
        onShowFailed={showFailed}
      />
      <RetrievalNotice availability={data.availability} onOpenProviders={onOpenProviders} />
      <RetrievalFigures status={data} />
      <div className="ov">
        <div className="ov-m">
          <RetrievalSources
            status={data}
            canUpdate={canUpdate}
            busy={patch.isPending}
            onReindex={openReindex}
            onShowFailures={showFailures}
            onToggle={(sourceType, enabled) =>
              patch.mutate({
                patch: { [SOURCE_SETTING[sourceType]]: enabled },
                done: enabled
                  ? t("{0} will be indexed", t(SOURCE_LABEL[sourceType].label))
                  : t("{0} found by their words only", t(SOURCE_LABEL[sourceType].label)),
              })
            }
          />
          <section className="sec" id={FAILED_SECTION_ID}>
            <DataTableLazyComponent>
              <FailedEntriesTable />
            </DataTableLazyComponent>
          </section>
        </div>
        <aside className="ov-a">
          <RetrievalSettingsPanel
            settings={data.settings}
            canUpdate={canUpdate}
            busy={patch.isPending}
            onPause={pause}
            onBudget={(budgetUsd) =>
              patch.mutate({
                patch: { monthlyIndexingBudgetUsd: budgetUsd },
                done: t("Budget set to {0}", formatUsd(budgetUsd) ?? budgetUsd),
              })
            }
          />
        </aside>
      </div>
      <ReindexDialog
        sourceType={canUpdate ? reindexing : null}
        paused={data.settings.paused}
        onClose={closeReindex}
      />
    </div>
  );
}
