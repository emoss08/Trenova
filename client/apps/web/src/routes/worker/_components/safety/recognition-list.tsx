import { useT } from "@trenova/shared/i18n/use-t";
import { RowActionsMenu } from "@/components/row-actions-menu";
import type { WorkerRecognitionRow } from "@/lib/graphql/worker-safety";
import { Badge } from "@trenova/shared/components/ui/badge";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { RECOGNITION_KIND_LABELS, type RecognitionKind } from "@trenova/shared/types/worker-safety";
import { AwardIcon, EyeOffIcon, Trash2Icon } from "lucide-react";

type RecognitionListProps = {
  recognitions: WorkerRecognitionRow[];
  canDelete: boolean;
  busy: boolean;
  onDelete: (recognition: WorkerRecognitionRow) => void;
};

/**
 * Praise as small cards rather than a list: each one is a moment somebody
 * chose to write down, and a card reads as one. Visible entries reach the
 * driver's Dash as kudos.
 */
export function RecognitionList({ recognitions, canDelete, busy, onDelete }: RecognitionListProps) {
  const t = useT();

  return (
    <section className="flex flex-col gap-2">
      <div className="flex items-baseline justify-between gap-3">
        <h4 className="text-muted-foreground text-[11px] font-semibold uppercase">{t("Recognition")}</h4>
        <p className="text-muted-foreground truncate text-xs">
          {t("Visible entries show up as kudos in the driver's Dash.")}
        </p>
      </div>
      {recognitions.length === 0 ? (
        <p className="text-muted-foreground rounded-lg border border-dashed px-3 py-4 text-center text-xs">
          {t("Nothing recorded yet")}
        </p>
      ) : (
        <ul className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          {recognitions.map((recognition) => (
            <li
              key={recognition.id}
              data-testid={`recognition-${recognition.id}`}
              className="border-border/80 hover:border-border flex flex-col gap-2 rounded-lg border p-3 transition-colors"
            >
              <div className="flex items-start gap-3">
                <span className="bg-accent inline-flex size-7 shrink-0 items-center justify-center rounded-md">
                  <AwardIcon className="size-3.5" />
                </span>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-medium">{t(recognition.title)}</p>
                  <p className="text-muted-foreground text-xs">
                    {formatUnixDate(recognition.occurredAt)}
                    {recognition.awardedBy?.name ? ` · ${recognition.awardedBy.name}` : ""}
                  </p>
                </div>
                <RowActionsMenu
                  label={`Actions for ${recognition.title}`}
                  actions={
                    canDelete
                      ? [
                          {
                            id: "remove",
                            label: `Remove ${recognition.title}`,
                            icon: Trash2Icon,
                            disabled: busy,
                            destructive: true,
                            onSelect: () => onDelete(recognition),
                          },
                        ]
                      : []
                  }
                />
              </div>
              {recognition.message ? <p className="text-xs">{recognition.message}</p> : null}
              <div className="mt-auto flex flex-wrap items-center gap-1.5">
                <Badge variant="outline">
                  {RECOGNITION_KIND_LABELS[recognition.kind as RecognitionKind] ?? recognition.kind}
                </Badge>
                {!recognition.visibleToWorker ? (
                  <Badge variant="secondary">
                    <EyeOffIcon />
                    {t("Internal")}
                  </Badge>
                ) : null}
              </div>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
