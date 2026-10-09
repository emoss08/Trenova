import { Button } from "@trenova/shared/components/ui/button";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import { deskLinkClass } from "./desk-button-styles";
import { DeskIcon, type DeskIconName } from "./desk-icons";

/** What a card is about: something failed, something to watch, a boundary, or news. */
export type DeskErrorTone = "err" | "warn" | "neutral" | "info";

/**
 * A card in the conversation for something that did not go to plan: a reply
 * that failed or stopped, a question the agent may not take, a limit reached.
 * Its tone colours it; its actions are the one or two ways forward.
 */
export function DeskErrorCard({
  tone = "err",
  icon = "alert",
  title,
  sub,
  compact = false,
  actions,
  children,
}: {
  tone?: DeskErrorTone;
  icon?: DeskIconName;
  title?: ReactNode;
  sub?: ReactNode;
  compact?: boolean;
  actions?: ReactNode;
  children?: ReactNode;
}) {
  return (
    <div
      className={cn("dk-ec-card", `dk-t-${tone}`, compact && "dk-sm")}
      role={tone === "err" ? "alert" : undefined}
    >
      <span className="dk-ec-ic">
        <DeskIcon name={icon} size={15} stroke={2} />
      </span>
      <div className="dk-ec-body">
        {title && <b>{title}</b>}
        {sub && <span className="dk-ec-sub">{sub}</span>}
        {children}
        {actions && <div className="dk-ec-acts">{actions}</div>}
      </div>
    </div>
  );
}

/** A button in an error card: ink for the way forward, quiet for the rest. */
export function DeskErrorButton({
  ink = false,
  disabled,
  title,
  onClick,
  children,
}: {
  ink?: boolean;
  disabled?: boolean;
  title?: string;
  onClick?: () => void;
  children: ReactNode;
}) {
  return (
    <Button
      variant={ink ? "default" : "quiet"}
      size="sm"
      className={cn(
        "rounded-lg px-2.75",
        ink
          ? "bg-dsk-ink text-dsk-ink-fg hover:bg-dsk-ink hover:opacity-90 active:bg-dsk-ink"
          : "text-dsk-fg2 ring-1 ring-dsk-b ring-inset hover:bg-dsk-fg/6 hover:text-dsk-fg2 active:bg-dsk-fg/6",
      )}
      disabled={disabled}
      title={title}
      onClick={onClick}
    >
      {children}
    </Button>
  );
}

/** A link-like action in an error card or lock line: underlined text, no fill. */
export function DeskErrorLink({
  className,
  onClick,
  children,
}: {
  className?: string;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <Button
      variant="bare"
      size="bare"
      className={cn(
        deskLinkClass,
        className,
      )}
      onClick={onClick}
    >
      {children}
    </Button>
  );
}
