import { useT } from "@trenova/shared/i18n/use-t";
import { formatShortcut } from "@trenova/shared/lib/shortcuts";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import { Ic } from "../kit/ic";
import type { EditFlow } from "./use-edit-flow";

type SaveBarProps = {
  flow: EditFlow;
  /** The list of changes the "unsaved changes" control opens. */
  review?: ReactNode;
  saveLabel?: string;
  /** Anything the editor shows beside the state, such as a test button. */
  leading?: ReactNode;
};

/**
 * The foot of every editor: what is unsaved, why it cannot be saved, and the save. While
 * closing would lose changes it asks first, in place.
 */
export function SaveBar({ flow, review, saveLabel, leading }: SaveBarProps) {
  const t = useT();
  const count = flow.changed.length;

  if (flow.confirmingClose) {
    return (
      <>
        <Ic n="warn" s={14} />
        <span className="es-cft" role="alert">
          {count === 1
            ? t("Discard 1 unsaved change?")
            : t("Discard {0} unsaved changes?", count)}
        </span>
        <span className="sp" />
        <button type="button" className="btn sm" autoFocus onClick={flow.keepEditing}>
          {t("Keep editing")}
        </button>
        <button type="button" className="btn sm dng" onClick={flow.close}>
          {t("Discard and close")}
        </button>
      </>
    );
  }

  return (
    <>
      {flow.reviewing && flow.dirty && review}
      {flow.dirty ? (
        <button
          type="button"
          className={cn("es-chg", flow.reviewing && "on")}
          aria-expanded={flow.reviewing}
          onClick={flow.toggleReview}
        >
          <i />
          {count === 1 ? t("1 unsaved change") : t("{0} unsaved changes", count)}
          <Ic n="chevD" s={12} />
        </button>
      ) : flow.saved ? (
        <span className="es-ok" role="status">
          <Ic n="check" s={12} w={2.4} />
          {t("Saved")}
        </span>
      ) : (
        <span className="es-idle">
          {flow.create ? t("Nothing is created until you save") : t("No changes")}
        </span>
      )}
      {flow.invalid && (
        <span className="es-inv">
          <Ic n="alert" s={12} />
          {flow.invalid}
        </span>
      )}
      {leading}
      <span className="sp" />
      {flow.dirty && !flow.create && (
        <button type="button" className="btn sm" onClick={flow.discard}>
          {t("Discard")}
        </button>
      )}
      <button type="button" className="btn ink" disabled={!flow.canSave} onClick={flow.save}>
        {flow.saving ? (
          <>
            <i className="spn" />
            {t("Saving")}
          </>
        ) : (
          (saveLabel ?? (flow.create ? t("Create") : t("Save changes")))
        )}
        <span className="kbd">{formatShortcut("S")}</span>
      </button>
    </>
  );
}
