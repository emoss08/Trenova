import { fileExtension, isImageExtension } from "@/lib/triage/reply-files";
import { cn } from "@trenova/shared/lib/utils";

/** A file drawn as a small page with its extension, coloured by kind: PDF, image or other. */
export function FileBadge({ fileName, dimmed = false }: { fileName: string; dimmed?: boolean }) {
  const extension = fileExtension(fileName);

  return (
    <span
      aria-hidden
      className={cn(
        "bg-sunken border-border flex h-8 w-7 shrink-0 items-end justify-center rounded-sm border pb-1 font-mono text-3xs font-semibold uppercase",
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
