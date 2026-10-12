import { Button } from "@trenova/shared/components/ui/button";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import { deskLinkClass } from "./desk-button-styles";
import { DeskIcon, type DeskIconName } from "./desk-icons";

/**
 * What a card is about: something failed, something to watch, a boundary,
 * news, or a question waiting on the person.
 */
export type DeskCardTone = "err" | "warn" | "neutral" | "info" | "ask";

/**
 * A card in the conversation or the dock: an icon in its tone, a title, a
 * line under it, whatever the card holds, and the one or two ways forward.
 * Its tone colours it. A failure announces itself; a card that waits on the
 * person is a landmark named by `label`, so it can be found without reading
 * the whole conversation.
 */
export function DeskCard({
  tone,
  icon,
  title,
  titleId,
  sub,
  compact = false,
  label,
  className,
  actions,
  children,
}: {
  tone: DeskCardTone;
  icon: DeskIconName;
  title?: ReactNode;
  /** Set where something else is labelled by the title. */
  titleId?: string;
  sub?: ReactNode;
  compact?: boolean;
  /** Names the card as a region. */
  label?: string;
  className?: string;
  actions?: ReactNode;
  children?: ReactNode;
}) {
  return (
    <div
      className={cn("dk-ec-card", `dk-t-${tone}`, compact && "dk-sm", className)}
      role={tone === "err" ? "alert" : label ? "region" : undefined}
      aria-label={label}
    >
      <span className="dk-ec-ic">
        <DeskIcon name={icon} size={15} stroke={2} />
      </span>
      <div className="dk-ec-body">
        {title && <b id={titleId}>{title}</b>}
        {sub && <span className="dk-ec-sub">{sub}</span>}
        {children}
        {actions && <div className="dk-ec-acts">{actions}</div>}
      </div>
    </div>
  );
}

/** A button in a card: ink for the way forward, quiet for the rest. */
export function DeskCardButton({
  ink = false,
  type = "button",
  disabled,
  title,
  onClick,
  children,
}: {
  ink?: boolean;
  /** A submit button sends the form it sits in. */
  type?: "button" | "submit";
  disabled?: boolean;
  title?: string;
  onClick?: () => void;
  children: ReactNode;
}) {
  return (
    <Button
      type={type}
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

/** A link-like action in a card or lock line: underlined text, no fill. */
export function DeskCardLink({
  className,
  onClick,
  children,
}: {
  className?: string;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <Button variant="bare" size="bare" className={cn(deskLinkClass, className)} onClick={onClick}>
      {children}
    </Button>
  );
}
