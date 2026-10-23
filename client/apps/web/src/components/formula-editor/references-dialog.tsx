import { tariffDocText } from "@/components/formula-editor/tariff-doc";
import { lineOf } from "@/lib/tariff-source-lines";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { useRules } from "@trenova/shared/hooks/use-rules";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { tariffCatalog, type TariffReference } from "@trenova/shared/rules/tariff";
import type { TariffContext } from "@trenova/shared/types/tariff";
import { useRef, type KeyboardEvent } from "react";
import type { TariffReferencesRequest } from "./tariff-service";

const KIND_LABELS: Record<TariffReference["kind"], string> = defineLabels({
  declaration: "Declared",
  read: "Read",
  write: "Written",
});

/** The most of a line a row shows on each side of the use. */
const PREVIEW_CONTEXT = 60;

export type ReferencePreview = {
  /** The 1-based line the use is on, or null when its source is not here. */
  line: number | null;
  before: string;
  match: string;
  after: string;
};

/** The line a use is on, cut around it, from the source it is in. */
export function referencePreview(
  source: string | null,
  reference: TariffReference,
): ReferencePreview {
  if (source === null) return { line: null, before: "", match: "", after: "" };
  const from = Math.min(reference.range.from, source.length);
  const to = Math.min(Math.max(reference.range.to, from), source.length);
  const start = source.lastIndexOf("\n", from - 1) + 1;
  const newline = source.indexOf("\n", to);
  const end = newline === -1 ? source.length : newline;
  const before = source.slice(start, from).trimStart();
  const after = source.slice(to, end).trimEnd();
  return {
    line: lineOf(source, from),
    before: before.length > PREVIEW_CONTEXT ? `…${before.slice(-PREVIEW_CONTEXT)}` : before,
    match: source.slice(from, to),
    after: after.length > PREVIEW_CONTEXT ? `${after.slice(0, PREVIEW_CONTEXT)}…` : after,
  };
}

/** Moves focus between the rows with the arrow keys, Home and End. */
function moveFocus(event: KeyboardEvent<HTMLElement>, list: HTMLElement | null) {
  if (!list) return;
  const rows = [...list.querySelectorAll<HTMLButtonElement>("button[data-reference-row]")];
  const at = rows.indexOf(document.activeElement as HTMLButtonElement);
  const next =
    event.key === "ArrowDown"
      ? Math.min(at + 1, rows.length - 1)
      : event.key === "ArrowUp"
        ? Math.max(at - 1, 0)
        : event.key === "Home"
          ? 0
          : event.key === "End"
            ? rows.length - 1
            : null;
  if (next === null) return;
  event.preventDefault();
  rows[next]?.focus();
}

function ReferenceRow({
  reference,
  preview,
  fallback,
  onPick,
}: {
  reference: TariffReference;
  preview: ReferencePreview;
  /** What a use whose source is not here shows instead of its line. */
  fallback: string | null;
  onPick: ((reference: TariffReference) => void) | null;
}) {
  const t = useT();
  const body = (
    <>
      <span className="text-foreground-subtle w-12 shrink-0 font-mono text-2xs tabular-nums">
        {preview.line !== null ? t("Ln {0}", preview.line) : null}
      </span>
      <code className="min-w-0 flex-1 truncate font-mono text-xs">
        {preview.line !== null ? (
          <>
            <span className="text-foreground-muted">{preview.before}</span>
            <span
              className={cn(
                "rounded-badge text-foreground px-0.5",
                reference.kind === "read" ? "bg-editor-reference" : "bg-editor-reference-write",
              )}
            >
              {preview.match}
            </span>
            <span className="text-foreground-muted">{preview.after}</span>
          </>
        ) : (
          <span className="text-foreground-muted">{fallback}</span>
        )}
      </code>
      <Badge
        variant={reference.kind === "read" ? "neutral" : "accent-amber"}
        appearance={reference.kind === "read" ? "outline" : "subtle"}
        className="shrink-0"
      >
        {KIND_LABELS[reference.kind]}
      </Badge>
    </>
  );
  const rowClass = "flex w-full min-w-0 items-center gap-2 rounded-md px-2 py-1.5 text-left";
  return (
    <li>
      {onPick ? (
        <button
          type="button"
          data-reference-row
          onClick={() => onPick(reference)}
          className={cn(rowClass, "ui-focus-ring hover:bg-surface-hover")}
        >
          {body}
        </button>
      ) : (
        <div className={rowClass}>{body}</div>
      )}
    </li>
  );
}

