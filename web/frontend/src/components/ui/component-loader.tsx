import { cn } from "@/lib/utils";
import { faSpinner } from "@fortawesome/pro-duotone-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";

export function ComponentLoader({ className }: { className?: string }) {
  return (
    <div
      className={cn("flex flex-col items-center justify-center p-2", className)}
    >
      <FontAwesomeIcon
        icon={faSpinner}
        size="1x"
        className="text-primary motion-safe:animate-spin"
      />
      <p className="text-foreground mt-2 text-sm">Loading data...</p>
      <p className="text-muted-foreground mt-2 text-sm">
        If this takes too long, please refresh the page.
      </p>
    </div>
  );
}
