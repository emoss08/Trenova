import { XCloseIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatToUserTimezone } from "@trenova/shared/lib/date";

/** A time on the map's cards and rail: the user's zone, to the minute. */
export function formatMapTime(timestamp: number): string {
  return formatToUserTimezone(timestamp, { showTimeZone: false, showSeconds: false });
}

export function MapCardHeader({
  title,
  badge,
  subtitle,
  leading,
  onClose,
}: {
  title: React.ReactNode;
  badge?: React.ReactNode;
  subtitle?: React.ReactNode;
  leading?: React.ReactNode;
  onClose: () => void;
}) {
  const t = useT();

  return (
    <header className="border-subtle flex flex-col gap-0.5 border-b px-3 pt-2.5 pb-2">
      <div className="flex min-w-0 items-center gap-2">
        {leading}
        <span className="min-w-0 truncate">{title}</span>
        {badge}
        <Button
          variant="ghost"
          size="icon-sm"
          className="ml-auto shrink-0"
          aria-label={t("Close")}
          onClick={onClose}
        >
          <XCloseIcon className="size-3.5" />
        </Button>
      </div>
      {subtitle ? <p className="text-muted-foreground text-xs leading-snug">{subtitle}</p> : null}
    </header>
  );
}

export function MapCardSection({
  title,
  aside,
  children,
}: {
  title: React.ReactNode;
  aside?: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <section className="border-subtle flex flex-col gap-1.5 border-t px-3 py-2.5">
      <h3 className="text-muted-foreground flex items-center gap-1.5 text-xs font-medium">
        {title}
        {aside}
      </h3>
      {children}
    </section>
  );
}

/** A category's swatch, in the colour its alert areas are drawn in. */
export function AlertSwatch({ color, className }: { color: string; className?: string }) {
  return (
    <span
      aria-hidden
      className={className ?? "size-2.5 shrink-0 rounded-sm"}
      style={{ backgroundColor: color }}
    />
  );
}
