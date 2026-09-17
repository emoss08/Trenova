import { cn } from "@trenova/shared/lib/utils";

export function VLLMLogo({ className }: { className?: string }) {
  return (
    <svg
      className={cn("size-4", className)}
      viewBox="0 0 24 24"
      fill="currentColor"
      role="img"
      aria-hidden="true"
      focusable="false"
    >
      <path d="m23.6 0-8.721 4.59L9.829 24h7.41zM9.83 24V5.142H.4Z" />
    </svg>
  );
}
