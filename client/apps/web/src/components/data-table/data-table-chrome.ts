/**
 * The classes of a data table's frame, in a module light enough for a page to import up
 * front. A board's skeleton draws the same frame while the table's code loads, so reading
 * these from one place is what keeps the two the same size.
 */
export const TABLE_SHELL_CLASS = "bleed:gap-0 bleed:min-h-0 flex size-full min-w-0 flex-col gap-2";
export const TABLE_PANEL_CLASS = "bleed:min-h-0 bleed:flex-1 flex h-full min-w-0";
export const TABLE_PANEL_CONTENT_CLASS =
  "bleed:gap-0 bleed:min-h-0 flex min-w-0 flex-1 flex-col gap-2";
export const TABLE_TOOLBAR_CLASS =
  "bleed:border-border bleed:border-b bleed:px-3 bleed:py-2 flex items-center justify-between gap-2";
export const TABLE_VIEWPORT_CLASS = "bleed:min-h-0 bleed:flex-1 relative min-w-0";
export const TABLE_PAGER_CLASS =
  "bleed:border-border bleed:min-h-10 bleed:shrink-0 bleed:border-t bleed:px-3 bleed:py-1.5 flex items-center justify-between gap-4 px-2 tabular-nums";
export const DENSITY_COMPACT_ROW = "[--row-h:var(--row-h-compact)]";
export const DENSITY_COMPACT_CELLS = "[&_td]:py-0.5";
// The vertical track starts below the sticky column header, so the scrollbar runs beside the
// rows and never over the header.
export const TABLE_CONTAINER_CLASS =
  "bleed:h-full bleed:max-h-none bleed:rounded-none bleed:border-0 max-h-[65vh] rounded-lg border border-border [&>[data-slot=scroll-area-scrollbar][data-orientation=vertical]]:top-(--row-head-h)! [&>[data-slot=scroll-area-mask][data-orientation=horizontal]]:left-[var(--dt-pin-start,0px)] [&>[data-slot=scroll-area-mask][data-orientation=horizontal]]:right-[var(--dt-pin-end,0px)]";
export const TABLE_BODY_CELL_CLASS = "border-border truncate border-b font-sans";
export const TABLE_HEAD_CELL_CLASS = "border-border relative border-b uppercase";
