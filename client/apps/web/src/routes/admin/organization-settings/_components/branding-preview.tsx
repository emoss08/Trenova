import { INVOICE_LOGO_CLASS } from "@/components/documents/sample-invoice-data";
import { SampleInvoice } from "@/components/documents/sample-invoice";
import type { ImageDropTarget } from "@/components/image-upload/use-image-upload";
import {
  ChevronSelectorVerticalIcon,
  ImagePlusIcon,
  Trash01Icon,
  Upload01Icon,
} from "@trenova/shared/components/icons";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { useElementWidth } from "@trenova/shared/hooks/use-element-width";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useState, type ReactNode } from "react";

export type BrandingSurface = "sidebar" | "invoice";

/** Draws one of the organization's pictures with the classes of the place that shows it. */
export type BrandMark = (className: string) => ReactNode;

/**
 * What makes a picture in the preview something to press: choosing another, taking it away,
 * and the drop it accepts. Without one the picture is only drawn.
 */
export type BrandMarkControl = {
  /** "Upload logo", said when there is none. */
  uploadLabel: string;
  /** "Change logo", said when there is one. */
  changeLabel: string;
  onChoose: () => void;
  /** Reads the crop dialog's code as the pointer arrives. */
  onIntent: () => void;
  onRemove: () => void;
  removing: boolean;
  dropTarget: ImageDropTarget;
  dragging: boolean;
};

/** One of the organization's pictures: how it is drawn, and how it is changed. */
export type BrandImageSlot = {
  /** Nothing when the picture has not been uploaded. */
  mark?: BrandMark;
  control?: BrandMarkControl;
};

const SIDEBAR_MARK_CLASS = "size-7 shrink-0 rounded-md";

/** The invoice's own width on screen, in pixels: the widest the template is drawn. */
const INVOICE_WIDTH = 704;

function initialsOf(name: string): string {
  return (
    name
      .split(/\s+/)
      .map((word) => word[0] ?? "")
      .join("")
      .toUpperCase()
      .slice(0, 2) || "T"
  );
}

/**
 * A picture in the preview that is pressed to change it. With a picture in place it opens a
 * menu to replace or remove it; with none it goes straight to choosing one. It also takes a
 * dropped file. The dashed outline is what says it can be pressed, before the pointer is on it.
 */
