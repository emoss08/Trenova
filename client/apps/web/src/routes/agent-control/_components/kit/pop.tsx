import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useRef, type ReactNode } from "react";

type PopProps = {
  onClose: () => void;
  className?: string;
  label: string;
  children: ReactNode;
};

/**
 * A popover dropped from the control beside it. It closes on Escape or a click anywhere
 * outside it, and Escape stops there rather than closing what it sits in.
 */
export function Pop({ onClose, className, label, children }: PopProps) {
  const root = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const onDown = (event: MouseEvent) => {
      if (!root.current?.contains(event.target as Node)) {
        onClose();
      }
    };
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault();
        event.stopPropagation();
        onClose();
      }
    };
    const timer = window.setTimeout(() => document.addEventListener("mousedown", onDown));
    window.addEventListener("keydown", onKey, true);
    return () => {
      window.clearTimeout(timer);
      document.removeEventListener("mousedown", onDown);
      window.removeEventListener("keydown", onKey, true);
    };
  }, [onClose]);

  return (
    <div ref={root} className={cn("pop", className)} role="dialog" aria-label={label}>
      {children}
    </div>
  );
}
