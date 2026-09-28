import type { CapturePage } from "@/lib/graphql/capture";
import { useDroppable } from "@dnd-kit/core";
import { SortableContext, rectSortingStrategy } from "@dnd-kit/sortable";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { LOOSE } from "./page-layout";
import { PageThumbnail, type PageActions, type PageMoveTarget } from "./page-thumbnail";

/**
 * The pages in no document: the separators the splitter took out, blank
 * backs, and anything the person set aside. They stay on the stack, so a page
 * dropped by mistake is here to be dragged back, and they are deleted with
 * the rest of the stack's unfiled pages when its retention runs out.
 */
export function LoosePages({
  pageIds,
  pages,
  rotations,
  moveTargets,
  pageActions,
  canEdit,
}: {
  pageIds: string[];
  pages: Map<string, CapturePage>;
  rotations: Readonly<Record<string, number>>;
  moveTargets: PageMoveTarget[];
  pageActions: Omit<PageActions, "leaveOut" | "splitAfter">;
  canEdit: boolean;
}) {
  const t = useT();
  const { setNodeRef, isOver } = useDroppable({ id: LOOSE, disabled: !canEdit });

  return (
    <section
      aria-label={t("Set aside")}
      className="border-border flex flex-col gap-2 rounded-lg border border-dashed p-3"
    >
      <div className="flex items-baseline justify-between gap-2">
        <h3 className="text-sm font-semibold">{t("Set aside")}</h3>
        <span className="text-foreground-subtle text-xs">
          {pageIds.length === 0
            ? t("Drag a page here to keep it out of every document")
            : t("Not in any document. Drag a page into one to file it.")}
        </span>
      </div>
      <SortableContext items={pageIds} strategy={rectSortingStrategy}>
        <div
          ref={setNodeRef}
          className={cn(
            "flex min-h-16 flex-wrap gap-3 rounded-md p-1 transition-colors",
            isOver && "bg-surface-selected",
          )}
        >
          {pageIds.map((pageId) => {
            const page = pages.get(pageId);
            if (page === undefined) {
              return null;
            }
            return (
              <PageThumbnail
                key={pageId}
                page={page}
                rotation={rotations[pageId] ?? 0}
                number={page.sequence}
                moveTargets={moveTargets}
                disabled={!canEdit}
                actions={pageActions}
              />
            );
          })}
        </div>
      </SortableContext>
    </section>
  );
}
