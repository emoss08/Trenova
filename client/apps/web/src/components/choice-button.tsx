import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";

type ChoiceButtonProps = {
  selected: boolean;
  onClick: () => void;
  disabled?: boolean;
  className?: string;
  children: ReactNode;
};

export function ChoiceButton({
  selected,
  onClick,
  disabled,
  className,
  children,
}: ChoiceButtonProps) {
  return (
    <button
      type="button"
      role="radio"
      aria-checked={selected}
      onClick={onClick}
      disabled={disabled}
      className={cn(
        "ui-focus-ring flex flex-1 cursor-pointer flex-col items-start gap-0.5 rounded-md border px-3 py-2 text-left text-sm transition-colors disabled:cursor-not-allowed disabled:opacity-60",
        selected
          ? "border-brand-border bg-surface-selected"
          : "border-border hover:border-border-strong hover:bg-surface-hover",
        className,
      )}
    >
      {children}
    </button>
  );
}
