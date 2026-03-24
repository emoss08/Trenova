import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Kbd, KbdGroup } from "@/components/ui/kbd";
import { keybindGroups } from "@/config/keybinds.config";
import { useHotkey } from "@tanstack/react-hotkeys";
import { useState } from "react";

export function KeyboardShortcutsDialog() {
  const [open, setOpen] = useState(false);

  useHotkey(
    "Mod+/",
    () => {
      setOpen((prev) => !prev);
    },
    {
      ignoreInputs: true,
      preventDefault: true,
    },
  );

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Keyboard Shortcuts</DialogTitle>
          <DialogDescription>
            Available keyboard shortcuts throughout the application.
          </DialogDescription>
        </DialogHeader>
        <div className="-mx-4 max-h-[60vh] overflow-y-auto px-4">
          <div className="flex flex-col gap-4">
            {keybindGroups.map((group) => (
              <div key={group.id} className="flex flex-col gap-2">
                <h3 className="text-xs font-medium tracking-wider text-muted-foreground uppercase">
                  {group.label}
                </h3>
                <div className="flex flex-col">
                  {group.keybinds.map((keybind) => (
                    <div
                      key={keybind.id}
                      className="flex items-center justify-between rounded-md px-2 py-1.5 hover:bg-muted/50"
                    >
                      <div className="flex flex-col gap-0.5">
                        <span className="text-sm font-medium">{keybind.label}</span>
                        <span className="text-xs text-muted-foreground">{keybind.description}</span>
                      </div>
                      <KbdGroup>
                        {keybind.keys.map((key) => (
                          <Kbd key={key}>{key}</Kbd>
                        ))}
                      </KbdGroup>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
