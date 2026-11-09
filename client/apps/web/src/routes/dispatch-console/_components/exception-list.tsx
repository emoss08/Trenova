import { VirtualRows, type VirtualRow } from "@/components/virtual-rows";
import type { WatchtowerItem } from "@/lib/graphql/watchtower";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { LinkExternal01Icon } from "@trenova/shared/components/icons";
import { EmptyTable, EmptyTableSketch } from "@trenova/shared/components/ui/empty-table";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatMinutesSpan } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { Link } from "react-router";
import { MoveQueueRowsSkeleton, QUEUE_GRID_CLASS } from "./console-skeletons";
import { formatWhen } from "./format-when";
import type { QueueRead } from "./move-queue";

const EXCEPTION_COLUMNS = [{ label: "Due" }, { label: "Exception" }, { label: "Owner" }] as const;
const WATCHTOWER_PATH = "/desk/needs?lens=watchtower";
const EXCEPTION_ROW_ESTIMATE_PX = 52;

function DueCell({ item, nowSeconds }: { item: WatchtowerItem; nowSeconds: number }) {
  const t = useT();

  if (item.overdue && item.dueAt) {
    return (
      <>
        <span className="text-danger-foreground block truncate font-medium">
          {t(
            "Overdue {0}",
            formatMinutesSpan(Math.max(1, Math.round((nowSeconds - item.dueAt) / 60))),
          )}
        </span>
        <span className="text-muted-foreground block truncate text-2xs">
          {item.dueLabel || t("was due {0}", formatWhen(item.dueAt, nowSeconds))}
        </span>
      </>
    );
  }
  if (item.dueAt) {
    return (
      <>
        <span className="block truncate font-medium">
          {t("Due {0}", formatWhen(item.dueAt, nowSeconds))}
        </span>
        <span className="text-muted-foreground block truncate text-2xs">
          {item.dueLabel || item.kindLabel}
        </span>
      </>
    );
  }
  return (
    <>
      <span className="block truncate font-medium">
        {t("Since {0}", formatWhen(item.occurredAt, nowSeconds))}
      </span>
      <span className="text-muted-foreground block truncate text-2xs">{item.kindLabel}</span>
    </>
  );
}

function ExceptionRow({ item, nowSeconds }: { item: WatchtowerItem; nowSeconds: number }) {
  const t = useT();

  return (
    <Link
      to={item.path || WATCHTOWER_PATH}
      data-glide-target
      className={cn(
        QUEUE_GRID_CLASS,
        "ui-focus-ring relative mx-1.5 items-center rounded-md px-2 py-1.75 text-xs",
      )}
    >
      <div className="min-w-0">
        <DueCell item={item} nowSeconds={nowSeconds} />
      </div>
      <div className="min-w-0">
        <div className="flex min-w-0 items-center gap-2">
          {item.record?.label && !item.title.includes(item.record.label) ? (
            <span className="shrink-0 font-mono font-medium">{item.record.label}</span>
          ) : null}
          <span className="truncate">{item.title}</span>
        </div>
        <div className="text-muted-foreground truncate text-2xs">{item.summary}</div>
      </div>
      <div className="min-w-0">
        <div className="flex min-w-0 items-center gap-1.5">
          <span className="truncate">{item.assignee?.name ?? t("Nobody yet")}</span>
          <LinkExternal01Icon
            className="text-muted-foreground ml-auto size-3 shrink-0"
            aria-hidden
          />
        </div>
        <div className="text-muted-foreground truncate text-2xs">{item.kindLabel}</div>
      </div>
    </Link>
  );
}

/**
 * The exceptions a dispatcher still owns once a move is covered, read from the
 * Watchtower: the console keeps no list of its own, so working one here or there is the
 * same item. Each row opens the record it is about.
 */
export function ExceptionList({
  items,
  hasMore,
  read,
}: {
  items: readonly WatchtowerItem[];
  hasMore: boolean;
  read: QueueRead;
}) {
  const t = useT();
  const nowSeconds = useNow("minute");

  if (read.isPending) {
    return <MoveQueueRowsSkeleton />;
  }
  if (read.isError) {
    return (
      <ErrorState
        error={read.error}
        layout="sheet"
        sketch={<EmptyTableSketch columns={EXCEPTION_COLUMNS} />}
        title={t("Exceptions didn't load")}
        description={t(
          "This isn't an empty list: covered moves may still be slipping. The Watchtower has them too.",
        )}
        onRetry={read.refetch}
      />
    );
  }
  if (items.length === 0) {
    return (
      <EmptyTable
        columns={EXCEPTION_COLUMNS}
        title={t("No open exceptions")}
        description={t(
          "No covered move is running late, missed a stop, gone without a check call or run out of carriers.",
        )}
      />
    );
  }

  const rows: VirtualRow[] = items.map((item) => ({
    key: item.id,
    render: () => <ExceptionRow item={item} nowSeconds={nowSeconds} />,
  }));

  return (
    <VirtualRows
      rows={rows}
      estimateSize={EXCEPTION_ROW_ESTIMATE_PX}
      initialHeight={640}
      aria-label={t("Dispatch exceptions")}
      hoverGlide
      mask="card"
      maskHeight={18}
      className="min-h-0 flex-1"
      rowClassName="pt-px"
      footer={
        hasMore ? (
          <div className="px-3.5 py-3 text-xs">
            <Link to={WATCHTOWER_PATH} className="text-brand ui-focus-ring rounded-sm">
              {t("More in the Watchtower")}
            </Link>
          </div>
        ) : (
          <div className="h-2" aria-hidden />
        )
      }
    />
  );
}
