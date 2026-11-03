import { DotsHorizontalIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatCompactAge } from "@trenova/shared/lib/date";
import { formatFileSize } from "@trenova/shared/lib/utils";
import type { Document } from "@trenova/shared/types/document";

/**
 * The chosen document's line under the board: which file it is, what it needs,
 * and what can be done with it. The one fix it needs is a button; the rest wait in
 * a menu.
 */
export function DocumentDetail({
  doc,
  attention,
  busy,
  inPacket,
  onView,
  onRecordSignature,
  onMove,
  onReplace,
  onDelete,
}: {
  doc: Document;
  attention?: string;
  busy: boolean;
  /** Whether it goes out with the invoice; absent when the item sends no packet. */
  inPacket?: boolean;
  onView: () => void;
  onRecordSignature?: () => void;
  onMove?: () => void;
  onReplace?: () => void;
  onDelete?: () => void;
}) {
  const t = useT();
  const now = useNow("minute");
  const name = doc.documentType?.name ?? doc.originalName;
  const hasMenu = Boolean(onMove || onReplace || onDelete);

  return (
    <div className="border-border flex flex-wrap items-center gap-x-3 gap-y-2 border-t pt-3.5">
      <div className="flex min-w-44 flex-1 flex-col gap-0.5">
        <span className="flex min-w-0 items-center gap-2">
          <span className="text-foreground truncate font-medium">{name}</span>
          {attention ? (
            <span className="text-foreground flex shrink-0 items-center gap-1.5 text-xs">
              <span aria-hidden className="bg-warning size-1.5 rounded-full" />
              {attention}
            </span>
          ) : null}
        </span>
        <span className="text-foreground-muted truncate text-xs">
          {t(
            "{0} · {1} · {2} ago",
            doc.originalName,
            formatFileSize(doc.fileSize),
            formatCompactAge(now - doc.createdAt),
          )}
        </span>
      </div>
      <div className="flex items-center gap-1">
        {onRecordSignature ? (
          <Button size="xs" variant="outline" onClick={onRecordSignature}>
            {t("Record signature")}
          </Button>
        ) : null}
        <Button size="xs" variant="outline" onClick={onView}>
          {t("View")}
        </Button>
        {hasMenu ? (
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button
                  size="icon-xs"
                  variant="ghost"
                  isLoading={busy}
                  aria-label={t("More for {0}", name)}
                >
                  <DotsHorizontalIcon className="size-3.5" />
                </Button>
              }
            />
            <DropdownMenuContent align="end" className="w-48">
              {onMove ? (
                <DropdownMenuItem
                  title={inPacket ? t("Leave out of packet") : t("Send with invoice")}
                  onClick={onMove}
                />
              ) : null}
              {onMove && (onReplace || onDelete) ? <DropdownMenuSeparator /> : null}
              {onReplace ? (
                <DropdownMenuItem title={t("Upload a new version")} onClick={onReplace} />
              ) : null}
              {onDelete ? (
                <DropdownMenuItem title={t("Delete")} color="danger" onClick={onDelete} />
              ) : null}
            </DropdownMenuContent>
          </DropdownMenu>
        ) : null}
      </div>
    </div>
  );
}
