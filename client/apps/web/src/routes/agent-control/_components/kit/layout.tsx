import { ScrollArea, type ScrollAreaMaskVariant } from "@trenova/shared/components/ui/scroll-area";
import { cn } from "@trenova/shared/lib/utils";
import { type CSSProperties, type ReactNode } from "react";
import { Ic, type IcName } from "./ic";

type SecHProps = {
  t: string;
  /** A count or a note beside the heading, in the small type. */
  n?: ReactNode;
  /** What sits at the right end: a link or a button. */
  r?: ReactNode;
  ic?: IcName;
};

/** A section heading on the page: title, a note, and an action at the far end. */
export function SecH({ t, n, r, ic }: SecHProps) {
  return (
    <header className="sh2">
      {ic && <Ic n={ic} s={13} />}
      <h3>{t}</h3>
      {n != null && n !== "" && <em className="mono">{n}</em>}
      <span className="sp" />
      {r}
    </header>
  );
}

type ScrollBoxProps = {
  /** What the box holds, so a screen reader user can find it. */
  label: string;
  /** How tall it may grow before it scrolls: sm for prose, md for rows, lg for cards. */
  size?: "sm" | "md" | "lg";
  /** The surface it sits on, so the fade at a clipped edge melts into it. */
  surface?: ScrollAreaMaskVariant;
  children: ReactNode;
};

/**
 * A list kept to its own height inside a section, so a long one scrolls in place
 * instead of pushing everything after it down the page. An edge with more beyond it
 * fades out, and the box takes keyboard focus whenever it has anything to scroll.
 */
export function ScrollBox({
  label,
  size = "md",
  surface = "background",
  children,
}: ScrollBoxProps) {
  return (
    <ScrollArea role="region" aria-label={label} maskVariant={surface} className={cn("scb", size)}>
      {children}
    </ScrollArea>
  );
}

export type Fig = {
  label: string;
  value: ReactNode;
  sub: ReactNode;
  /** An extra class on the value: "dim" for a value that is a prompt rather than a figure. */
  tone?: string;
};

/** A row of figures separated by rules. */
export function Figs({ items, label }: { items: Fig[]; label?: string }) {
  return (
    <section className="figs" aria-label={label} style={{ "--n": items.length } as CSSProperties}>
      {items.map((item) => (
        <div key={item.label} className="fg">
          <span className="lbl">{item.label}</span>
          <b className={cn("big mono", item.tone)}>{item.value}</b>
          <span className="sub">{item.sub}</span>
        </div>
      ))}
    </section>
  );
}

type SegProps<T extends string | number> = {
  v: T;
  opts: readonly (readonly [T, ReactNode])[];
  onChange: (value: T) => void;
  className?: string;
  label?: string;
  /** Options shown but not open to choose, with why in their title. */
  disabled?: (value: T) => string | null;
  /** Called when the pointer or focus reaches an option, to warm what choosing it would load. */
  onHover?: (value: T) => void;
};

/** A segmented choice between a few options. */
export function Seg<T extends string | number>({
  v,
  opts,
  onChange,
  className,
  label,
  disabled,
  onHover,
}: SegProps<T>) {
  return (
    <div className={cn("seg", className)} role="radiogroup" aria-label={label}>
      {opts.map(([key, text]) => {
        const why = disabled?.(key) ?? null;
        return (
          <button
            key={String(key)}
            type="button"
            role="radio"
            aria-checked={v === key}
            disabled={why !== null}
            title={why ?? undefined}
            className={cn(v === key && "on")}
            onClick={(event) => {
              event.stopPropagation();
              onChange(key);
            }}
            onMouseEnter={onHover ? () => onHover(key) : undefined}
            onFocus={onHover ? () => onHover(key) : undefined}
          >
            {text}
          </button>
        );
      })}
    </div>
  );
}

type SwitchProps = {
  on: boolean;
  onChange: (on: boolean) => void;
  label: string;
  disabled?: boolean;
};

/** An on/off switch. */
export function Switch({ on, onChange, label, disabled = false }: SwitchProps) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={on}
      aria-label={label}
      disabled={disabled}
      className={cn("swt", on && "on")}
      onClick={(event) => {
        event.stopPropagation();
        onChange(!on);
      }}
    >
      <i />
    </button>
  );
}