function EditableMark({
  control,
  filled,
  className,
  children,
}: {
  control: BrandMarkControl;
  /** Whether the picture this stands for has been uploaded. */
  filled: boolean;
  className?: string;
  children: ReactNode;
}) {
  const t = useT();
  const buttonClass = cn(
    "relative inline-grid shrink-0 cursor-pointer place-items-center rounded-md ui-focus-ring outline-1 outline-offset-4 outline-border-strong transition-[outline-color] outline-dashed hover:outline-brand data-popup-open:outline-brand",
    control.dragging && "outline-brand",
    className,
  );
  const overlay =
    control.removing || control.dragging ? (
      <span className="absolute -inset-1 grid place-items-center rounded-md bg-scrim text-foreground-on-solid">
        {control.removing ? (
          <Spinner className="size-3.5" />
        ) : (
          <Upload01Icon className="size-3.5" />
        )}
      </span>
    ) : null;

  if (!filled) {
    return (
      <button
        type="button"
        aria-label={control.uploadLabel}
        title={control.uploadLabel}
        className={buttonClass}
        onClick={control.onChoose}
        onPointerEnter={control.onIntent}
        onFocus={control.onIntent}
        {...control.dropTarget}
      >
        {children}
        {overlay}
      </button>
    );
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <button
            type="button"
            aria-label={control.changeLabel}
            title={control.changeLabel}
            disabled={control.removing}
            className={buttonClass}
            onPointerEnter={control.onIntent}
            onFocus={control.onIntent}
            {...control.dropTarget}
          />
        }
      >
        {children}
        {overlay}
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" sideOffset={8} className="w-40">
        <DropdownMenuItem
          title={t("Replace")}
          startContent={<Upload01Icon className="size-4" />}
          onClick={control.onChoose}
        />
        <DropdownMenuItem
          title={t("Remove")}
          color="danger"
          startContent={<Trash01Icon className="size-4" />}
          onClick={control.onRemove}
        />
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

/**
 * The sidebar's mark as the organization switcher draws it: the icon filling its tile, the
 * logo fitted whole when there is no icon, the organization's initials with neither. Pressing
 * it changes the icon, whichever of the three is standing there.
 */
function SidebarMark({
  name,
  logo,
  icon,
}: {
  name: string;
  logo: BrandImageSlot;
  icon: BrandImageSlot;
}) {
  const drawn = icon.mark ? (
    icon.mark(cn(SIDEBAR_MARK_CLASS, "object-cover"))
  ) : logo.mark ? (
    logo.mark(cn(SIDEBAR_MARK_CLASS, "object-contain"))
  ) : (
    <span
      className={cn(
        SIDEBAR_MARK_CLASS,
        "flex items-center justify-center bg-primary text-2xs font-semibold text-primary-foreground",
      )}
    >
      {initialsOf(name)}
    </span>
  );

  return icon.control ? (
    <EditableMark control={icon.control} filled={Boolean(icon.mark)}>
      {drawn}
    </EditableMark>
  ) : (
    drawn
  );
}

/** A stretch of the sidebar, collapsed and open: the two ways it carries the mark. */
function SidebarSurface({
  name,
  logo,
  icon,
}: {
  name: string;
  logo: BrandImageSlot;
  icon: BrandImageSlot;
}) {
  const rows = ["w-24", "w-32", "w-20", "w-28"];

  return (
    <div className="flex h-full items-stretch justify-center gap-4 p-5">
      <div className="flex w-13 flex-col items-center gap-3.5 rounded-lg border bg-sidebar py-3.5">
        <SidebarMark name={name} logo={logo} icon={icon} />
        {rows.map((width) => (
          <span key={width} aria-hidden className="size-4 rounded-sm bg-surface-active" />
        ))}
      </div>
      <div className="flex w-64 flex-col gap-3.5 rounded-lg border bg-sidebar p-3.5 text-sidebar-foreground">
        <div className="flex items-center gap-3">
          <SidebarMark name={name} logo={logo} icon={icon} />
          <span className="min-w-0 flex-1 truncate text-sm font-semibold">{name}</span>
          <ChevronSelectorVerticalIcon className="size-3.5 shrink-0 text-muted-foreground" />
        </div>
        {rows.map((width) => (
          <div key={width} aria-hidden className="flex items-center gap-2.5 px-1.5">
            <span className="size-4 shrink-0 rounded-sm bg-surface-active" />
            <span className={cn("h-2 rounded-full bg-surface-active", width)} />
          </div>
        ))}
      </div>
    </div>
  );
}

/**
 * The top of the sample invoice: the masthead that carries the logo, and the parties. It is
 * drawn at its own size where there is room and smaller where there is not, and fades out
 * below, since only its head matters here.
 */
function InvoiceSurface({
  name,
  lines,
  logo,
}: {
  name: string;
  lines: readonly string[];
  logo: BrandImageSlot;
}) {
  const [paneRef, paneWidth] = useElementWidth<HTMLDivElement>();
  const scale = paneWidth > 0 ? Math.min(1, paneWidth / INVOICE_WIDTH) : 1;
  const drawn = logo.mark ? (
    logo.mark(INVOICE_LOGO_CLASS)
  ) : logo.control ? (
    <span className="flex h-11 w-24 items-center justify-center text-doc-ink-3">
      <ImagePlusIcon className="size-4" />
    </span>
  ) : null;

  return (
    <div className="h-full overflow-hidden [mask-image:linear-gradient(to_bottom,black_72%,transparent)] px-5 pt-5">
      <div ref={paneRef} className="mx-auto w-full" style={{ maxWidth: INVOICE_WIDTH }}>
        <div
          className="origin-top-left"
          style={{ width: INVOICE_WIDTH, transform: `scale(${scale})` }}
        >
          <SampleInvoice
            issuer={{
              name,
              lines,
              logo:
                drawn && logo.control ? (
                  <EditableMark control={logo.control} filled={Boolean(logo.mark)}>
                    {drawn}
                  </EditableMark>
                ) : (
                  drawn
                ),
            }}
          />
        </div>
      </div>
    </div>
  );
}

/**
 * Where the organization's pictures land, drawn as those places draw them: the sidebar, which
 * carries the icon, and an invoice, which carries the logo. On the settings page the pictures
 * are pressed here to replace or remove them, so each is changed where it is seen. Beside the
 * crop dialog's stage it only shows, with the picture in hand drawn live.
 */
export function BrandingPreview({
  name,
  lines,
  logo,
  icon,
  surfaces = ["sidebar", "invoice"],
  className,
}: {
  name: string;
  /** The organization's mailing address, a line each. */
  lines: readonly string[];
  logo: BrandImageSlot;
  icon: BrandImageSlot;
  /** The places offered, the first shown to begin with. */
  surfaces?: readonly [BrandingSurface, ...BrandingSurface[]];
  className?: string;
}) {
  const t = useT();
  const [chosen, setChosen] = useState<BrandingSurface>(surfaces[0]);
  const surface = surfaces.includes(chosen) ? chosen : surfaces[0];
  const labels: Record<BrandingSurface, string> = {
    sidebar: t("Sidebar"),
    invoice: t("Invoice"),
  };

  return (
    <section
      aria-label={t("Preview")}
      className={cn("flex min-w-0 flex-col overflow-hidden rounded-lg border bg-sunken", className)}
    >
      <header className="flex h-11 shrink-0 items-center justify-between gap-3 border-b px-3">
        <span className="text-xs font-medium text-foreground-subtle">
          {surfaces.length > 1 ? t("Preview") : t("Preview: {0}", labels[surface])}
        </span>
        {surfaces.length > 1 ? (
          <SegmentedControl
            aria-label={t("Preview")}
            items={surfaces.map((value) => ({ value, label: labels[value] }))}
            value={surface}
            onValueChange={setChosen}
          />
        ) : null}
      </header>
      {/* A preview is pressed, not read: its text and pictures are not there to be selected. */}
      <div className="min-h-0 flex-1 select-none">
        {surface === "sidebar" ? (
          <SidebarSurface name={name} logo={logo} icon={icon} />
        ) : (
          <InvoiceSurface name={name} lines={lines} logo={logo} />
        )}
      </div>
    </section>
  );
}
