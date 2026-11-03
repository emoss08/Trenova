import {
  ChevronLeftIcon,
  ChevronRightIcon,
  Expand01Icon,
  SwitchHorizontal01Icon,
} from "@trenova/shared/components/icons";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useState, type ReactNode } from "react";
import { ACTUAL_PAGE_WIDTH, type PageZoom } from "./billing-queue-page-zoom";

/**
 * The bar over the foot of the page column: fit to width or page, zoom, rotate every page,
 * download, and turn pages. The page count is a field too: click it, type a page and press
 * Enter to go there, or Escape to leave it.
 */
export function BillingQueuePageToolbar({
  zoom,
  sheetWidth,
  page,
  pageCount,
  onZoom,
  onStepZoom,
  onRotateAll,
  onDownload,
  downloading,
  onTurn,
}: {
  zoom: PageZoom;
  sheetWidth: number;
  page: number;
  pageCount: number;
  onZoom: (zoom: PageZoom) => void;
  onStepZoom: (direction: 1 | -1) => void;
  onRotateAll: (() => void) | null;
  onDownload: () => void;
  downloading: boolean;
  onTurn: (page: number) => void;
}) {
  const t = useT();
  const [going, setGoing] = useState(false);
  const label =
    zoom.mode === "width"
      ? t("Fit width")
      : zoom.mode === "page"
        ? t("Fit page")
        : t("{0}%", Math.round((sheetWidth / ACTUAL_PAGE_WIDTH) * 100));

  return (
    <div
      role="toolbar"
      aria-label={t("Viewer controls")}
      className="bg-foreground text-background absolute bottom-3 left-1/2 flex -translate-x-1/2 items-center gap-0.5 rounded-lg p-0.75 text-sm whitespace-nowrap"
    >
      <ToolbarButton
        label={t("Fit width")}
        pressed={zoom.mode === "width"}
        onClick={() => onZoom({ mode: "width" })}
      >
        <SwitchHorizontal01Icon className="size-3.5" />
      </ToolbarButton>
      <ToolbarButton
        label={t("Fit page")}
        pressed={zoom.mode === "page"}
        onClick={() => onZoom({ mode: "page" })}
      >
        <Expand01Icon className="size-3.5" />
      </ToolbarButton>
      <span aria-hidden className="bg-background/25 mx-1 h-4 w-px" />
      <ToolbarButton label={t("Zoom out")} onClick={() => onStepZoom(-1)}>
        −
      </ToolbarButton>
      <span aria-live="polite" className="min-w-16 px-1.5 text-center font-mono text-xs">
        {label}
      </span>
      <ToolbarButton label={t("Zoom in")} onClick={() => onStepZoom(1)}>
        +
      </ToolbarButton>
      <span aria-hidden className="bg-background/25 mx-1 h-4 w-px" />
      {onRotateAll ? (
        <ToolbarButton wide label={t("Rotate all pages")} onClick={onRotateAll}>
          {t("Rotate all")}
        </ToolbarButton>
      ) : null}
      <ToolbarButton wide label={t("Download")} onClick={onDownload} busy={downloading}>
        <span className="relative">
          <span className={cn(downloading && "invisible")}>{t("Download")}</span>
          {downloading ? (
            <span className="absolute inset-0 grid place-items-center">
              <Spinner className="size-3.5" />
            </span>
          ) : null}
        </span>
      </ToolbarButton>
      <span aria-hidden className="bg-background/25 mx-1 h-4 w-px" />
      <ToolbarButton
        label={t("Previous page")}
        onClick={() => onTurn(page - 1)}
        disabled={page <= 1}
      >
        <ChevronLeftIcon className="size-3.5" />
      </ToolbarButton>
      {going ? (
        <GoToPage
          page={page}
          pageCount={pageCount}
          onGo={(to) => {
            setGoing(false);
            onTurn(to);
          }}
          onLeave={() => setGoing(false)}
        />
      ) : (
        <button
          type="button"
          aria-label={t("Page {0} of {1}. Go to a page.", page, pageCount)}
          onClick={() => setGoing(true)}
          className="ui-focus-ring text-background/80 hover:bg-background/15 hover:text-background flex h-6.5 items-center rounded-md px-1.5 font-mono text-xs"
        >
          {t("{0} of {1}", page, pageCount)}
        </button>
      )}
      <ToolbarButton
        label={t("Next page")}
        onClick={() => onTurn(page + 1)}
        disabled={page >= pageCount}
      >
        <ChevronRightIcon className="size-3.5" />
      </ToolbarButton>
    </div>
  );
}

function GoToPage({
  page,
  pageCount,
  onGo,
  onLeave,
}: {
  page: number;
  pageCount: number;
  onGo: (page: number) => void;
  onLeave: () => void;
}) {
  const t = useT();
  const [value, setValue] = useState(String(page));

  return (
    <span className="flex h-6.5 items-center gap-1 px-1 font-mono text-xs">
      <input
        // oxlint-disable-next-line jsx-a11y/no-autofocus -- the field opens because the person clicked the page count to type a page
        autoFocus
        inputMode="numeric"
        aria-label={t("Go to page")}
        value={value}
        onFocus={(event) => event.currentTarget.select()}
        onChange={(event) => setValue(event.currentTarget.value.replace(/\D/g, ""))}
        onBlur={onLeave}
        onKeyDown={(event) => {
          if (event.key === "Escape") {
            event.preventDefault();
            event.stopPropagation();
            onLeave();
          }
          if (event.key === "Enter") {
            event.preventDefault();
            const to = Number.parseInt(value, 10);
            if (Number.isFinite(to)) onGo(Math.min(pageCount, Math.max(1, to)));
            else onLeave();
          }
        }}
        className="bg-background/15 text-background w-10 rounded-xs px-1 text-right outline-none"
      />
      <span className="text-background/70">{t("of {0}", pageCount)}</span>
    </span>
  );
}

function ToolbarButton({
  label,
  onClick,
  disabled = false,
  wide = false,
  pressed,
  busy = false,
  children,
}: {
  label: string;
  onClick: () => void;
  disabled?: boolean;
  wide?: boolean;
  pressed?: boolean;
  busy?: boolean;
  children: ReactNode;
}) {
  return (
    <button
      type="button"
      aria-label={label}
      title={label}
      aria-pressed={pressed}
      aria-busy={busy || undefined}
      disabled={disabled || busy}
      onClick={onClick}
      className={cn(
        "ui-focus-ring hover:bg-background/15 flex h-6.5 items-center justify-center rounded-md disabled:opacity-40",
        pressed && "bg-background/20",
        wide ? "px-2" : "w-7",
      )}
    >
      {children}
    </button>
  );
}
