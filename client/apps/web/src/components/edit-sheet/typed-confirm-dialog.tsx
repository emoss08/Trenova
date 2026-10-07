import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Input } from "@trenova/shared/components/ui/input";
import { useT } from "@trenova/shared/i18n/use-t";
import { type ReactNode, useId, useState } from "react";

type TypedConfirmDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description?: ReactNode;
  confirmLabel: string;
  /** A word the person types before the action is offered, for what cannot be undone. */
  typed?: string;
  danger?: boolean;
  onConfirm: () => void;
};

/** A confirmation that, for the irreversible, asks the person to type a word first. */
export function TypedConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel,
  typed,
  danger = false,
  onConfirm,
}: TypedConfirmDialogProps) {
  const t = useT();
  const inputId = useId();
  const [value, setValue] = useState("");
  const ready = !typed || value.trim() === typed;

  return (
    <AlertDialog
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          setValue("");
        }
        onOpenChange(next);
      }}
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{title}</AlertDialogTitle>
          {description && <AlertDialogDescription>{description}</AlertDialogDescription>}
        </AlertDialogHeader>
        {typed && (
          <label htmlFor={inputId} className="flex flex-col gap-1.5 text-xs">
            <span>
              {t("To confirm, type")} <b className="font-mono">{typed}</b>
            </span>
            <Input
              id={inputId}
              autoFocus
              autoComplete="off"
              className="font-mono"
              value={value}
              onChange={(event) => setValue(event.target.value)}
            />
          </label>
        )}
        <AlertDialogFooter>
          <AlertDialogCancel>{t("Cancel")}</AlertDialogCancel>
          <AlertDialogAction
            variant={danger ? "destructive" : "default"}
            disabled={!ready}
            onClick={() => {
              onConfirm();
              setValue("");
            }}
          >
            {confirmLabel}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
