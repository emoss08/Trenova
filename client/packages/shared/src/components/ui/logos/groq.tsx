import { cn } from "@trenova/shared/lib/utils";

/**
 * Groq's flash. Drawn in the vendor's own 33-unit grid rather than rescaled, so
 * the corners land exactly where they do on their mark.
 */
export function GroqLogo({ className }: { className?: string }) {
  return (
    <svg
      className={cn("size-4", className)}
      viewBox="0.54 0.39 32 32"
      fill="currentColor"
      role="img"
      aria-hidden="true"
      focusable="false"
    >
      <path d="m18.445 4.406-9.468 13.74 7.341.665-1.69 9.578 9.469-13.74-7.342-.664 1.69-9.579Z" />
    </svg>
  );
}
