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
    </div>
  );
}
