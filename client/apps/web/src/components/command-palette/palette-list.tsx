import { Button } from "@trenova/shared/components/ui/button";
import { CommandGroup } from "@trenova/shared/components/ui/command";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { RotateCwIcon } from "lucide-react";
import type { PaletteItem, PaletteSection } from "./palette-model";
import { PaletteRow } from "./palette-row";

function SectionHeading({ section }: { section: PaletteSection }) {
  const count =
    section.count ?? (section.loading || section.error ? undefined : section.items.length);
  return (
    <span className="flex items-center gap-1.5">
      {section.heading}
      {count !== undefined && count > 0 && (
        <span className="bg-sunken text-2xs text-foreground-muted rounded-full px-1.5 tabular-nums">
          {count}
        </span>
      )}
    </span>
  );
}

function SkeletonRows({ rows }: { rows: number }) {
  return (
    <div aria-hidden className="flex flex-col">
      {Array.from({ length: rows }, (_, index) => (
        <div key={index} className="flex min-h-11 items-center gap-3 px-2.5 py-1.5">
          <Skeleton className="rounded-control size-7 shrink-0" />
          <div className="flex flex-1 flex-col gap-1.5">
            <Skeleton className="h-3 w-2/5" />
            <Skeleton className="h-2.5 w-3/5" />
          </div>
        </div>
      ))}
    </div>
  );
}

/**
 * The result list, section by section. Loading and failure are drawn inside
 * the section they belong to, so one slow or failing source never blanks
 * the rest of the palette.
 */
export function PaletteList({
  sections,
  query,
  onSelect,
  onRetry,
}: {
  sections: readonly PaletteSection[];
  query: string;
  onSelect: (item: PaletteItem) => void;
  onRetry: () => void;
}) {
  const t = useT();

  return (
    <>
      {sections.map((section) => (
        <CommandGroup
          key={section.id}
          heading={<SectionHeading section={section} />}
          className="px-2 pt-2 pb-0 [&_[cmdk-group-heading]]:px-2.5 [&_[cmdk-group-heading]]:pb-1 [&_[cmdk-group-heading]]:text-xs"
        >
          {section.items.map((item) => (
            <PaletteRow key={item.key} item={item} query={query} onSelect={onSelect} />
          ))}
          {section.loading && section.items.length === 0 && (
            <SkeletonRows rows={section.placeholderRows ?? 3} />
          )}
          {section.error && (
            <div className="rounded-control bg-danger-subtle text-danger-subtle-foreground ring-danger-border flex items-center justify-between gap-3 px-3 py-2 text-xs ring-1 ring-inset">
              <span>{t("Record search is unavailable right now.")}</span>
              <Button
                type="button"
                size="xs"
                variant="ghost"
                tabIndex={-1}
                onMouseDown={(event) => event.preventDefault()}
                onClick={onRetry}
              >
                <RotateCwIcon className="size-3" />
                {t("Try again")}
              </Button>
            </div>
          )}
        </CommandGroup>
      ))}
    </>
  );
}
