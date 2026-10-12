import { keybindGroups } from "@/config/keybinds.config";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { useT } from "@trenova/shared/i18n/use-t";

const TRIAGE_KEYBINDS = keybindGroups.find((group) => group.id === "triage")?.keybinds ?? [];

/** Triage's keys, read from the same config that binds them. */
export function ShortcutsSheet({ open, onClose }: { open: boolean; onClose: () => void }) {
  const t = useT();

  return (
    <Dialog open={open} onOpenChange={(next) => (next ? undefined : onClose())}>
      <DialogContent size="sm">
        <DialogHeader>
          <DialogTitle>{t("Shortcuts")}</DialogTitle>
          <DialogDescription>{t("Work the queue without leaving the keyboard.")}</DialogDescription>
        </DialogHeader>
        <dl className="grid grid-cols-[1fr_auto] gap-x-6 gap-y-2 text-sm">
          {TRIAGE_KEYBINDS.map((keybind) => (
            <div key={keybind.id} className="contents">
              <dt className="text-foreground-muted">{t(keybind.label)}</dt>
              <dd className="flex justify-end gap-1">
                {keybind.keys.map((key) => (
                  <Kbd key={key}>{key}</Kbd>
                ))}
              </dd>
            </div>
          ))}
        </dl>
      </DialogContent>
    </Dialog>
  );
}
