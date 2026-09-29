import { SectionPanel } from "@/components/section-panel";
import type { CapturePage } from "@/lib/graphql/capture";
import { useDroppable } from "@dnd-kit/core";
import { SortableContext, rectSortingStrategy } from "@dnd-kit/sortable";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { LOOSE, pageNumber } from "./page-layout";
import { PageThumbnail, type PageMenu } from "./page-thumbnail";

/**
 * The pages in no document: the separators the splitter took out, blank
 * backs, and anything the person set aside. They stay on the stack, so a page
 * dropped by mistake is here to be dragged back, and they are deleted with
 * the rest of the stack's unfiled pages when its retention runs out.
 *
 * Before a stack is split every page is in no document, and none was set
 * aside: `arriving` shows them as the pages received so far instead.
 */
export function LoosePages({
  pageIds,
  pages,
  rotations,
  sequence,
  menu,
  onPreview,
  canEdit,
  arriving = false,
}: {
  pageIds: string[];
  pages: Map<string, CapturePage>;
  rotations: Readonly<Record<string, number>>;
  sequence: Readonly<Record<string, number>>;
  menu: PageMenu;
  onPreview: (pageId: string) => void;
  canEdit: boolean;
  arriving?: boolean;
}) {
  const t = useT();
  const { setNodeRef, isOver } = useDroppable({ id: LOOSE, disabled: !canEdit || arriving });

  return (
    <SectionPanel
      title={arriving ? t("Pages received") : t("Set aside")}
      count={pageIds.length}
      className="overflow-visible"
    >
      <div className="flex flex-col gap-2 p-3">
        <p className="text-foreground-subtle text-xs">
          {arriving
            ? pageIds.length === 0
              ? t("Pages appear here as they arrive")
              : t("Shown as they arrive. They are split into documents once every page is in.")
            : pageIds.length === 0
              ? t("Drag a page here to keep it out of every document")
              : t("Not in any document. Drag a page into one to file it.")}
        </p>
        <SortableContext items={pageIds} strategy={rectSortingStrategy}>
          <div
            ref={setNodeRef}
            className={cn(
              "flex min-h-16 flex-wrap gap-3 rounded-md p-2 transition-colors",
              !arriving && "border-border border border-dashed",
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
                  number={pageNumber({ sequence }, page)}
                  groupKey={LOOSE}
                  last={false}
                  menu={menu}
                  onPreview={onPreview}
                  disabled={!canEdit || arriving}
                />
              );
            })}
          </div>
        </SortableContext>
      </div>
    </SectionPanel>
  );
}
