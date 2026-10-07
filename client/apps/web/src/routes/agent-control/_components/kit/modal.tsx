import { Dialog } from "@base-ui/react/dialog";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useId, useState, type MouseEvent, type ReactNode } from "react";
import { F } from "../edit/fields";
import { Ic } from "./ic";

type ModalProps = {
  open: boolean;
  onClose: () => void;
  title: string;
  description?: ReactNode;
  footer?: ReactNode;
  children?: ReactNode;
};

/** A dialog in the middle of the page, over everything else. */
export function Modal({ open, onClose, title, description, footer, children }: ModalProps) {
  const t = useT();

  const onBackdrop = (event: MouseEvent<HTMLDivElement>) => {
    if (event.target === event.currentTarget) {
      onClose();
    }
  };

  return (
    <Dialog.Root open={open} onOpenChange={(next) => !next && onClose()}>
      <Dialog.Portal>
        <div className="aic aic-layer">
          <Dialog.Popup className="shx mdx" onMouseDown={onBackdrop}>
            <div className="modal">
              <header className="md-h">
                <div>
                  <Dialog.Title render={<b />}>{title}</Dialog.Title>
                  {description && (
                    <Dialog.Description render={<span />}>{description}</Dialog.Description>
                  )}
                </div>
                <button type="button" className="ib" aria-label={t("Close")} onClick={onClose}>
                  <Ic n="x" s={14} />
                </button>
              </header>
              {children && <div className="md-b">{children}</div>}
              {footer && <footer className="md-f">{footer}</footer>}
            </div>
          </Dialog.Popup>
        </div>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

type ConfirmDialogProps = {
  open: boolean;
  onClose: () => void;
  title: string;
  description?: ReactNode;
  confirmLabel: string;
  /** A word the person types before the action is offered, for what cannot be undone. */
  typed?: string;
  danger?: boolean;
  busy?: boolean;
  onConfirm: () => void;
};

/** Asks before doing something; for the irreversible, asks the person to type a word first. */
export function ConfirmDialog({
  open,
  onClose,
  title,
  description,
  confirmLabel,
  typed,
  danger = false,
  busy = false,
  onConfirm,
}: ConfirmDialogProps) {
  const t = useT();
  const inputId = useId();
  const [value, setValue] = useState("");
  const ready = !typed || value === typed;

  const close = () => {
    setValue("");
    onClose();
  };

  return (
    <Modal
      open={open}
      onClose={close}
      title={title}
      description={description}
      footer={
        <>
          <span className="sp" />
          <button type="button" className="btn sm" onClick={close}>
            {t("Cancel")}
          </button>
          <button
            type="button"
            className={cn("btn sm", danger ? "dng" : "ink")}
            disabled={!ready || busy}
            onClick={onConfirm}
          >
            {busy && <i className="spn" />}
            {confirmLabel}
          </button>
        </>
      }
    >
      {typed && (
        <F
          htmlFor={inputId}
          label={
            <>
              {t("Type")} <b className="mono">{typed}</b> {t("to confirm")}
            </>
          }
        >
          <label className="inx mono">
            <input
              id={inputId}
              autoFocus
              autoComplete="off"
              value={value}
              onChange={(event) => setValue(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === "Enter" && ready && !busy) {
                  onConfirm();
                }
              }}
            />
          </label>
        </F>
      )}
    </Modal>
  );
}
