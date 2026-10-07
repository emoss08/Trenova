import { Sheet, SheetContent, SheetDescription, SheetTitle } from "@trenova/shared/components/ui/sheet";
import { Button } from "@trenova/shared/components/ui/button";
import { XCloseIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { type ReactNode, useCallback, useRef, useState } from "react";
import type { FieldValues, UseFormReturn } from "react-hook-form";
import { ChangeReview, type EditFields } from "./change-review";
import { ConflictBar } from "./conflict-bar";
import { SaveBar } from "./save-bar";
import type { EditFlow } from "./use-edit-flow";

/** One titled part of an editor. Keys are the fields it holds, to mark it while they differ. */
export type EditSection = {
  id: string;
  label: string;
  keys?: string[];
  note?: ReactNode;
  /** Shown at the right of the section's title. */
  actions?: ReactNode;
  /** A reason the section needs a look, marked beside its title. */
  warning?: string;
  content: ReactNode;
};

const WIDTHS = {
  md: "data-[side=right]:sm:max-w-xl",
  lg: "data-[side=right]:sm:max-w-3xl",
  xl: "data-[side=right]:sm:max-w-5xl",
} as const;

/** Pixels from the top of the body at which a section counts as the one being read. */
const SECTION_READ_OFFSET = 90;
/** More sections than this get a contents list beside them. */
const SECTION_NAV_THRESHOLD = 2;

type EditSheetProps<T extends FieldValues> = {
  open: boolean;
  form: UseFormReturn<T>;
  flow: EditFlow;
  title: string;
  subtitle?: ReactNode;
  icon?: ReactNode;
  headerActions?: ReactNode;
  sections: EditSection[];
  fields: EditFields;
  saveLabel?: string;
  /** A column beside the sections, such as a live preview. */
  aside?: ReactNode;
  /** A notice above the sections. */
  banner?: ReactNode;
  /** Beside the save bar's state, such as a test button. */
  footerLeading?: ReactNode;
  width?: keyof typeof WIDTHS;
};

/**
 * The one shell every side editor is drawn in: a header, its sections with a contents list,
 * an optional aside, and the save bar. Closing goes through the edit flow, so unsaved work
 * is never lost to an Escape or a click outside.
 */
export function EditSheet<T extends FieldValues>({
  open,
  form,
  flow,
  title,
  subtitle,
  icon,
  headerActions,
  sections,
  fields,
  saveLabel,
  aside,
  banner,
  footerLeading,
  width = "lg",
}: EditSheetProps<T>) {
  const t = useT();
  const body = useRef<HTMLDivElement>(null);
  const [active, setActive] = useState<string | undefined>(sections[0]?.id);
  const changed = new Set(flow.changed);

  const onScroll = useCallback(() => {
    const element = body.current;
    if (!element) {
      return;
    }
    let current: string | undefined = sections[0]?.id;
    for (const section of sections) {
      const node = element.querySelector<HTMLElement>(`[data-section="${section.id}"]`);
      if (node && node.offsetTop - element.scrollTop < SECTION_READ_OFFSET) {
        current = section.id;
      }
    }
    if (element.scrollTop + element.clientHeight >= element.scrollHeight - 4) {
      current = sections.at(-1)?.id;
    }
    setActive(current);
  }, [sections]);

  const jump = (id: string) => {
    const node = body.current?.querySelector<HTMLElement>(`[data-section="${id}"]`);
    if (node && body.current) {
      body.current.scrollTo({ top: node.offsetTop - 8, behavior: "smooth" });
    }
    setActive(id);
  };

  const showNav = sections.length > SECTION_NAV_THRESHOLD;
  const footerShown = flow.dirty || flow.confirmingClose || flow.saved || flow.create || footerLeading;

  return (
    <Sheet
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          flow.back();
        }
      }}
    >
      <SheetContent
        side="right"
        showCloseButton={false}
        className={cn("w-full gap-0 overflow-hidden p-0", WIDTHS[width])}
      >
        <header className="flex items-start gap-3 border-b border-border px-4 py-3">
          {icon}
          <div className="min-w-0 flex-1">
            <SheetTitle className="truncate text-base font-semibold">{title}</SheetTitle>
            {subtitle && (
              <SheetDescription className="truncate text-xs text-muted-foreground">
                {subtitle}
              </SheetDescription>
            )}
          </div>
          {headerActions}
          <Button
            size="icon-sm"
            variant="ghost"
            onClick={flow.tryClose}
            aria-label={t("Close (Esc)")}
          >
            <XCloseIcon className="size-4" />
          </Button>
        </header>
        {flow.conflict && (
          <ConflictBar
            conflict={flow.conflict}
            onLoadTheirs={() => void flow.loadTheirs()}
            onKeepMine={flow.keepMine}
          />
        )}
        {banner}
        <div className="flex min-h-0 flex-1">
          {showNav && (
            <nav className="hidden w-40 shrink-0 flex-col gap-0.5 border-r border-border p-2 md:flex">
              {sections.map((section) => (
                <button
                  key={section.id}
                  type="button"
                  onClick={() => jump(section.id)}
                  aria-current={active === section.id ? "true" : undefined}
                  className={cn(
                    "ui-focus-ring flex items-center gap-2 rounded-control px-2 py-1 text-left text-xs",
                    active === section.id
                      ? "bg-muted font-medium text-foreground"
                      : "text-muted-foreground hover:bg-muted",
                  )}
                >
                  <span className="flex-1 truncate">{section.label}</span>
                  {section.keys?.some((key) => changed.has(key)) && (
                    <span className="size-1.5 rounded-full bg-warning" aria-label={t("Changed")} />
                  )}
                </button>
              ))}
            </nav>
          )}
          <div ref={body} onScroll={onScroll} className="min-w-0 flex-1 overflow-y-auto">
            {sections.map((section) => (
              <section
                key={section.id}
                data-section={section.id}
                className="flex flex-col gap-3 border-b border-border px-4 py-4 last:border-b-0"
              >
                <header className="flex items-center gap-2">
                  <h3 className="flex items-center gap-1.5 text-sm font-semibold">
                    {section.label}
                    {section.warning && (
                      <span
                        className="size-1.5 rounded-full bg-warning"
                        title={section.warning}
                        aria-label={section.warning}
                      />
                    )}
                  </h3>
                  <span className="flex-1" />
                  {section.actions}
                </header>
                {section.note && <p className="text-xs text-muted-foreground">{section.note}</p>}
                {section.content}
              </section>
            ))}
          </div>
          {aside && (
            <div className="hidden w-80 shrink-0 overflow-y-auto border-l border-border lg:block">
              {aside}
            </div>
          )}
        </div>
        {flow.reviewing && flow.dirty && <ChangeReview form={form} flow={flow} fields={fields} />}
        <footer
          className={cn(
            "border-t border-border px-4 py-2.5",
            !footerShown && "text-muted-foreground",
          )}
        >
          <SaveBar flow={flow} saveLabel={saveLabel} leading={footerLeading} />
        </footer>
      </SheetContent>
    </Sheet>
  );
}
