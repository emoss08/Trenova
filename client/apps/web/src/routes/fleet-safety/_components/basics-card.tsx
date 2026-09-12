import { useT } from "@trenova/shared/i18n/use-t";
import { basicStandings } from "@/lib/fleet-safety-console";
import type { FleetSafetyBasicRow } from "@/lib/graphql/fleet-safety";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { csaBasicHint, csaBasicLabel } from "@trenova/shared/lib/csa";
import { cn } from "@trenova/shared/lib/utils";
import { InfoIcon, ShieldAlertIcon } from "lucide-react";
import { m, useReducedMotion } from "motion/react";
import { useMemo } from "react";

type BasicsCardProps = {
  basics: readonly FleetSafetyBasicRow[];
  inferred: boolean;
};

/**
 * The seven CSA BASICs in the agency's own order, every one of them, zero or
 * not. Bars are relative to this fleet's own worst category: a national
 * percentile needs peer data the system does not have, and calling a
 * relative bar a percentile would be a lie a safety director would catch.
 */
export function BasicsCard({ basics, inferred }: BasicsCardProps) {
  const t = useT();

  const standings = useMemo(() => basicStandings(basics), [basics]);
  const reduceMotion = useReducedMotion();

  return (
    <section aria-labelledby="basics-heading" className="bg-card overflow-hidden rounded-lg border">
      <header className="flex flex-wrap items-center justify-between gap-2 border-b px-3 py-2">
        <div className="flex items-center gap-2">
          <ShieldAlertIcon className="text-muted-foreground size-3.5" aria-hidden />
          <h3 id="basics-heading" className="text-sm font-medium">
            {t("CSA BASICs")}
          </h3>
        </div>
        <Tooltip>
          <TooltipTrigger
            render={
              <span className="text-muted-foreground flex items-center gap-1 text-xs">
                <InfoIcon className="size-3" aria-hidden />
                {t("How the score is made")}
              </span>
            }
          />
          <TooltipContent className="max-w-72">
            {t("Severity, plus two for an out-of-service order, weighted three times inside six months and twice inside a year. Bars are relative to this fleet's own worst category, not to a national percentile.")}
          </TooltipContent>
        </Tooltip>
      </header>

      {inferred ? (
        <Alert variant="warning" className="rounded-none border-x-0 border-t-0">
          <AlertDescription>
            {t("Some categories were reached from the kind of event rather than from violations somebody keyed in. Record the violation codes off an inspection report and these become the real thing.")}
          </AlertDescription>
        </Alert>
      ) : null}

      <ul className="divide-y">
        {standings.map(({ basic, tone, share }, index) => (
          <li
            key={basic.basic}
            className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 px-3 py-2"
          >
            <div className="flex min-w-0 flex-col gap-1.5">
              <span className="flex min-w-0 flex-wrap items-center gap-2">
                <Tooltip>
                  <TooltipTrigger
                    render={
                      <span
                        className={cn(
                          "truncate text-sm",
                          tone === "muted" ? "text-muted-foreground" : "font-medium",
                        )}
                      />
                    }
                  >
                    {csaBasicLabel(basic.basic)}
                  </TooltipTrigger>
                  <TooltipContent className="max-w-64">{csaBasicHint(basic.basic)}</TooltipContent>
                </Tooltip>
                {tone === "critical" ? <Badge variant="inactive">{t("Highest")}</Badge> : null}
                {tone === "warning" ? <Badge variant="warning">{t("Elevated")}</Badge> : null}
                {basic.inferred ? <Badge variant="secondary">{t("Inferred")}</Badge> : null}
                {basic.outOfService > 0 ? (
                  <Badge variant="outline" className="tabular-nums">
                    {t("{0} OOS", basic.outOfService)}
                  </Badge>
                ) : null}
              </span>
              <span
                role="img"
                aria-label={`${csaBasicLabel(basic.basic)}: ${basic.weightedScore} weighted`}
                className="bg-muted flex h-1.5 w-full max-w-md overflow-hidden rounded-full"
              >
                <m.span
                  aria-hidden
                  className={cn(
                    "h-full rounded-full",
                    tone === "critical"
                      ? "bg-brand"
                      : tone === "warning"
                        ? "bg-brand/60"
                        : "bg-brand/30",
                  )}
                  initial={reduceMotion ? false : { width: 0 }}
                  animate={{ width: `${share}%` }}
                  transition={{ duration: 0.5, delay: index * 0.04, ease: "easeOut" }}
                />
              </span>
            </div>
            <span className="text-right text-xs tabular-nums">
              <span className="font-mono font-medium">{basic.weightedScore}</span>
              <span className="text-muted-foreground">
                {basic.violations > 0
                  ? t("· {0} violation{1}", basic.violations, basic.violations === 1 ? "" : "s")
                  : basic.events > 0
                    ? t("· {0} event{1}", basic.events, basic.events === 1 ? "" : "s")
                    : ""}
              </span>
            </span>
          </li>
        ))}
      </ul>
    </section>
  );
}
