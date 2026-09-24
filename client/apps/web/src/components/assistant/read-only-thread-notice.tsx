import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { LockIcon } from "lucide-react";

/**
 * What stands where the composer was, in a conversation whose reader may no
 * longer ask its agent anything. It sits where the box sat, floating over the
 * foot of the thread the same way, so the transcript above reads to its end
 * and is padded clear of it by the same measurement.
 *
 * It is a statement rather than an error: nothing went wrong, the
 * conversation is simply a record now. It says who can change that, and
 * nothing on it offers to send.
 */
export function ReadOnlyThreadNotice({
  compact = false,
  ref,
}: {
  compact?: boolean;
  /** Measured by the thread, as the composer is, so the last message is never hidden. */
  ref?: React.Ref<HTMLDivElement>;
}) {
  const t = useT();

  return (
    <div ref={ref} className="pointer-events-none absolute inset-x-0 bottom-0 z-10">
      <div
        aria-hidden
        className={cn(
          "from-popover pointer-events-none bg-gradient-to-t to-transparent",
          compact ? "h-6" : "h-10",
        )}
      />
      <div
        className={cn(
          "bg-popover pointer-events-auto",
          compact ? "px-3 pt-0.5 pb-1.5" : "px-4 pt-0.5 pb-2.5",
        )}
      >
        <div className={cn("mx-auto", !compact && "max-w-3xl")}>
          <Alert size="sm" role="status" data-testid="read-only-thread-notice">
            <LockIcon aria-hidden />
            <AlertDescription>
              {t(
                "You no longer have access to this agent. An administrator can give one of your roles access to it.",
              )}
            </AlertDescription>
          </Alert>
        </div>
      </div>
    </div>
  );
}
