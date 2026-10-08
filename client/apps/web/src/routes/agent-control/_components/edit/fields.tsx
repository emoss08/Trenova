import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import { Ic, type IcName } from "../kit/ic";
import { Switch } from "../kit/layout";

type FProps = {
  label: ReactNode;
  hint?: ReactNode;
  /** The field's error, shown in place of the hint. */
  error?: string;
  htmlFor?: string;
  /** Beside the label, such as a count. */
  aside?: ReactNode;
  className?: string;
  children: ReactNode;
};

/**
 * One field of an editor: the label and its hint in a fixed column, the control beside
 * it. Every editor lays its fields out with this, so they line up from one sheet to the
 * next.
 */
export function F({ label, hint, error, htmlFor, aside, className, children }: FProps) {
  return (
    <div className={cn("f", error && "err", className)}>
      <div className="f-l">
        <label htmlFor={htmlFor}>{label}</label>
        {aside}
      </div>
      {children}
      {(error || hint) && <p className="f-h">{error || hint}</p>}
    </div>
  );
}

export type CalloutTone = "i" | "w" | "d" | "k";

const CALLOUT_ICON: Record<CalloutTone, IcName> = {
  i: "sparkle",
  w: "warn",
  d: "alert",
  k: "check",
};

type CalloutProps = {
  tone?: CalloutTone;
  icon?: IcName;
  /** A button at the end of the callout. */
  action?: ReactNode;
  children: ReactNode;
};

/** A note inside a section that says what a change will do. */
export function Callout({ tone = "i", icon, action, children }: CalloutProps) {
  return (
    <div className={cn("cal", tone)} role={tone === "w" || tone === "d" ? "alert" : "status"}>
      <Ic n={icon ?? CALLOUT_ICON[tone]} s={13} />
      <div>{children}</div>
      {action}
    </div>
  );
}

type TxtProps = {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  mono?: boolean;
  prefix?: ReactNode;
  suffix?: ReactNode;
  width?: number;
  id?: string;
  label?: string;
  type?: "text" | "date" | "password" | "number";
  autoFocus?: boolean;
  /** Hands the browser's own validation the range a number may take. */
  min?: number;
  max?: number;
};

/** A one-line text box, with an optional unit before or after it. */
export function Txt({
  value,
  onChange,
  placeholder,
  mono = false,
  prefix,
  suffix,
  width,
  id,
  label,
  type = "text",
  autoFocus = false,
  min,
  max,
}: TxtProps) {
  return (
    <label className={cn("inx", mono && "mono")} style={width ? { width } : undefined}>
      {prefix && <span className="inx-a">{prefix}</span>}
      <input
        id={id}
        type={type}
        aria-label={label}
        autoFocus={autoFocus}
        autoComplete={type === "password" ? "new-password" : undefined}
        min={min}
        max={max}
        value={value}
        placeholder={placeholder}
        onChange={(event) => onChange(event.target.value)}
      />
      {suffix && <span className="inx-a">{suffix}</span>}
    </label>
  );
}

type SelProps<T extends string | number> = {
  value: T;
  onChange: (value: T) => void;
  options: readonly (readonly [T, string])[];
  label: string;
  width?: number;
};

/** A native select drawn as a field. */
export function Sel<T extends string | number>({
  value,
  onChange,
  options,
  label,
  width,
}: SelProps<T>) {
  return (
    <label className="inx sel" style={width ? { width } : undefined}>
      <select
        aria-label={label}
        value={String(value)}
        onChange={(event) => {
          const picked = options.find(([key]) => String(key) === event.target.value);
          if (picked) onChange(picked[0]);
        }}
      >
        {options.map(([key, text]) => (
          <option key={String(key)} value={String(key)}>
            {text}
          </option>
        ))}
      </select>
      <Ic n="chevD" s={12} />
    </label>
  );
}

type ChipsProps<T extends string> = {
  value: readonly T[];
  onChange: (value: T[]) => void;
  /** Each choice: its key, its label, and an optional mark after the label. */
  options: readonly (readonly [T, string, ReactNode?])[];
  label: string;
};

/** A set of choices, each a chip that toggles. */
export function Chips<T extends string>({ value, onChange, options, label }: ChipsProps<T>) {
  return (
    <div className="tks" role="group" aria-label={label}>
      {options.map(([key, text, mark]) => {
        const on = value.includes(key);
        return (
          <button
            key={key}
            type="button"
            className={cn("tkb", on && "on")}
            aria-pressed={on}
            onClick={() => onChange(on ? value.filter((entry) => entry !== key) : [...value, key])}
          >
            <Ic n={on ? "check" : "plus"} s={11} w={2.2} />
            {text}
            {mark}
          </button>
        );
      })}
    </div>
  );
}

type SwRowProps = {
  label: string;
  note?: ReactNode;
  on: boolean;
  onChange: (on: boolean) => void;
  disabled?: boolean;
  /** Why it is disabled, shown in place of nothing. */
  why?: string;
};

/** A setting that is a switch: what it is, a line on what it does, and the switch. */
export function SwRow({ label, note, on, onChange, disabled = false, why }: SwRowProps) {
  return (
    <div className={cn("swr", disabled && "dis")}>
      <span>
        <b>{label}</b>
        {note && <em>{note}</em>}
        {disabled && why && <em className="t-w">{why}</em>}
      </span>
      <Switch on={on} label={label} disabled={disabled} onChange={onChange} />
    </div>
  );
}
