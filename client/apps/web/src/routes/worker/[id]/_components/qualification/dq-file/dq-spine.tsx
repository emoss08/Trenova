import { useRichT } from "@trenova/shared/i18n/rich";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import type { DQFSectionValue } from "@trenova/shared/lib/dqf";
import { cn } from "@trenova/shared/lib/utils";
import type { SpineBlock } from "./dq-file-model";

export function spineName(t: TranslateFn, section: DQFSectionValue): string {
  switch (section) {
    case "Credentials":
      return t("Licenses and reviews");
    case "Documents":
      return t("Documents");
    case "SafetyHistory":
      return t("Employers");
    case "DrugAlcohol":
      return t("Drug and alcohol");
  }
}

function spineNote(t: TranslateFn, block: SpineBlock): { text: string; tone: string } {
  if (block.segments.length === 0)
    return { text: t("Nothing recorded"), tone: "text-muted-foreground" };
  if (block.outstanding > 0) {
    return { text: t("{0} outstanding", block.outstanding), tone: "text-danger-foreground" };
  }
  if (block.warning > 0) {
    return {
      text:
        block.section === "SafetyHistory"
          ? t("{0, plural, one {# awaiting an answer} other {# awaiting answers}}", block.warning)
          : t("{0} expiring soon", block.warning),
      tone: "text-warning-foreground",
    };
  }
  return { text: t("All on file"), tone: "text-success-foreground" };
}

const SEGMENT_TONE = {
  success: "bg-success",
  warning: "bg-warning",
  danger: "bg-danger",
} as const;

/**
 * The file's spine: four joined blocks, one per section, each with how much of it is on file
 * and a bar of one segment per row in three tones. Pressing a block brings its section into view.
 */
export function DQSpine({
  blocks,
  onSelect,
}: {
  blocks: readonly SpineBlock[];
  onSelect: (section: DQFSectionValue) => void;
}) {
  const t = useT();
  const rt = useRichT();
  return (
    <div
      role="group"
      aria-label={t("Sections of the file")}
      data-testid="dqf-spine"
      className="grid grid-cols-2 overflow-hidden rounded-lg border bg-card lg:grid-cols-4"
    >
      {blocks.map((block, blockIndex) => {
        const note = spineNote(t, block);
        const name = spineName(t, block.section);
        return (
          <button
            key={block.section}
            type="button"
            data-testid={`dqf-spine-${block.section}`}
            aria-label={t(
              "{0}: {1} of {2} on file, {3}",
              name,
              block.onFile,
              block.segments.length,
              note.text,
            )}
            onClick={() => onSelect(block.section)}
            className={cn(
              "grid min-w-0 grid-cols-[minmax(0,1fr)_auto] gap-x-2 gap-y-1 ui-focus-ring px-3.5 py-3 text-left transition-colors hover:bg-surface-hover active:bg-surface-active",
              blockIndex % 2 === 1 && "border-l border-border-subtle",
              blockIndex >= 2 && "border-t border-border-subtle lg:border-t-0",
              blockIndex === 2 && "lg:border-l",
            )}
          >
            <span className="truncate text-xs font-medium text-foreground-subtle">{name}</span>
            <span className="text-xs whitespace-nowrap text-muted-foreground">
              {rt(
                "<b>{0}</b> of {1}",
                {
                  b: (chunks) => (
                    <b className="text-base font-semibold text-foreground tabular-nums">{chunks}</b>
                  ),
                },
                block.onFile,
                block.segments.length,
              )}
            </span>
            <span aria-hidden className="col-span-2 grid h-1.5 auto-cols-fr grid-flow-col gap-0.5">
              {block.segments.length === 0 ? (
                <span className="rounded-xs bg-surface-active" />
              ) : (
                block.segments.map((tone, index) => (
                  <span
                    key={index}
                    data-tone={tone}
                    style={{ animationDelay: `${index * 30 + blockIndex * 90}ms` }}
                    className={cn(
                      "animate-fill origin-left rounded-xs motion-reduce:animate-none",
                      SEGMENT_TONE[tone],
                    )}
                  />
                ))
              )}
            </span>
            <span className={cn("col-span-2 truncate text-xs", note.tone)}>{note.text}</span>
          </button>
        );
      })}
    </div>
  );
}
