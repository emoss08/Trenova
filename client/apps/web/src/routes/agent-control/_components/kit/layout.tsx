import { cn } from "@trenova/shared/lib/utils";
import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type CSSProperties,
  type KeyboardEvent,
  type ReactNode,
} from "react";
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

export type Fig = {
  label: string;
  value: ReactNode;
  sub: ReactNode;
  /** An extra class on the value: "dim" for a value that is a prompt rather than a figure. */
  tone?: string;
};

/** A row of figures separated by rules. */
export function Figs({ items }: { items: Fig[] }) {
  return (
    <section className="figs" style={{ "--n": items.length } as CSSProperties}>
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

type HoldProps = {
  label: string;
  onDone: () => void;
  /** How long the hold takes. */
  ms?: number;
  disabled?: boolean;
};

/** A button that does its thing only once held down until it fills. */
export function Hold({ label, onDone, ms = 900, disabled = false }: HoldProps) {
  const [progress, setProgress] = useState(0);
  const frame = useRef<number | null>(null);
  const startedAt = useRef(0);

  const stop = useCallback(() => {
    if (frame.current !== null) {
      cancelAnimationFrame(frame.current);
      frame.current = null;
    }
    setProgress(0);
  }, []);

  const start = useCallback(() => {
    if (disabled) {
      return;
    }
    stop();
    startedAt.current = performance.now();
    const tick = (now: number) => {
      const value = Math.min(1, (now - startedAt.current) / ms);
      setProgress(value);
      if (value < 1) {
        frame.current = requestAnimationFrame(tick);
      } else {
        frame.current = null;
        setProgress(0);
        onDone();
      }
    };
    frame.current = requestAnimationFrame(tick);
  }, [disabled, ms, onDone, stop]);

  useEffect(() => stop, [stop]);

  const onKeyDown = (event: KeyboardEvent<HTMLButtonElement>) => {
    if ((event.key === " " || event.key === "Enter") && !event.repeat) {
      event.preventDefault();
      start();
    }
  };

  return (
    <button
      type="button"
      className={cn("hold", progress > 0 && "ing")}
      style={{ "--p": progress } as CSSProperties}
      disabled={disabled}
      onPointerDown={start}
      onPointerUp={stop}
      onPointerLeave={stop}
      onKeyDown={onKeyDown}
      onKeyUp={stop}
    >
      <span className="hold-f" />
      <span className="hold-t">
        <Ic n="pause" s={13} w={2.2} />
        {label}
      </span>
    </button>
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
};

/** A segmented choice between a few options. */
export function Seg<T extends string | number>({
  v,
  opts,
  onChange,
  className,
  label,
  disabled,
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
