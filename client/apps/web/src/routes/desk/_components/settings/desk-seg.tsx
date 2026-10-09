import { Button } from "@trenova/shared/components/ui/button";
import { cn } from "@trenova/shared/lib/utils";
import type { CSSProperties } from "react";
import { onRadioArrows } from "../use-modal-focus";

/** A row of mutually exclusive choices, the chosen one raised. */
export function Seg<V extends string>({
  value,
  options,
  onChange,
  label,
  disabled = false,
  small = false,
}: {
  value: V;
  options: Array<[V, string]>;
  onChange: (value: V) => void;
  label: string;
  /** Read-only: shown, but not changeable. */
  disabled?: boolean;
  /** A smaller track, for a control repeated down a list. */
  small?: boolean;
}) {
  const chosen = options.some(([option]) => option === value);
  return (
    <div
      className={cn("dk-sx-seg", small && "dk-ck-sseg")}
      role="radiogroup"
      aria-label={label}
      style={
        {
          "--n": options.length,
          "--i": Math.max(
            0,
            options.findIndex(([option]) => option === value),
          ),
        } as CSSProperties
      }
    >
      <span className="dk-sx-kn" />
      {options.map(([option, text], index) => (
        <Button
          key={option}
          variant="bare"
          size="bare"
          role="radio"
          aria-checked={value === option}
          tabIndex={value === option || (!chosen && index === 0) ? 0 : -1}
          disabled={disabled}
          className="h-7 rounded-md px-3 text-sm disabled:cursor-default disabled:opacity-60 whitespace-nowrap text-dsk-subtle transition-colors duration-150 hover:text-dsk-fg aria-checked:bg-dsk-card aria-checked:font-medium aria-checked:text-dsk-fg aria-checked:ring-1 aria-checked:ring-dsk-b"
          onClick={() => onChange(option)}
          onKeyDown={(event) =>
            onRadioArrows(
              event,
              options.map(([choice]) => choice),
              value,
              onChange,
            )
          }
        >
          {text}
        </Button>
      ))}
    </div>
  );
}
