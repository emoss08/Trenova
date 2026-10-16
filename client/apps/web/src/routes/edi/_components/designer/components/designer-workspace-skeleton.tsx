import { useT } from "@trenova/shared/i18n/use-t";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
import { useEDIDesignerUrlState } from "../hooks/use-edi-designer-url-state";

function SkeletonLine({ className }: { className?: string }) {
  return <Skeleton className={cn("h-3 rounded-sm", className)} />;
}

function SkeletonToolbar({ count = 3 }: { count?: number }) {
  return (
    <div className="flex items-center justify-between gap-2 border-b px-3 py-2">
      <div className="space-y-2">
        <SkeletonLine className="w-40" />
        <SkeletonLine className="w-56" />
      </div>
      <div className="flex shrink-0 gap-2">
        {Array.from({ length: count }).map((_, index) => (
          <Skeleton key={index} className="h-8 w-24 rounded-md" />
        ))}
      </div>
    </div>
  );
}

export function DesignerPanelSkeleton() {
  return (
    <div className="grid h-full min-h-0 grid-rows-[auto_minmax(0,1fr)]">
      <SkeletonToolbar count={2} />
      <div className="grid min-h-0 grid-cols-[minmax(0,1fr)_320px]">
        <div className="space-y-3 p-3">
          {Array.from({ length: 8 }).map((_, index) => (
            <Skeleton key={index} className="h-9 rounded-md" />
          ))}
        </div>
        <div className="space-y-3 border-l p-3">
          <SkeletonLine className="w-32" />
          {Array.from({ length: 7 }).map((_, index) => (
            <Skeleton key={index} className="h-10 rounded-md" />
          ))}
        </div>
      </div>
    </div>
  );
}

export function DesignerAsideSkeleton() {
  return (
    <aside className="bg-background flex h-full min-h-0 flex-col overflow-hidden rounded-md border">
      <div className="flex h-11 items-center gap-2 border-b px-3">
        <Skeleton className="size-4 rounded-sm" />
        <SkeletonLine className="w-24" />
      </div>
      <div className="space-y-3 border-b p-3">
        <Skeleton className="h-8 rounded-md" />
        <Skeleton className="h-8 rounded-md" />
        <div className="grid grid-cols-2 gap-2">
          <Skeleton className="h-8 rounded-md" />
          <Skeleton className="h-8 rounded-md" />
        </div>
      </div>
      <div className="min-h-0 flex-1 overflow-hidden">
        {Array.from({ length: 8 }).map((_, index) => (
          <div key={index} className="space-y-2 border-b px-3 py-3">
            <SkeletonLine className="w-4/5" />
            <SkeletonLine className="w-2/3" />
          </div>
        ))}
      </div>
      <div className="space-y-2 border-t p-3">
        <SkeletonLine className="w-36" />
        <Skeleton className="h-8 rounded-md" />
        <Skeleton className="h-8 rounded-md" />
      </div>
    </aside>
  );
}

function DesignerTabStripSkeleton({
  widths,
  icons = false,
  className,
}: {
  widths: readonly string[];
  icons?: boolean;
  className?: string;
}) {
  return (
    <div className={cn("flex items-center gap-x-0.5 py-1", className)}>
      {widths.map((width, index) => (
        <span key={index} className="flex h-9 items-center gap-1.5 px-2.5 sm:h-8">
          {icons ? <Skeleton className="size-4 rounded-sm" /> : null}
          <SkeletonLine className={width} />
        </span>
      ))}
    </div>
  );
}

const DESIGNER_TAB_WIDTHS = ["w-16", "w-40"] as const;
const EDITOR_TAB_WIDTHS = ["w-14", "w-12", "w-16", "w-12"] as const;
const VERSION_ROW_COUNT = 3;
const SEGMENT_ROW_COUNT = 8;
const DIAGNOSTIC_ROW_COUNT = 5;
const PROFILE_FIELD_COUNT = 3;
const ARCHIVE_ROW_COUNT = 8;

function PanelHeaderSkeleton({ titleWidth }: { titleWidth: string }) {
  return (
    <div className="flex h-11 items-center gap-2 border-b px-3">
      <Skeleton className="size-4 rounded-sm" />
      <SkeletonLine className={titleWidth} />
    </div>
  );
}

function FieldSkeleton() {
  return (
    <div className="flex flex-col gap-0.5">
      <span className="flex h-4 items-center">
        <Skeleton className="h-2.5 w-16" />
      </span>
      <Skeleton className="h-7 rounded-md" />
    </div>
  );
}

function VersionAndSegmentRailSkeleton() {
  return (
    <div className="grid min-h-0 grid-rows-[180px_minmax(0,1fr)] overflow-hidden border-r">
      <div className="min-h-0 overflow-hidden border-b">
        <div className="space-y-1 border-b px-3 py-2">
          <SkeletonLine className="w-16" />
          <SkeletonLine className="w-20" />
        </div>
        {Array.from({ length: VERSION_ROW_COUNT }, (_, index) => (
          <div key={index} className="flex items-center justify-between gap-2 border-b px-3 py-2">
            <SkeletonLine className="w-6" />
            <Skeleton className="h-5 w-16 rounded-badge" />
          </div>
        ))}
      </div>
      <div className="min-h-0 overflow-hidden">
        <div className="space-y-1 border-b px-3 py-2">
          <SkeletonLine className="w-24" />
          <SkeletonLine className="w-16" />
        </div>
        {Array.from({ length: SEGMENT_ROW_COUNT }, (_, index) => (
          <div key={index} className="space-y-1.5 border-b px-3 py-2">
            <SkeletonLine className="w-10" />
            <SkeletonLine className="w-36" />
          </div>
        ))}
      </div>
    </div>
  );
}

