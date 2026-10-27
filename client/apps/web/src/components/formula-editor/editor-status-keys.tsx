// editor-status-keys.tsx is what a source editor's status bar says about its keys: the Vim
// toggle and the keys of the language features as the person has them bound. It is shared by the
// Tariff Studios and the Lading studio, so it imports nothing of either.

import { useEditorPreferenceStore } from "@/stores/editor-preference-store";
import { Button } from "@trenova/shared/components/ui/button";
import { Kbd, KbdGroup } from "@trenova/shared/components/ui/kbd";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";

/** Vim keys on or off for the source editors (Tariff and Lading) on this device. */
export function VimToggle() {
  const t = useT();
  const vimKeys = useEditorPreferenceStore((state) => state.vimKeys);
  const setVimKeys = useEditorPreferenceStore((state) => state.setVimKeys);
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            type="button"
            variant="ghost"
            size="xs"
            aria-pressed={vimKeys}
            onClick={() => setVimKeys(!vimKeys)}
            className={cn("font-mono", vimKeys ? "text-foreground" : "text-foreground-muted")}
          />
        }
      >
        {t("Vim")}
      </TooltipTrigger>
      <TooltipContent>
        {vimKeys
          ? t("Vim keys are on. jj or jk leaves insert mode and Space is the leader.")
          : t("Turn on Vim keys")}
      </TooltipContent>
    </Tooltip>
  );
}

/** The editor's keys for its language features, as the person has them bound. */
export function EditorKeys() {
  const t = useT();
  const vimKeys = useEditorPreferenceStore((state) => state.vimKeys);
  const keys = vimKeys
    ? [
        { keys: ["K"], label: t("hover") },
        { keys: ["g", "d"], label: t("definition") },
        { keys: ["Space", "l", "r"], label: t("rename") },
      ]
    : [
        { keys: ["Ctrl", "Space"], label: t("suggest") },
        { keys: ["F12"], label: t("definition") },
        { keys: ["F2"], label: t("rename") },
      ];
  return keys.map(({ keys: chord, label }) => (
    <span key={label} className="flex items-center gap-1 max-[700px]:hidden">
      <KbdGroup>
        {chord.map((key) => (
          <Kbd key={key}>{key}</Kbd>
        ))}
      </KbdGroup>
      {label}
    </span>
  ));
}
