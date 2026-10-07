import { Dialog } from "@base-ui/react/dialog";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { MouseEvent, ReactNode } from "react";
import type { FieldValues, UseFormReturn } from "react-hook-form";
import { Ic } from "../kit/ic";
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
  /** "xl" for an editor that needs the room, such as one with a preview beside it. */
  width?: "xl";
};

/**
 * The one shell every side editor is drawn in: a header, one column of sections, an
 * optional aside, and the save bar, which slides in once there is something to save.
 * Closing goes through the edit flow, so unsaved work is never lost to an Escape or a
 * click outside.
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
  width,
}: EditSheetProps<T>) {
  const t = useT();
  const changed = new Set(flow.changed);
  const footerShown = Boolean(
    flow.dirty || flow.confirmingClose || flow.saved || flow.create || footerLeading,
  );

  const onBackdrop = (event: MouseEvent<HTMLDivElement>) => {
    if (event.target === event.currentTarget) {
      flow.tryClose();
    }
  };

  return (
    <Dialog.Root
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          flow.back();
        }
      }}
    >
      <Dialog.Portal>
        <div className="aic aic-layer">
          <Dialog.Popup className="shx" onMouseDown={onBackdrop}>
            <aside
              className={cn("sheet es es2", aside && "has-aside", width && `w-${width}`)}
              aria-label={title}
            >
              <header className="sh-h">
                {icon}
                <div className="sh-t">
                  <Dialog.Title render={<b />}>{title}</Dialog.Title>
                  {subtitle && (
                    <Dialog.Description render={<span />}>{subtitle}</Dialog.Description>
                  )}
                </div>
                {headerActions}
                <button
                  type="button"
                  className="ib"
                  title={t("Close (Esc)")}
                  aria-label={t("Close (Esc)")}
                  onClick={flow.tryClose}
                >
                  <Ic n="x" s={14} />
                </button>
              </header>
              {flow.conflict && (
                <ConflictBar
                  conflict={flow.conflict}
                  onLoadTheirs={() => void flow.loadTheirs()}
                  onKeepMine={flow.keepMine}
                />
              )}
              {banner}
              <div className="es-m">
                <div className="es-b">
                  {sections.map((section) => (
                    <section
                      key={section.id}
                      id={`es-${section.id}`}
                      className={cn(
                        "es-s",
                        section.keys?.some((key) => changed.has(key)) && "dirty",
                      )}
                    >
                      <header className="es-sh">
                        <h3>
                          {section.label}
                          {section.warning && (
                            <i
                              className="es-wn"
                              title={section.warning}
                              aria-label={section.warning}
                            />
                          )}
                        </h3>
                        {section.actions}
                      </header>
                      {section.note && <p className="es-note">{section.note}</p>}
                      {section.content}
                    </section>
                  ))}
                </div>
                {aside && <div className="es-aside">{aside}</div>}
              </div>
              <footer
                className={cn("es-f", flow.confirmingClose && "cf", footerShown && "in")}
                aria-hidden={!footerShown}
              >
                <SaveBar
                  flow={flow}
                  saveLabel={saveLabel}
                  leading={footerLeading}
                  review={<ChangeReview form={form} flow={flow} fields={fields} />}
                />
              </footer>
            </aside>
          </Dialog.Popup>
        </div>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
