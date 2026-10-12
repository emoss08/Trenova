import { fileExtension, isImageExtension } from "@/lib/triage/reply-files";
import { cn } from "@trenova/shared/lib/utils";

/**
 * A file drawn as a small page with its extension, coloured by kind: PDF, image or other.
 * `compact` draws the extension as a one-line tag, for rows that hold a single line of text.
 */
export function FileBadge({
  fileName,
  dimmed = false,
  compact = false,
}: {
  fileName: string;
  dimmed?: boolean;
  compact?: boolean;
}) {
  const extension = fileExtension(fileName);

  return (
    <span
      aria-hidden
      className={cn(
        "bg-sunken border-border text-3xs flex shrink-0 justify-center rounded-sm border font-mono font-semibold uppercase",
        compact ? "h-4.5 w-8 items-center" : "h-8 w-7 items-end pb-1",
        extension === "pdf"
          ? "text-danger"
          : isImageExtension(extension)
            ? "text-brand"
            : "text-foreground-subtle",
        dimmed && "opacity-50",
      )}
    >
      {extension}
    </span>
  );
}
