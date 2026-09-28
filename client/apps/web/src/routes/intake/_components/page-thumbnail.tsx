import type { CapturePage } from "@/lib/graphql/capture";
import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { useT } from "@trenova/shared/i18n/use-t";
import { apiUrl } from "@trenova/shared/lib/api-url";
import { cn } from "@trenova/shared/lib/utils";
import {
  ArrowRightToLineIcon,
  EyeIcon,
  FilePlusIcon,
  MoreHorizontalIcon,
  RotateCcwIcon,
  RotateCwIcon,
  ScissorsIcon,
  UndoDotIcon,
} from "lucide-react";
import { useState } from "react";

/** A US letter page, for a page that did not say how big it is. */
const LETTER_ASPECT = 8.5 / 11;

export type PageMoveTarget = { key: string; label: string };

export type PageActions = {
  preview: (page: CapturePage) => void;
  rotate: (pageId: string, quarterTurns: number) => void;
  /** Starts a new document after this page; absent on a document's last page. */
  splitAfter?: (pageId: string) => void;
  /** Sets the page aside; absent for a page already set aside. */
  leaveOut?: (pageId: string) => void;
  /** Makes a document of this one page; only for a page set aside. */
  newDocument?: (pageId: string) => void;
  moveTo: (pageId: string, target: string) => void;
};

function PageMarks({ page }: { page: CapturePage }) {
  const t = useT();

  if (page.status === "Failed") {
    return <Badge variant="danger">{t("Unreadable")}</Badge>;
  }
  if (page.isCoverSheet) {
    return <Badge variant="info">{t("Cover sheet")}</Badge>;
  }
  if (page.unrecognizedCoverSheet) {
    return (
      <Badge
        variant="warning"
        title={t("A cover sheet this organization did not issue, or one that expired")}
      >
        {t("Unknown sheet")}
      </Badge>
    );
  }
  if (page.patchCode !== "") {
    return <Badge variant="neutral">{t("Patch {0}", page.patchCode)}</Badge>;
  }
  if (page.isBlank) {
    return <Badge variant="neutral">{t("Blank")}</Badge>;
  }

  return null;
}

/**
 * One page, drawn as the scanner saw it and turned as the person turned it.
 * It drags between documents; everything dragging does is also in its menu,
 * so the stack can be rearranged from a keyboard.
 */
export function PageThumbnail({
  page,
  rotation,
  number,
  moveTargets,
  actions,
  disabled = false,
}: {
  page: CapturePage;
  rotation: number;
  /** The page's place in the stack as scanned, which is what a person counts by. */
  number: number;
  moveTargets: PageMoveTarget[];
  actions: PageActions;
  disabled?: boolean;
}) {
  const t = useT();
  const [broken, setBroken] = useState(false);
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
          onDoubleClick={() => actions.preview(page)}
          aria-label={t("Page {0}. Drag to move it, or open its menu.", number)}
          className={cn(
            "ui-focus-ring bg-card border-border flex aspect-[17/22] w-full items-center justify-center overflow-hidden rounded-md border",
            disabled ? "cursor-default" : "cursor-grab active:cursor-grabbing",
          )}
        >
          {broken || page.thumbnailPath === "" ? (
            <span className="text-foreground-subtle text-2xs px-2 text-center">
              {t("No preview")}
            </span>
          ) : (
            <img
              src={apiUrl(page.thumbnailPath)}
              alt=""
              loading="lazy"
              draggable={false}
              onError={() => setBroken(true)}
              className="max-h-full max-w-full object-contain transition-transform"
              style={{
                transform: `rotate(${rotation}deg) scale(${sideways ? Math.min(aspect, 1 / aspect) : 1})`,
              }}
            />
          )}
        </button>

        <div className="absolute top-1 right-1 opacity-0 transition-opacity group-focus-within/page:opacity-100 group-hover/page:opacity-100">
          <DropdownMenu>
            <DropdownMenuTrigger
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
            <DropdownMenuContent align="end" className="w-60">
              <DropdownMenuItem
                title={t("Preview")}
                startContent={<EyeIcon className="size-3.5" />}
                onClick={() => actions.preview(page)}
              />
              {!disabled && (
                <>
                  <DropdownMenuItem
                    title={t("Rotate right")}
                    startContent={<RotateCwIcon className="size-3.5" />}
                    onClick={() => actions.rotate(page.id, 1)}
                  />
                  <DropdownMenuItem
                    title={t("Rotate left")}
                    startContent={<RotateCcwIcon className="size-3.5" />}
                    onClick={() => actions.rotate(page.id, -1)}
                  />
                  <DropdownMenuSeparator />
                  {actions.splitAfter && (
                    <DropdownMenuItem
                      title={t("Start a new document after this page")}
                      startContent={<ScissorsIcon className="size-3.5" />}
                      onClick={() => actions.splitAfter?.(page.id)}
                    />
                  )}
                  {actions.newDocument && (
                    <DropdownMenuItem
                      title={t("Make it a document of its own")}
                      startContent={<FilePlusIcon className="size-3.5" />}
                      onClick={() => actions.newDocument?.(page.id)}
                    />
                  )}
                  {moveTargets.length > 0 && (
                    <DropdownMenuSub>
                      <DropdownMenuSubTrigger>
                        <ArrowRightToLineIcon className="size-3.5" />
                        {t("Move to")}
                      </DropdownMenuSubTrigger>
                      <DropdownMenuSubContent className="w-52">
                        {moveTargets.map((target) => (
                          <DropdownMenuItem
                            key={target.key}
                            title={target.label}
                            onClick={() => actions.moveTo(page.id, target.key)}
                          />
                        ))}
                      </DropdownMenuSubContent>
                    </DropdownMenuSub>
                  )}
                  {actions.leaveOut && (
                    <DropdownMenuItem
                      title={t("Set aside")}
                      description={t("Keeps the page out of every document")}
                      startContent={<UndoDotIcon className="size-3.5" />}
                      onClick={() => actions.leaveOut?.(page.id)}
                    />
                  )}
                </>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      <div className="flex min-h-5 items-center justify-between gap-1">
        <span className="text-foreground-subtle text-2xs tabular-nums">
          {t("Page {0}", number)}
        </span>
        <PageMarks page={page} />
      </div>
    </div>
  );
}