/**
 * The templates tab: the template list, the editor with its header, the
 * version and segment rail, the editor's own tabs and footer, and the
 * diagnostics column, in the tab's three-column grid.
 */
export function DesignerTemplatesSkeleton() {
  const t = useT();

  return (
    <div
      className="grid h-full min-h-0 grid-cols-[310px_minmax(0,1fr)_250px] gap-3 overflow-hidden"
      aria-busy
      aria-label={t("Loading templates")}
    >
      <div className="contents" aria-hidden>
        <DesignerAsideSkeleton />
        <div className="bg-background flex min-h-0 flex-col overflow-hidden rounded-md border">
          <SkeletonToolbar count={4} />
          <div className="grid min-h-0 flex-1 grid-cols-[250px_minmax(0,1fr)] overflow-hidden max-md:grid-cols-1">
            <VersionAndSegmentRailSkeleton />
            <div className="grid min-h-0 grid-rows-[auto_minmax(0,1fr)] overflow-hidden">
              <div className="flex items-center justify-between gap-2 border-b px-1">
                <DesignerTabStripSkeleton widths={EDITOR_TAB_WIDTHS} />
                <div className="flex items-center gap-2 pr-2">
                  <Skeleton className="h-6 w-24 rounded-md" />
                  <Skeleton className="h-6 w-24 rounded-md" />
                </div>
              </div>
              <DesignerPanelSkeleton />
            </div>
          </div>
          <div className="flex items-center justify-between gap-2 border-t px-3 py-2">
            <SkeletonLine className="w-96 max-w-[55%]" />
            <Skeleton className="h-7 w-32 rounded-md" />
          </div>
        </div>
        <div className="bg-background flex h-full min-h-0 flex-col overflow-hidden rounded-md border max-xl:hidden">
          <PanelHeaderSkeleton titleWidth="w-20" />
          <div className="space-y-3 p-3">
            {Array.from({ length: DIAGNOSTIC_ROW_COUNT }, (_, index) => (
              <div key={index} className="space-y-2 rounded-md border p-2">
                <SkeletonLine className="w-20" />
                <SkeletonLine className="w-full" />
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

/**
 * The document preview and archive tab: the document profile form beside the
 * archive of rendered documents.
 */
export function DesignerDocumentsSkeleton() {
  const t = useT();

  return (
    <div
      className="grid h-full min-h-0 grid-cols-[360px_minmax(0,1fr)] gap-3 overflow-hidden"
      aria-busy
      aria-label={t("Loading document preview and archive")}
    >
      <div className="contents" aria-hidden>
        <div className="bg-background flex min-h-0 flex-col overflow-hidden rounded-md border">
          <PanelHeaderSkeleton titleWidth="w-28" />
          <div className="flex flex-col gap-3 p-3">
            {Array.from({ length: PROFILE_FIELD_COUNT }, (_, index) => (
              <FieldSkeleton key={index} />
            ))}
            <div className="grid grid-cols-2 gap-2">
              <FieldSkeleton />
              <FieldSkeleton />
            </div>
            <FieldSkeleton />
            <FieldSkeleton />
            <Skeleton className="h-32 rounded-md" />
            <Skeleton className="h-8 rounded-md" />
          </div>
        </div>
        <div className="bg-background min-h-0 overflow-hidden rounded-md border">
          <PanelHeaderSkeleton titleWidth="w-36" />
          {Array.from({ length: ARCHIVE_ROW_COUNT }, (_, index) => (
            <div key={index} className="flex items-center justify-between gap-3 border-b px-3 py-2">
              <div className="space-y-2">
                <SkeletonLine className="w-40" />
                <SkeletonLine className="w-64" />
              </div>
              <Skeleton className="h-7 w-20 rounded-md" />
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

/**
 * The whole designer while its code arrives: the workspace's underline tabs
 * and, under them, the tab the address names, in the workspace's own grid.
 */
export function DesignerWorkspaceSkeleton() {
  const t = useT();
  const [{ designerTab }] = useEDIDesignerUrlState();

  return (
    <div
      className="grid h-[calc(100vh-11rem)] min-h-0 grid-rows-[auto_minmax(0,1fr)] gap-0 overflow-hidden"
      aria-busy
      aria-label={t("Loading EDI designer workspace")}
    >
      <div className="contents" aria-hidden>
        <DesignerTabStripSkeleton
          widths={DESIGNER_TAB_WIDTHS}
          icons
          className="w-full justify-start border-b px-1"
        />
        <div className="min-h-0 overflow-hidden px-3 pt-3">
          {designerTab === "documents" ? (
            <DesignerDocumentsSkeleton />
          ) : (
            <DesignerTemplatesSkeleton />
          )}
        </div>
      </div>
    </div>
  );
}
