import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";

type DataTableChangesBannerProps = {
  count: number;
  onMarkAllSeen: () => void;
};

/** Says how many rows on the page changed since the person last had the table open. */
export function DataTableChangesBanner({ count, onMarkAllSeen }: DataTableChangesBannerProps) {
  const t = useT();

  return (
    <div className="border-border bg-muted/40 bleed:rounded-none bleed:border-x-0 bleed:border-t-0 flex items-center justify-center gap-2 rounded-md border px-3 py-1 text-xs">
      <span aria-hidden className="bg-info size-1.5 rounded-full" />
      <span className="text-muted-foreground">
        {t(
          "{0, plural, one {# row changed since your last visit} other {# rows changed since your last visit}}",
          count,
        )}
      </span>
      <Button
        type="button"
        variant="link"
        size="xs"
        className="h-auto p-0 text-xs"
        onClick={onMarkAllSeen}
      >
        {t("Mark all seen")}
      </Button>
    </div>
  );
}
