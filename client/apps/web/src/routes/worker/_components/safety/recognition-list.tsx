import type { WorkerRecognitionRow } from "@/lib/graphql/worker-safety";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { RECOGNITION_KIND_LABELS, type RecognitionKind } from "@trenova/shared/types/worker-safety";
import { AwardIcon, EyeOffIcon, Trash2Icon } from "lucide-react";

type RecognitionListProps = {
  recognitions: WorkerRecognitionRow[];
  canDelete: boolean;
  busy: boolean;
  onDelete: (recognition: WorkerRecognitionRow) => void;
};

export function RecognitionList({ recognitions, canDelete, busy, onDelete }: RecognitionListProps) {
  return (
    <section className="flex flex-col gap-2">
      <div>
        <h3 className="text-sm font-semibold">Recognition</h3>
        <p className="text-muted-foreground text-xs">
          Praise and milestones. Visible entries show up as kudos in the driver&apos;s Dash.
        </p>
      </div>
      {recognitions.length === 0 ? (
        <p className="text-muted-foreground border-border rounded-lg border border-dashed px-3 py-4 text-center text-xs">
          Nothing recorded yet
        </p>
      ) : (
        <ul className="divide-border border-border divide-y rounded-lg border">
          {recognitions.map((recognition) => (
            <li
              key={recognition.id}
              data-testid={`recognition-${recognition.id}`}
              className="flex items-start gap-3 px-3 py-2.5"
            >
              <AwardIcon className="mt-0.5 size-4 shrink-0 text-amber-500" />
              <div className="min-w-0 flex-1">
                <p className="flex flex-wrap items-center gap-2 text-sm font-medium">
                  {recognition.title}
                  <Badge variant="outline" className="px-1.5 py-0 text-[10px]">
                    {RECOGNITION_KIND_LABELS[recognition.kind as RecognitionKind] ??
                      recognition.kind}
                  </Badge>
                  {!recognition.visibleToWorker ? (
                    <span className="text-muted-foreground flex items-center gap-1 text-[10px]">
                      <EyeOffIcon className="size-3" />
                      Internal
                    </span>
                  ) : null}
                </p>
                {recognition.message ? <p className="text-xs">{recognition.message}</p> : null}
                <p className="text-muted-foreground text-[11px]">
                  {formatUnixDate(recognition.occurredAt)}
                  {recognition.awardedBy?.name ? ` · ${recognition.awardedBy.name}` : ""}
                </p>
              </div>
              {canDelete ? (
                <Button
                  size="sm"
                  variant="ghost"
                  className="text-destructive size-7"
                  disabled={busy}
                  aria-label={`Remove ${recognition.title}`}
                  onClick={() => onDelete(recognition)}
                >
                  <Trash2Icon className="size-3.5" />
                </Button>
              ) : null}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
