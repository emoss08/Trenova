import { Sheet, SheetContent, SheetDescription, SheetTitle } from "@trenova/shared/components/ui/sheet";
import { Button } from "@trenova/shared/components/ui/button";
import { XCloseIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
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

/** The sheet is min(680px, the viewport less its inset) wide; a wider editor asks for wide. */
const WIDTHS = {
  default: "data-[side=right]:sm:max-w-[680px]",
  wide: "data-[side=right]:sm:max-w-[960px]",
} as const;

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
 * The one shell every side editor is drawn in: a header, one column of sections, an
 * optional aside, and the save bar, which slides in once there is something to save. Closing goes through the edit flow, so unsaved work
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
  width = "default",
}: EditSheetProps<T>) {
  const t = useT();
  const changed = new Set(flow.changed);
  const footerShown = Boolean(
    flow.dirty || flow.confirmingClose || flow.saved || flow.create || footerLeading,
  );

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
        className={cn(
          "w-[calc(100%-16px)] gap-0 overflow-hidden bg-canvas p-0 data-[side=right]:top-2 data-[side=right]:right-2 data-[side=right]:bottom-2",
          WIDTHS[width],
        )}
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
          <div className="min-w-0 flex-1 overflow-y-auto">
            {sections.map((section) => (
              <section
                key={section.id}
                data-section={section.id}
                className="relative flex flex-col gap-3 border-b border-border px-5 py-4 last:border-b-0"
              >
                {section.keys?.some((key) => changed.has(key)) && (
                  <span
                    className="absolute top-5.5 left-2 size-1.5 rounded-full bg-brand"
                    aria-label={t("Changed")}
                  />
                )}
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
        {footerShown && (
          <footer className="border-t border-border px-4 py-2.5 animate-in fade-in-0 slide-in-from-bottom-2">
            <SaveBar
              flow={flow}
              saveLabel={saveLabel}
              leading={footerLeading}
              review={<ChangeReview form={form} flow={flow} fields={fields} />}
            />
          </footer>
        )}
      </SheetContent>
    </Sheet>
  );
}
