import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  useCallback,
  useEffect,
  useRef,
  type CSSProperties,
  type KeyboardEvent,
  type ReactNode,
  type Ref,
} from "react";
import { Ic } from "./ic";

type SearchProps = {
  value: string;
  onChange: (value: string) => void;
  placeholder: string;
  inputRef?: Ref<HTMLInputElement>;
};

/** A search box; Escape clears it and lets go, and "/" is the key that reaches it. */
export function Search({ value, onChange, placeholder, inputRef }: SearchProps) {
  const t = useT();

  return (
    <label className="srch">
      <Ic n="search" s={13} />
      <input
        ref={inputRef}
        value={value}
        placeholder={placeholder}
        aria-label={placeholder}
        onChange={(event) => onChange(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === "Escape") {
            onChange("");
            event.currentTarget.blur();
          }
        }}
      />
      {value ? (
        <button
          type="button"
          className="ib xs"
          aria-label={t("Clear the search")}
          onClick={() => onChange("")}
        >
          <Ic n="x" s={11} />
        </button>
      ) : (
        <span className="kbd">/</span>
      )}
    </label>
  );
}

/** Focuses a search box when "/" is pressed anywhere a person is not typing. */
export function useSlashFocus(inputRef: { current: HTMLInputElement | null }) {
  useEffect(() => {
    const onKey = (event: globalThis.KeyboardEvent) => {
      const target = event.target as HTMLElement | null;
      if (
        event.key !== "/" ||
        event.metaKey ||
        event.ctrlKey ||
        (target && (target.isContentEditable || /INPUT|TEXTAREA|SELECT/.test(target.tagName)))
      ) {
        return;
      }
      event.preventDefault();
      inputRef.current?.focus();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [inputRef]);
}

export type MenuItem =
  | { kind: "heading"; label: string }
  | { kind: "separator" }
  | {
      kind: "item";
      label: string;
      note?: string;
      icon?: ReactNode;
      danger?: boolean;
      disabled?: boolean;
      onSelect: () => void;
    };

type MenuProps = {
  items: MenuItem[];
  onClose: () => void;
  right?: boolean;
  className?: string;
  label: string;
};

/**
 * A menu dropped from the button beside it. It takes focus on its first item, moves with
 * the arrow keys, and closes on Escape, a choice, or a click anywhere else.
 */
export function Menu({ items, onClose, right = false, className, label }: MenuProps) {
  const root = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const first = root.current?.querySelector<HTMLButtonElement>("[role=menuitem]:not(:disabled)");
    first?.focus();
    const onDown = (event: MouseEvent) => {
      if (!root.current?.contains(event.target as Node)) {
        onClose();
      }
    };
    const timer = window.setTimeout(() => document.addEventListener("mousedown", onDown));
    return () => {
      window.clearTimeout(timer);
      document.removeEventListener("mousedown", onDown);
    };
  }, [onClose]);

  const onKeyDown = useCallback(
    (event: KeyboardEvent<HTMLDivElement>) => {
      if (event.key === "Escape") {
        event.preventDefault();
        event.stopPropagation();
        onClose();
        return;
      }
      if (event.key !== "ArrowDown" && event.key !== "ArrowUp") {
        return;
      }
      event.preventDefault();
      const options = [
        ...(root.current?.querySelectorAll<HTMLButtonElement>("[role=menuitem]:not(:disabled)") ??
          []),
      ];
      const at = options.indexOf(document.activeElement as HTMLButtonElement);
      const step = event.key === "ArrowDown" ? 1 : -1;
      options[(at + step + options.length) % options.length]?.focus();
    },
    [onClose],
  );

  return (
    <div
      ref={root}
      role="menu"
      aria-label={label}
      className={cn("mn", right && "right", className)}
      onKeyDown={onKeyDown}
    >
      {items.map((item, index) => {
        if (item.kind === "separator") {
          return <span key={index} className="mn-sep" role="separator" />;
        }
        if (item.kind === "heading") {
          return (
            <div key={index} className="mn-h">
              {item.label}
            </div>
          );
        }
        return (
          <button
            key={index}
            type="button"
            role="menuitem"
            className="mn-i"
            disabled={item.disabled}
            onClick={() => {
              onClose();
              item.onSelect();
            }}
          >
            {item.icon}
            <span className="mn-l">
              <b className={cn(item.danger && "t-d")}>{item.label}</b>
              {item.note && <em>{item.note}</em>}
            </span>
          </button>
        );
      })}
    </div>
  );
}

type RingProps = {
  /** How full, 0 to 1. */
  value: number;
  size?: number;
  className?: string;
};

/** A ring filled to a share. */
export function Ring({ value, size = 38, className }: RingProps) {
  const radius = size / 2 - 3;
  const circumference = 2 * Math.PI * radius;
  const half = size / 2;

  return (
    <svg
      className={cn("rng", className)}
      width={size}
      height={size}
      viewBox={`0 0 ${size} ${size}`}
      aria-hidden
    >
      <circle cx={half} cy={half} r={radius} className="rng-b" />
      <circle
        cx={half}
        cy={half}
        r={radius}
        className="rng-f"
        strokeDasharray={circumference}
        strokeDashoffset={circumference * (1 - Math.min(1, Math.max(0, value)))}
        transform={`rotate(-90 ${half} ${half})`}
      />
    </svg>
  );
}

type RunsProps = {
  days: readonly number[];
  on: boolean;
  label: string;
};

/** One bar a day, scaled to the busiest. */
export function Runs({ days, on, label }: RunsProps) {
  const max = Math.max(1, ...days);

  return (
    <span className={cn("runs", !on && "off")} title={label} role="img" aria-label={label}>
      {days.map((value, index) => (
        <i
          key={index}
          className={value ? undefined : "z"}
          style={{ height: value ? 3 + (value / max) * 15 : 2 }}
        />
      ))}
    </span>
  );
}

type TierBarProps = {
  /** Reads, proposals and acts, in that order. */
  counts: readonly [number, number, number];
  small?: boolean;
};

/** How an agent's tools split between reading, proposing and acting. */
export function TierBar({ counts, small = false }: TierBarProps) {
  return (
    <span className={cn("tier", small && "sm")}>
      {counts.map((count, index) =>
        count ? (
          <i key={index} className={`t${index}`} style={{ flex: count } as CSSProperties} />
        ) : null,
      )}
    </span>
  );
}
