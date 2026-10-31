import { DotsHorizontalIcon, type IconComponent } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { cn } from "@trenova/shared/lib/utils";
import { Children, type ReactNode } from "react";

/**
 * One paper or link a carrier tender carries (the rate confirmation, the tracking link), as a
 * row: an icon tile, its name, where it stands in a line beneath, its status, the one next step
 * as a button and every other action in a menu.
 */
export function CarrierDocumentRow({
  icon: Icon,
  title,
  subline,
  status,
  action,
  menuLabel,
  menu,
  busy = false,
  className,
}: {
  icon: IconComponent;
  title: string;
  /** Where it stands: each part a node, set apart by dots. */
  subline: ReactNode;
  status?: ReactNode;
  /** The next step, as a small button. */
  action?: ReactNode;
  /** Names the menu for assistive technology. */
  menuLabel: string;
  /** The menu's items; no menu when there are none. */
  menu?: ReactNode;
  /** A request is out: the menu shows a spinner and cannot be opened until it lands. */
  busy?: boolean;
  className?: string;
}) {
  const hasMenu = Children.toArray(menu).length > 0;

  return (
    <div className={cn("flex items-center gap-2.5", className)}>
      <span
        aria-hidden
        className="bg-muted text-foreground-muted flex size-7 shrink-0 items-center justify-center rounded-md"
      >
        <Icon className="size-3.5" />
      </span>
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span className="truncate text-xs font-medium">{title}</span>
        <div className="text-foreground-muted text-2xs flex min-w-0 flex-wrap items-center gap-x-1.5">
          {subline}
        </div>
      </div>
      {status}
      {action}
      {hasMenu && (
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button
                type="button"
                variant="quiet"
                size="icon-xs"
                aria-label={menuLabel}
                aria-busy={busy || undefined}
                disabled={busy}
              >
                {busy ? (
                  <Spinner className="size-3.5" />
                ) : (
                  <DotsHorizontalIcon className="size-3.5" />
                )}
              </Button>
            }
          />
          <DropdownMenuContent align="end" sideOffset={4} className="w-72">
            {menu}
          </DropdownMenuContent>
        </DropdownMenu>
      )}
    </div>
  );
}

/** The parts of a row's subline, set apart by quiet dots. */
export function SublineParts({ parts }: { parts: { key: string; node: ReactNode }[] }) {
  return parts.map((part, index) => (
    <span key={part.key} className="flex min-w-0 items-center gap-x-1.5">
      {index > 0 && <span aria-hidden>·</span>}
      {part.node}
    </span>
  ));
}

/** A placeholder in the row's shape while its data loads. */
export function CarrierDocumentRowSkeleton({ label }: { label: string }) {
  return (
    <div aria-busy aria-label={label} className="flex items-center gap-2.5">
      <span className="ui-shimmer size-7 shrink-0 rounded-md" />
      <div className="flex flex-1 flex-col gap-1">
        <span className="ui-shimmer h-3 w-28 rounded-sm" />
        <span className="ui-shimmer h-2.5 w-40 rounded-sm" />
      </div>
    </div>
  );
}
