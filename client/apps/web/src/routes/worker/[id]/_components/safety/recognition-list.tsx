import { RowActionsMenu } from "@/components/row-actions-menu";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import type { WorkerRecognitionRow } from "@/lib/graphql/worker-safety";
import {
  Award01Icon,
  EyeOffIcon,
  type IconComponent,
  ThumbsUpIcon,
  Trash01Icon,
  Trophy01Icon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { RECOGNITION_KIND_LABELS, type RecognitionKind } from "@trenova/shared/types/worker-safety";

const KIND_ICONS: Partial<Record<RecognitionKind, IconComponent>> = {
  CustomerPraise: ThumbsUpIcon,
  Performance: Trophy01Icon,
};

type RecognitionListProps = {
  recognitions: readonly WorkerRecognitionRow[];
  workerName: string;
  canAdd: boolean;
  canDelete: boolean;
  /** The recognition being removed, so its menu holds until the request settles. */
  removingId: string | null;
  onAdd: () => void;
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
  onDelete,
}: RecognitionListProps) {
  const t = useT();
  return (
    <SectionPanel
      title={t("Recognition")}
      count={recognitions.length}
      action={
        canAdd ? (
          <Button size="xs" variant="quiet" onClick={onAdd}>
            {t("Add")}
          </Button>
        ) : null
      }
    >
      {recognitions.length === 0 ? (
        <SectionPanelQuiet>{t("Nothing recorded yet.")}</SectionPanelQuiet>
      ) : (
        <ScrollArea viewportClassName="max-h-72" maskVariant="card">
          <ul className="flex flex-col divide-y divide-border-subtle px-1.5 py-1">
            {recognitions.map((recognition) => {
              const kind = recognition.kind as RecognitionKind;
              const Icon = KIND_ICONS[kind] ?? Award01Icon;
              const sub = [
                t(RECOGNITION_KIND_LABELS[kind] ?? kind),
                formatUnixDateMedium(recognition.occurredAt),
                recognition.visibleToWorker ? t("shared with {0}", workerName) : null,
                recognition.awardedBy?.name ? t("by {0}", recognition.awardedBy.name) : null,
              ].filter(Boolean);
              return (
                <li
                  key={recognition.id}
                  data-testid={`recognition-${recognition.id}`}
                  className="grid grid-cols-[0.875rem_minmax(0,1fr)_auto] items-start gap-2.5 p-2"
                >
                  <Icon className="mt-0.5 size-3.5 text-muted-foreground" />
                  <div className="min-w-0">
                    <p className="flex items-center gap-1.5 font-medium">
                      {recognition.title}
                      {!recognition.visibleToWorker ? (
                        <EyeOffIcon
                          className="size-3 text-muted-foreground"
                          aria-label={t("Kept internal")}
                        />
                      ) : null}
                    </p>
                    <p className="mt-px text-xs text-muted-foreground tabular-nums">
                      {sub.join(" · ")}
                    </p>
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
                              disabled: removingId !== null,
                              onSelect: () => onDelete(recognition),
                            },
                          ]
                        : []
                    }
                  />
                </li>
              );
            })}
          </ul>
        </ScrollArea>
      )}
    </SectionPanel>
  );
}
