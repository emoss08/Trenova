import { CommandGroup, CommandItem } from "@trenova/shared/components/ui/command";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { PaletteAction } from "./palette-model";
import { MatchText, ShortcutKeys } from "./palette-row";

export function actionItemKey(action: PaletteAction): string {
  return `action:${action.id}`;
}

/**
 * Everything that can be done with one row, as its own list. It replaces the
 * results rather than floating over them, so the arrow keys and the search
 * box keep working exactly as they did a moment ago.
 */
export function PaletteActionsView({
  title,
  actions,
  query,
  onRun,
}: {
  title: string;
  actions: readonly PaletteAction[];
  query: string;
  onRun: (action: PaletteAction) => void;
}) {
  const t = useT();

  return (
    <CommandGroup
      heading={t("Actions for {0}", title)}
      className="px-2 pt-2 [&_[cmdk-group-heading]]:px-2.5 [&_[cmdk-group-heading]]:pb-1 [&_[cmdk-group-heading]]:text-xs"
    >
      {actions.map((action) => (
        <CommandItem
          key={action.id}
          value={actionItemKey(action)}
          onSelect={() => onRun(action)}
          className={cn(
            "rounded-control relative h-10 gap-3 px-2.5 text-sm",
            "data-[selected=true]:bg-surface-selected",
            "before:bg-brand before:absolute before:inset-y-2 before:left-0 before:w-0.5 before:rounded-full before:opacity-0 data-[selected=true]:before:opacity-100",
            action.destructive && "text-danger",
          )}
        >
          <action.icon className="size-4" />
          <MatchText text={action.label} query={query} className="flex-1 truncate" />
          {action.shortcut && <ShortcutKeys keys={action.shortcut} className="shrink-0" />}
        </CommandItem>
      ))}
    </CommandGroup>
  );
}
