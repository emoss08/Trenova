import type { CapturePage } from "@/lib/graphql/capture";
import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenuTrigger,
  type DropdownMenuHandle,
} from "@trenova/shared/components/ui/dropdown-menu";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { apiUrl } from "@trenova/shared/lib/api-url";
import { cn } from "@trenova/shared/lib/utils";
import type { BadgeTone } from "@trenova/shared/types/badge";
import { MoreHorizontalIcon } from "lucide-react";
import { memo, useState } from "react";

/** A US letter page, for a page that did not say how big it is. */
const LETTER_ASPECT = 8.5 / 11;

/** Which page a page menu was opened on, and where that page sits. */
export type PageMenuPayload = {
  pageId: string;
  /** The document it is in, or LOOSE for a page set aside. */
  groupKey: string;
  /** Whether it is its document's last page, which nothing can be split after. */
  last: boolean;
};

export type PageMenu = DropdownMenuHandle<PageMenuPayload>;

function pageMark(
  t: TranslateFn,
  page: CapturePage,
): { variant: BadgeTone; text: string; title?: string } | null {
  if (page.status === "Failed") {
    return { variant: "danger", text: t("Unreadable") };
  }
  if (page.isCoverSheet) {
    return { variant: "info", text: t("Cover sheet") };
  }
  if (page.unrecognizedCoverSheet) {
    return {
      variant: "warning",
      text: t("Unknown sheet"),
      title: t("A cover sheet this organization did not issue, or one that expired"),
    };
  }
  if (page.patchCode !== "") {
    return { variant: "neutral", text: t("Patch {0}", page.patchCode) };
  }
  if (page.isBlank) {
    return { variant: "neutral", text: t("Blank") };
  }

  return null;
}

/** What the page is, when it is something other than a page of a document. */
function PageMarks({ page }: { page: CapturePage }) {
  const t = useT();
  const mark = pageMark(t, page);
  if (mark === null) {
    return null;
  }

  return (
    <Badge
      variant={mark.variant}
      title={mark.title ?? mark.text}
      className="max-w-full min-w-0 shrink justify-start"
    >
      <span className="truncate">{mark.text}</span>
    </Badge>
  );
}

/**
 * One page, drawn as the scanner saw it and turned as the person turned it.
 * It drags between documents; everything dragging does is also in its menu,
 * so the stack can be rearranged from a keyboard.
 */
export const PageThumbnail = memo(function PageThumbnail({
  page,
  rotation,
  number,
  groupKey,
  last,
  menu,
  onPreview,
  disabled = false,
}: {
  page: CapturePage;
  rotation: number;
  /** The page's place in the stack as scanned, which is what a person counts by. */
  number: number;
  groupKey: string;
  last: boolean;
  /** The one menu every page of the stack opens. */
  menu: PageMenu;
  onPreview: (pageId: string) => void;
  disabled?: boolean;
}) {
  const t = useT();
  const [brokenPath, setBrokenPath] = useState<string | null>(null);
  const showImage = page.thumbnailPath !== "" && brokenPath !== page.thumbnailPath;
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: page.id,
    disabled,
  });

  const aspect =
    page.widthPx > 0 && page.heightPx > 0 ? page.widthPx / page.heightPx : LETTER_ASPECT;
  const sideways = rotation === 90 || rotation === 270;

  return (
    <div
      ref={setNodeRef}
      style={{ transform: CSS.Transform.toString(transform), transition }}
      className={cn("group/page flex w-24 shrink-0 flex-col gap-1", isDragging && "opacity-40")}
    >
      <div className="relative">
        <button
          type="button"
          {...attributes}
          {...listeners}
          onDoubleClick={() => onPreview(page.id)}
          aria-label={t("Page {0}. Drag to move it, or open its menu.", number)}
          className={cn(
            "ui-focus-ring bg-card border-border flex aspect-[17/22] w-full items-center justify-center overflow-hidden rounded-md border",
            disabled ? "cursor-default" : "cursor-grab active:cursor-grabbing",
          )}
        >
          {!showImage ? (
            <span className="text-foreground-subtle text-2xs px-2 text-center">
              {page.status === "Received" ? t("Reading page") : t("No preview")}
            </span>
          ) : (
            <img
              src={apiUrl(page.thumbnailPath)}
              alt=""
              loading="lazy"
              draggable={false}
              onError={() => setBrokenPath(page.thumbnailPath)}
              className="max-h-full max-w-full object-contain transition-transform"
              style={{
                transform: `rotate(${rotation}deg) scale(${sideways ? Math.min(aspect, 1 / aspect) : 1})`,
              }}
            />
          )}
        </button>

        <div className="absolute top-1 right-1 transition-opacity pointer-fine:opacity-0 pointer-fine:group-focus-within/page:opacity-100 pointer-fine:group-hover/page:opacity-100 pointer-fine:has-[[data-popup-open]]:opacity-100">
          <DropdownMenuTrigger
            handle={menu}
            payload={{ pageId: page.id, groupKey, last }}
            render={
              <Button
                type="button"
                size="icon-xs"
                variant="outline"
                aria-label={t("Page {0} actions", number)}
              >
                <MoreHorizontalIcon className="size-3.5" />
              </Button>
            }
          />
        </div>
      </div>

      <div className="flex min-w-0 flex-col items-start gap-1">
        <span className="text-foreground-subtle text-2xs tabular-nums">
          {t("Page {0}", number)}
        </span>
        <PageMarks page={page} />
      </div>
    </div>
  );
});
