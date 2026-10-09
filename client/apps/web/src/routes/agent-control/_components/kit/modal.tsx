import { Dialog } from "@base-ui/react/dialog";
import { FieldWrapper } from "@/components/fields/field-components";
import { Input } from "@trenova/shared/components/ui/input";
import { useT } from "@trenova/shared/i18n/use-t";
import { useId, useState, type MouseEvent, type ReactNode } from "react";
import { aicFieldTrigger } from "../edit/field-trigger";
import { Ic } from "./ic";
import { Button } from "@trenova/shared/components/ui/button";

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
                <Button type="button" variant="ghost" size="icon-sm" className="text-muted-foreground hover:text-foreground" aria-label={t("Close")} onClick={onClose}>
                  <Ic n="x" s={14} />
                </Button>
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
          <Button type="button" variant="outline" size="sm" onClick={close}>
            {t("Cancel")}
          </Button>
          <Button
            type="button"
            variant={danger ? "outline" : "default"}
            size="sm"
            className={
              danger
                ? "border-danger-border text-danger-foreground hover:bg-danger-subtle"
                : undefined
            }
            disabled={!ready}
            isLoading={busy}
            loadingText={confirmLabel}
            onClick={onConfirm}
          >
            {confirmLabel}
          </Button>
        </>
      }
    >
      {typed && (
        <FieldWrapper
          label={
            <>
              {t("Type")} <b className="font-mono">{typed}</b> {t("to confirm")}
            </>
          }
        >
          <Input
            id={inputId}
            autoFocus
            autoComplete="off"
            aria-label={`${t("Type")} ${typed} ${t("to confirm")}`}
            className={aicFieldTrigger}
            value={value}
            onChange={(event) => setValue(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Enter" && ready && !busy) {
                onConfirm();
              }
            }}
          />
        </FieldWrapper>
      )}
    </Modal>
  );
}
