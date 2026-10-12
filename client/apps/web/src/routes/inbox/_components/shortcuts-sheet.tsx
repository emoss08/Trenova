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

/** Which column a key sits in: moving around Triage, or acting on a message. */
const MOVE_IDS: ReadonlySet<string> = new Set([
  "triage-briefing",
  "triage-next",
  "triage-previous",
  "triage-escape",
  "triage-search",
  "triage-panel",
  "triage-shortcuts",
]);

const KEY_GLYPH: Readonly<Record<string, string>> = { Enter: "↵" };

/**
 * Triage's keys in two columns, moving and acting, read from the same config
 * that binds them so the sheet cannot drift from what the keys do.
 */
export function ShortcutsSheet({ open, onClose }: { open: boolean; onClose: () => void }) {
  const t = useT();
  const columns = [
    { id: "move", title: t("Move"), keybinds: TRIAGE_KEYBINDS.filter((row) => MOVE_IDS.has(row.id)) },
    { id: "act", title: t("Act"), keybinds: TRIAGE_KEYBINDS.filter((row) => !MOVE_IDS.has(row.id)) },
  ];

  return (
    <Dialog open={open} onOpenChange={(next) => (next ? undefined : onClose())}>
      <DialogContent size="md" className="gap-0 p-0">
        <DialogHeader className="border-border-subtle gap-0.5 border-b px-4 pt-3.5 pb-3">
          <DialogTitle className="text-base font-semibold">{t("Keyboard shortcuts")}</DialogTitle>
          <DialogDescription className="text-foreground-subtle text-sm">
            {t("Moving is opening.")}
          </DialogDescription>
        </DialogHeader>
        <div className="grid grid-cols-2 gap-x-3 px-2 pt-1 pb-3 max-sm:grid-cols-1">
          {columns.map((column) => (
            <section key={column.id} aria-label={column.title}>
              <h3 className="text-foreground-subtle px-2 pt-2.5 pb-1 text-xs font-medium">
                {column.title}
              </h3>
              <dl className="flex flex-col">
                {column.keybinds.map((keybind) => (
                  <div
                    key={keybind.id}
                    className="hover:bg-surface-hover flex h-8 items-center justify-between gap-3 rounded-md px-2 text-base"
                  >
                    <dt>{t(keybind.label)}</dt>
                    <dd className="flex gap-1">
                      {keybind.keys.map((key) => (
                        <Kbd key={key}>{KEY_GLYPH[key] ?? key}</Kbd>
                      ))}
                    </dd>
                  </div>
                ))}
              </dl>
            </section>
          ))}
        </div>
      </DialogContent>
    </Dialog>
  );
}
