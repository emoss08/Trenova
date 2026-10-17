import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { CheckIcon } from "@trenova/shared/components/icons";
import { ShortcutKeys } from "./palette-parts";

export interface FooterHint {
  keys: string[];
  label: string;
}

export type FooterAside =
  | { kind: "ask" }
  | { kind: "question" }
  | { kind: "subject"; title: string }
  | null;

/**
 * The keys that work right now, and only those: the hints follow the
 * selected row, so a command offers no new tab and a record offers its
 * actions. A confirmation (a link copied, a page pinned) takes their place
 * for a moment, since the palette stays open and that is where the eye is.
 */
export function PaletteFooter({
  hints,
  aside,
  confirmation,
}: {
  hints: readonly FooterHint[];
  aside: FooterAside;
  confirmation: string | null;
}) {
  const t = useT();

  return (
    <footer className="border-border-subtle text-foreground-muted flex h-9.5 shrink-0 items-center gap-4 overflow-hidden border-t px-4 text-sm">
      {confirmation ? (
        <span role="status" className="text-foreground animate-rise flex items-center gap-1.5">
          <CheckIcon className="text-success size-3.5" strokeWidth={2} />
          {confirmation}
        </span>
      ) : (
        hints.map((hint, index) => (
          <span
            key={hint.label}
            className={
              index > 2
                ? "hidden shrink-0 items-center gap-1.5 min-[780px]:flex"
                : "flex shrink-0 items-center gap-1.5"
            }
          >
            <ShortcutKeys keys={hint.keys} />
            <span>{hint.label}</span>
          </span>
        ))
      )}
      {aside && (
        <span className="ml-auto hidden shrink-0 items-center gap-1.5 sm:flex">
          {aside.kind === "subject" ? (
            <>
              <ShortcutKeys keys={["⌫"]} />
              <span>{t("Stop asking about {0}", aside.title)}</span>
            </>
          ) : aside.kind === "question" ? (
            <>
              <NovaBadge size="sm" />
              <span>{t("Nova will answer this")}</span>
            </>
          ) : (
            <>
              <NovaBadge size="sm" />
              <span>{t("End with ? to ask Nova")}</span>
            </>
          )}
        </span>
      )}
    </footer>
  );
}
