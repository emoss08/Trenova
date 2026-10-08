import { cn } from "@trenova/shared/lib/utils";

export function VLLMLogo({ className }: { className?: string }) {
  return (
    <svg
      className={cn("size-4", className)}
      viewBox="0 0 24 24"
      role="img"
      aria-hidden="true"
      focusable="false"
    >
      <path d="M0 4.973h9.324V23L0 4.973z" fill="#FDB515" />
      <path d="M13.986 4.351L22.378 0l-6.216 23H9.324l4.662-18.649z" fill="#30A2FF" />
    </svg>
  );
}
