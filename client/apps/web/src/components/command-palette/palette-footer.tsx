import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { useT } from "@trenova/shared/i18n/use-t";
import { ShortcutKeys } from "./palette-row";

export interface FooterHint {
  keys: string[];
  label: string;
}

/**
 * The keys that work right now, and only those. The hints follow the
 * selected row: a command offers no new tab, a record offers its actions.
 */
export function PaletteFooter({
  hints,
  showAskHint,
}: {
  hints: readonly FooterHint[];
  showAskHint: boolean;
}) {
  const t = useT();

  return (
    <footer className="border-border-subtle bg-sunken text-foreground-muted flex h-10 shrink-0 items-center gap-4 overflow-hidden border-t px-4 text-xs">
      {hints.map((hint) => (
        <span key={hint.label} className="flex shrink-0 items-center gap-1.5">
          <ShortcutKeys keys={hint.keys} />
          <span>{hint.label}</span>
        </span>
      ))}
      {showAskHint && (
        <span className="ml-auto hidden shrink-0 items-center gap-1.5 sm:flex">
          <AssistMark className="text-foreground-subtle size-3.5" />
          <span>{t("End with ? to ask the assistant")}</span>
        </span>
      )}
    </footer>
  );
}
