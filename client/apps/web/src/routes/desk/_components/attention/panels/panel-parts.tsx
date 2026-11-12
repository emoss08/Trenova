import { cn } from "@trenova/shared/lib/utils";
import { createContext, useContext, type ReactNode } from "react";

/**
 * The frame every panel of a Needs you item is drawn in: one quiet card that
 * says what is happening in a line, then shows it. A panel is made for one
 * kind of thing (a stop, a clock, a paper), so no two kinds read alike.
 */
export function PanelCard({
  eyebrow,
  title,
  badge,
  children,
}: {
  /** The kind of thing, in a word or two: "Delivery", "Detention". */
  eyebrow: string;
  /** What it is about: the place, the person, the paper. */
  title: ReactNode;
  badge?: ReactNode;
  children?: ReactNode;
}) {
  const action = useContext(PanelActionContext);

  return (
    <section className="mt-5.5 animate-rise rounded-[8px] border border-dsk-b bg-dsk-card">
      <header className="flex items-start gap-3 px-3.5 pt-3 pb-2.5">
        <div className="flex min-w-0 flex-1 flex-col gap-0.5">
          <span className="text-xs text-dsk-faint">{eyebrow}</span>
          <h2 className="text-sm/snug font-semibold text-dsk-fg">{title}</h2>
        </div>
        {badge}
      </header>
      {children}
      {action}
    </section>
  );
}

/**
 * Where the thing a panel shows is fixed, as a link along the panel's foot.
 * The item's detail provides it, so every panel offers the way there without
 * each one knowing where its record lives.
 */
export const PanelActionContext = createContext<ReactNode>(null);

/** A panel's figures in a row: a few numbers a person reads at a glance. */
export function Figures({ children }: { children: ReactNode }) {
  return (
    <dl className="grid grid-cols-[repeat(auto-fit,minmax(7.5rem,1fr))] gap-px overflow-hidden rounded-b-[8px] border-t border-dsk-b-sub bg-dsk-b-sub">
      {children}
    </dl>
  );
}

export function Figure({
  label,
  value,
  note,
  tone,
}: {
  label: string;
  value: ReactNode;
  /** A quieter word under the value: how long ago, how far off. */
  note?: string;
  tone?: "warn" | "bad";
}) {
  return (
    <div className="flex min-w-0 flex-col gap-0.5 bg-dsk-card px-3.5 py-2.5">
      <dt className="truncate text-xs text-dsk-subtle">{label}</dt>
      <dd
        className={cn(
          "truncate text-dsk-base font-medium tabular-nums",
          tone === "bad" ? "text-dsk-error" : tone === "warn" ? "text-dsk-warn-fg" : "text-dsk-fg",
        )}
      >
        {value}
      </dd>
      {note ? <dd className="truncate text-xs text-dsk-faint">{note}</dd> : null}
    </div>
  );
}

/** A panel's body between its head and its figures. */
export function PanelBody({ className, children }: { className?: string; children: ReactNode }) {
  return <div className={cn("px-3.5 pb-3", className)}>{children}</div>;
}
