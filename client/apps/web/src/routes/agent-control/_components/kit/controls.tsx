import { SearchLgIcon } from "@trenova/shared/components/icons";
import { Input } from "@trenova/shared/components/ui/input";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, type CSSProperties, type Ref } from "react";
import { aicFieldTrigger } from "../edit/field-trigger";
import { Ic } from "./ic";
import { Button } from "@trenova/shared/components/ui/button";

type SearchProps = {
  value: string;
  onChange: (value: string) => void;
  placeholder: string;
  inputRef?: Ref<HTMLInputElement>;
  /** The large box a page leads with, such as the extension marketplace's. */
  size?: "lg";
};

/** A search box; Escape clears it and lets go, and "/" is the key that reaches it. */
export function Search({ value, onChange, placeholder, inputRef, size }: SearchProps) {
  const t = useT();

  return (
    <Input
      ref={inputRef}
      value={value}
      placeholder={placeholder}
      aria-label={placeholder}
      inputContainerClassName={cn(
        "w-full min-w-45 flex-[0_1_20rem]",
        size === "lg" ? "max-w-95 flex-[0_1_23.75rem]" : "max-w-80",
      )}
      className={size === "lg" ? "h-9 min-h-9" : aicFieldTrigger}
      leftElement={<SearchLgIcon className="size-3.5 text-muted-foreground" />}
      rightElement={
        value ? (
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            className="text-muted-foreground hover:text-foreground"
            aria-label={t("Clear the search")}
            onClick={() => onChange("")}
          >
            <Ic n="x" s={11} />
          </Button>
        ) : (
          <Kbd className="mr-1">/</Kbd>
        )
      }
      onChange={(event) => onChange(event.target.value)}
      onKeyDown={(event) => {
        if (event.key === "Escape") {
          onChange("");
          event.currentTarget.blur();
        }
      }}
    />
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
