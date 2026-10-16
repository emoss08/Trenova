import { ChevronDownIcon, ChevronUpIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { ComponentProps, ReactNode } from "react";

export const QUEUE_NAV_CLASS =
  "size-7.5 rounded-lg text-dsk-muted inset-ring inset-ring-dsk-b-sub hover:bg-dsk-hover hover:text-dsk-fg";
const ACTION_CLASS = "h-9.5 gap-2.5 rounded-lg pr-2.5 pl-4 text-base @max-[900px]/mainc:pl-3";
const OUTLINE_CLASS =
  "text-dsk-fg inset-ring inset-ring-dsk-b hover:bg-dsk-hover hover:text-dsk-fg";

/** The bar under the card: moving through the queue on the left, acting on the right. */
export function QueueBar({ children }: { children: ReactNode }) {
  return <div className="dk-dc2-bar">{children}</div>;
}

/** Fills the bar between the navigation and the actions. */
export function QueueBarSpacer() {
  return <span className="flex-1" />;
}

/** Previous (K) and next (J). */
export function QueueNav({ onPrevious, onNext }: { onPrevious: () => void; onNext: () => void }) {
  const t = useT();

  return (
    <>
      <Button
        variant="quiet"
        size="icon-sm"
        className={QUEUE_NAV_CLASS}
        onClick={onPrevious}
        title={t("Previous (K)")}
        aria-label={t("Previous")}
        aria-keyshortcuts="K"
      >
        <ChevronUpIcon className="size-3.5" strokeWidth={2.4} />
      </Button>
      <Button
        variant="quiet"
        size="icon-sm"
        className={QUEUE_NAV_CLASS}
        onClick={onNext}
        title={t("Next (J)")}
        aria-label={t("Next")}
        aria-keyshortcuts="J"
      >
        <ChevronDownIcon className="size-3.5" strokeWidth={2.4} />
      </Button>
    </>
  );
}

/** The key a button mirrors, drawn inside it. */
export function QueueKey({ children }: { children: ReactNode }) {
  return (
    <span className="dk-kbd" aria-hidden>
      {children}
    </span>
  );
}

/**
 * One of the bar's actions with the key that does the same. The ink one is
 * the action the queue expects; the rest are outlined.
 */
export function QueueButton({
  shortcut,
  ink = false,
  className,
  children,
  ...props
}: Omit<ComponentProps<typeof Button>, "variant" | "size"> & {
  shortcut?: string;
  ink?: boolean;
}) {
  return (
    <Button
      variant={ink ? "default" : "quiet"}
      size="lg"
      className={cn(ACTION_CLASS, ink ? "dk-dc2-ok" : OUTLINE_CLASS, className)}
      aria-keyshortcuts={shortcut}
      {...props}
    >
      {children}
      {shortcut && <QueueKey>{shortcut}</QueueKey>}
    </Button>
  );
}
