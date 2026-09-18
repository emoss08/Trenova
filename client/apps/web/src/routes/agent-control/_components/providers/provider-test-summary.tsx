import { useT } from "@trenova/shared/i18n/use-t";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { formatSecondsAgo } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { AIProviderTestOutcome } from "@/lib/graphql/ai-provider";
import { AlertTriangleIcon, CheckCircle2Icon, CircleDashedIcon, XCircleIcon } from "lucide-react";
import { useEffect, useState } from "react";

const CLOCK_TICK_MS = 60_000;

function useNowSeconds(): number {
  const [now, setNow] = useState(() => Math.floor(Date.now() / 1000));

  useEffect(() => {
    const timer = window.setInterval(() => setNow(Math.floor(Date.now() / 1000)), CLOCK_TICK_MS);
    return () => window.clearInterval(timer);
  }, []);

  return now;
}

type ProviderTestSummaryProps = {
  outcome: AIProviderTestOutcome | null | undefined;
  className?: string;
};

/**
 * A successful connection is not the same as a usable one. An endpoint that
 * answers but ignores the JSON schema will fail later, deep inside a billing
 * diagnosis, so that case is a warning rather than a success.
 */
export function ProviderTestSummary({ outcome, className }: ProviderTestSummaryProps) {
  const t = useT();
  const now = useNowSeconds();

  if (!outcome) {
    return (
      <p className={cn("text-muted-foreground flex items-center gap-1.5 text-xs", className)}>
        <CircleDashedIcon className="size-3.5 shrink-0" />
        {t("Never tested")}
      </p>
    );
  }

  const tone = !outcome.success ? "error" : outcome.schemaHonoured ? "ok" : "warn";
  const Icon = { ok: CheckCircle2Icon, warn: AlertTriangleIcon, error: XCircleIcon }[tone];
  const toneClass = {
    ok: "text-success-foreground",
    warn: "text-warning-foreground",
    error: "text-destructive",
  }[tone];

  const ago = formatSecondsAgo(Math.max(0, now - outcome.testedAt));
  const parts = [t("Tested {0}", ago)];
  if (outcome.success) {
    parts.push(`${outcome.latencyMs} ms`);
    parts.push(outcome.schemaHonoured ? t("Schema honoured") : t("Schema not honoured"));
  }

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <p className={cn("flex items-center gap-1.5 text-xs", className)}>
            <Icon className={cn("size-3.5 shrink-0", toneClass)} />
            <span className="text-muted-foreground truncate">{parts.join(" · ")}</span>
          </p>
        }
      />
      <TooltipContent className="max-w-xs">
        <p className="font-medium">{outcome.message}</p>
        {outcome.detail && <p className="text-muted-foreground mt-1 text-xs">{outcome.detail}</p>}
        {outcome.modelIdentifier && (
          <p className="text-muted-foreground mt-1 font-mono text-xs">
            {outcome.modelIdentifier}
          </p>
        )}
      </TooltipContent>
    </Tooltip>
  );
}
