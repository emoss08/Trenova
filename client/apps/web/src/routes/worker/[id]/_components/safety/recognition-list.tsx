import { RowActionsMenu } from "@/components/row-actions-menu";
import type { PreloadIntent } from "@/components/work-pane";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import { VirtualRows, type VirtualRow } from "@/components/virtual-rows";
import type { WorkerRecognitionRow } from "@/lib/graphql/worker-safety";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import {
  Award01Icon,
  EyeOffIcon,
  type IconComponent,
  PlusIcon,
  ThumbsUpIcon,
  Trash01Icon,
  Trophy01Icon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { RECOGNITION_KIND_LABELS, type RecognitionKind } from "@trenova/shared/types/worker-safety";
import { ListCapLine, SERVER_LIST_CAP } from "../list-more";
import { ShimmerLine } from "../tab-skeleton-parts";

const RECOGNITION_KIND_ICONS: Partial<Record<RecognitionKind, IconComponent>> = {
  CustomerPraise: ThumbsUpIcon,
  Performance: Trophy01Icon,
};

type RecognitionListProps = {
  /** The read of the worker's recognition: the newest, up to the server's cap. */
  recognitions: {
    data: WorkerRecognitionRow[] | undefined;
    isPending: boolean;
    isError: boolean;
    error: unknown;
    refetch: () => unknown;
  };
  workerName: string;
  canAdd: boolean;
  canDelete: boolean;
  /** The recognition being removed, so its menu holds until the request settles. */
  removingId: string | null;
  onAdd: () => void;
  /** Reads the recognition pane's code when the add button is pointed at or focused. */
  addIntent: PreloadIntent;
  onDelete: (recognition: WorkerRecognitionRow) => void;
};

/** The other half of the safety record: what somebody chose to write down in praise. */
export function RecognitionList({
  recognitions,
  workerName,
  canAdd,
  canDelete,
  removingId,
  onAdd,
  addIntent,
  onDelete,
}: RecognitionListProps) {
  const t = useT();
  const all = recognitions.data ?? [];
  const capped = all.length >= SERVER_LIST_CAP;
  const rows: VirtualRow[] = all.map((recognition, index) => ({
    key: recognition.id,
    render: () => (
      <RecognitionRow
        first={index === 0}
        recognition={recognition}
        workerName={workerName}
        canDelete={canDelete}
        removing={removingId !== null}
        onDelete={onDelete}
      />
    ),
  }));
  return (
    <SectionPanel
      title={t("Recognition")}
      count={capped ? undefined : all.length}
      action={
        canAdd ? (
          <Button size="sm" variant="quiet" {...addIntent} onClick={onAdd}>
            <PlusIcon className="size-3.5" />
            {t("Add recognition")}
          </Button>
        ) : null
      }
    >
      {recognitions.isPending ? (
        <div
          aria-busy
          aria-label={t("Loading the recognition")}
          className="flex flex-col gap-2 p-3"
        >
          <ShimmerLine className="w-3/5" />
          <ShimmerLine className="w-2/5" />
        </div>
      ) : recognitions.isError ? (
        <div className="p-3">
          <ErrorState
            error={recognitions.error}
            layout="compact"
            title={t("The recognition could not be loaded")}
            onRetry={() => void recognitions.refetch()}
          />
        </div>
      ) : rows.length === 0 ? (
        <SectionPanelQuiet>{t("No recognition yet.")}</SectionPanelQuiet>
      ) : (
        <VirtualRows
          rows={rows}
          mask="card"
          className="max-h-72"
          contentClassName="px-1.5 py-1"
          estimateSize={60}
          footer={<ListCapLine count={all.length} />}
          aria-label={t("Recognition")}
        />
      )}
    </SectionPanel>
  );
}

function RecognitionRow({
  first,
  recognition,
  workerName,
  canDelete,
  removing,
  onDelete,
}: {
  /** The rows are ruled between, as the list was: every row but the first has a rule above. */
  first: boolean;
  recognition: WorkerRecognitionRow;
  workerName: string;
  canDelete: boolean;
  removing: boolean;
  onDelete: (recognition: WorkerRecognitionRow) => void;
}) {
  const t = useT();
  const kind = recognition.kind as RecognitionKind;
  const Icon = RECOGNITION_KIND_ICONS[kind] ?? Award01Icon;
  const sub = [
    t(RECOGNITION_KIND_LABELS[kind] ?? kind),
    formatUnixDateMedium(recognition.occurredAt),
    recognition.visibleToWorker ? t("shared with {0}", workerName) : null,
    recognition.awardedBy?.name ? t("by {0}", recognition.awardedBy.name) : null,
  ].filter(Boolean);
  return (
    <div
      data-testid={`recognition-${recognition.id}`}
      className={cn(
        "grid grid-cols-[0.875rem_minmax(0,1fr)_auto] items-start gap-2.5 p-2",
        !first && "border-t border-border-subtle",
      )}
    >
      <Icon className="mt-0.5 size-3.5 text-muted-foreground" />
      <div className="min-w-0">
        <p className="flex items-center gap-1.5 font-medium">
          {recognition.title}
          {!recognition.visibleToWorker ? (
            <EyeOffIcon className="size-3 text-muted-foreground" aria-label={t("Kept internal")} />
          ) : null}
        </p>
        <p className="mt-px text-xs text-muted-foreground tabular-nums">{sub.join(" · ")}</p>
        {recognition.message ? (
          <p className="mt-1 text-xs text-pretty">{recognition.message}</p>
        ) : null}
      </div>
      <RowActionsMenu
        label={t("Options for {0}", recognition.title)}
        actions={
          canDelete
            ? [
                {
                  id: "remove",
                  label: t("Remove"),
                  icon: Trash01Icon,
                  destructive: true,
                  disabled: removing,
                  onSelect: () => onDelete(recognition),
                },
              ]
            : []
        }
      />
    </div>
  );
}
