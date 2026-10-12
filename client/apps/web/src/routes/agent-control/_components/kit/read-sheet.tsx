import { Dialog } from "@base-ui/react/dialog";
import { useT } from "@trenova/shared/i18n/use-t";
import type { MouseEvent, ReactNode } from "react";
import { Ic } from "./ic";
import { Button } from "@trenova/shared/components/ui/button";

type ReadSheetProps = {
  open: boolean;
  onClose: () => void;
  /** Read by assistive technology as the sheet's name. */
  label: string;
  head: ReactNode;
  children: ReactNode;
};

/** A side sheet for reading one record, with its own controls in the header. */
export function ReadSheet({ open, onClose, label, head, children }: ReadSheetProps) {
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
          <Dialog.Popup className="shx" onMouseDown={onBackdrop}>
            <aside className="sheet" aria-label={label}>
              <header className="sh-h">
                {head}
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-sm"
                  className="text-muted-foreground hover:text-foreground"
                  title={t("Close (Esc)")}
                  aria-label={t("Close (Esc)")}
                  onClick={onClose}
                >
                  <Ic n="x" s={14} />
                </Button>
              </header>
              <div className="sh-b">{children}</div>
            </aside>
          </Dialog.Popup>
        </div>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