/**
 * Find references: every use of the name at the cursor, grouped by where it is. A use in this
 * file moves the cursor there; a use in an imported library opens that library at it, the way go
 * to definition does. The arrow keys move between the rows and Escape closes.
 */
export function ReferencesDialog({
  request,
  source,
  context,
  onPick,
  canOpenLibraries,
  onClose,
}: {
  request: TariffReferencesRequest | null;
  /** The file being edited, which the uses' ranges are in. */
  source: string;
  context: TariffContext | null;
  onPick: (reference: TariffReference) => void;
  /** Whether a use in a library can be opened (the page shows imported libraries). */
  canOpenLibraries: boolean;
  onClose: () => void;
}) {
  const t = useT();
  const rules = useRules();
  const listRef = useRef<HTMLDivElement | null>(null);
  const references = request?.references ?? [];
  const local = references.filter((reference) => reference.module === null);
  const libraries = references.filter((reference) => reference.module !== null);
  const catalog = libraries.some(
    (reference) => !context?.libraries.some((library) => library.path === reference.module),
  )
    ? tariffCatalog(rules)
    : null;

  const librarySource = (module: string) =>
    context?.libraries.find((library) => library.path === module)?.source ?? null;
  const standardSignature = (module: string) =>
    catalog?.modules
      .find((candidate) => candidate.path === module)
      ?.declarations.find((declaration) => declaration.name === request?.name) ?? null;

  return (
    <Dialog open={request !== null} onOpenChange={(open) => !open && onClose()}>
      <DialogContent size="lg">
        <DialogHeader>
          <DialogTitle>{t("References to {0}", request?.name ?? "")}</DialogTitle>
          <DialogDescription>
            {t(
              "{0, plural, one {# use} other {# uses}}. Choose one to go to it.",
              references.length,
            )}
          </DialogDescription>
        </DialogHeader>
        <div
          ref={listRef}
          className="flex max-h-[60vh] flex-col gap-3 overflow-auto"
          onKeyDown={(event) => moveFocus(event, listRef.current)}
        >
          {local.length > 0 && (
            <section aria-label={t("In this file")} className="flex flex-col gap-1">
              <h3 className="text-foreground-subtle px-2 text-xs font-medium">
                {t("In this file")}
              </h3>
              <ul className="flex flex-col">
                {local.map((reference) => (
                  <ReferenceRow
                    key={`${reference.range.from}:${reference.kind}`}
                    reference={reference}
                    preview={referencePreview(source, reference)}
                    fallback={null}
                    onPick={onPick}
                  />
                ))}
              </ul>
            </section>
          )}
          {libraries.length > 0 && (
            <section aria-label={t("In libraries")} className="flex flex-col gap-1">
              <h3 className="text-foreground-subtle px-2 text-xs font-medium">
                {t("In libraries")}
              </h3>
              <ul className="flex flex-col">
                {libraries.map((reference) => {
                  const module = reference.module ?? "";
                  const text = librarySource(module);
                  const declaration = text === null ? standardSignature(module) : null;
                  return (
                    <li key={`${module}:${reference.range.from}`} className="flex flex-col">
                      <span className="text-foreground-subtle px-2 pt-1 font-mono text-2xs">
                        {module}
                      </span>
                      <ul>
                        <ReferenceRow
                          reference={reference}
                          preview={referencePreview(text, reference)}
                          fallback={
                            declaration
                              ? [declaration.signature, tariffDocText(declaration.doc)]
                                  .filter(Boolean)
                                  .join(" · ")
                              : t("Part of Tariff's standard library.")
                          }
                          onPick={canOpenLibraries ? onPick : null}
                        />
                      </ul>
                    </li>
                  );
                })}
              </ul>
            </section>
          )}
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            {t("Close")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
