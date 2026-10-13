import { describeApiError } from "@/lib/api-error-message";
import { apiService } from "@/services/api";
import { Button } from "@trenova/shared/components/ui/button";
import { NovaMark } from "@trenova/shared/components/ui/nova-mark";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { ScenarioTraceRow } from "@trenova/shared/types/formula-template";
import { ArrowRightIcon } from "@trenova/shared/components/icons";
import { useQuery } from "@tanstack/react-query";
import { NovaButton } from "../ai/nova-button";
import { TickedText } from "../ai/ticked-text";
import { formatMoney, formatStudioValue, numericAmount } from "../model/format";
import { useStudio } from "../studio-context";
import type { ScenarioRun, StudioScenario } from "../use-studio-scenarios";

const MAX_TRACE_ROWS = 3;

function traceValue(row: ScenarioTraceRow, value: string | null | undefined): string {
  if (value === null || value === undefined) return "—";
  const amount = numericAmount(value);
  if (row.kind === "input" || amount === null) return formatStudioValue(value);
  return formatMoney(amount);
}

function signed(amount: number): string {
  return `${amount >= 0 ? "+" : "−"}${formatMoney(Math.abs(amount))}`;
}

/**
 * Why a scenario broke, in the popover on its row. The engine runs the
 * scenario's inputs through the version it was recorded on and through the
 * editor, statement by statement, and Nova says in two sentences what moved
 * and whether it looks meant. The actions follow the verdict.
 */
export function ScenarioWhy({
  scenario,
  run,
  onExpect,
  onClose,
}: {
  scenario: StudioScenario;
  run: ScenarioRun | undefined;
  onExpect: (amount: number) => void;
  onClose: () => void;
}) {
  const t = useT();
  const { template, scenarios, askNova } = useStudio();
  const candidate = scenarios.candidate();

  const why = useQuery({
    queryKey: [
      "formula-scenario-why",
      template?.id,
      scenario.id,
      scenario.expectedAmount,
      candidate,
    ],
    queryFn: ({ signal }) =>
      apiService.formulaTemplateService.explainScenario(
        template?.id ?? "",
        scenario.id,
        candidate,
        { signal },
      ),
    enabled: !!template?.id && scenario.saved !== null,
    staleTime: 60_000,
    retry: false,
  });

  const data = why.data;
  const actual = data?.actual ?? run?.actualAmount ?? null;
  const delta = data?.delta ?? (actual !== null ? actual - scenario.expectedAmount : null);
  const baseline =
    data?.baselineLabel ??
    (scenario.recordedOnVersion ? t("v{0}", scenario.recordedOnVersion) : t("the recording"));
  const recordedBy = data?.recordedBy?.name ?? scenario.recordedBy?.name;
  const verdict = data?.verdict;
  const rows = (data?.rows ?? []).slice(0, MAX_TRACE_ROWS);

  const fixPrompt = t(
    "Scenario “{0}” expects {1} but the formula now bills {2}. {3} Fix the formula so this scenario passes again without breaking the others.",
    scenario.name,
    formatMoney(scenario.expectedAmount),
    actual !== null ? formatMoney(actual) : t("nothing"),
    data?.why ?? run?.error ?? "",
  );
  const fix = () => {
    onClose();
    askNova(fixPrompt);
  };
  const expect =
    actual !== null ? (
      <Button
        type="button"
        size="xs"
        variant={verdict === "intended" ? "default" : "quiet"}
        onClick={() => onExpect(actual)}
      >
        {t("Expect {0}", formatMoney(actual))}
      </Button>
    ) : null;

  return (
    <div className="flex w-full flex-col gap-2.5">
      <div className="flex items-center gap-2">
        <NovaMark spin={why.isFetching ? "always" : "none"} className="size-3.5" />
        {verdict ? (
          <span
            className={cn(
              "text-sm font-semibold",
              verdict === "intended" && "text-success",
              verdict === "bug" && "text-danger",
              verdict === "unclear" && "text-warning",
            )}
          >
            {verdict === "intended"
              ? t("Looks intended")
              : verdict === "bug"
                ? t("Looks like a regression")
                : t("Needs a look")}
          </span>
        ) : (
          <span className="text-sm font-semibold">{t("Why it broke")}</span>
        )}
        <span className="flex-1" />
        {delta !== null && <span className="font-mono text-xs tabular-nums">{signed(delta)}</span>}
      </div>

      {why.isError ? (
        <p className="text-danger text-sm">
          {describeApiError(why.error, t("Nova could not explain this scenario."))}
        </p>
      ) : data ? (
        <p className="text-sm leading-relaxed">
          <TickedText text={data.why} />
        </p>
      ) : (
        <p className="ui-shimmer text-foreground-muted w-fit rounded-sm text-sm">
          {t("Comparing {0} with the editor…", baseline)}
        </p>
      )}

      {rows.length > 0 && (
        <div className="bg-sunken flex flex-col gap-1 rounded-md px-2.5 py-2">
          {rows.map((row) => (
            <div key={`${row.kind}:${row.name}`} className="flex items-center gap-1.5 text-xs">
              <code className="text-foreground-muted min-w-0 flex-1 truncate font-mono">
                {row.name}
              </code>
              <span className="font-mono tabular-nums">{traceValue(row, row.before)}</span>
              <ArrowRightIcon className="text-foreground-subtle size-2.5" />
              <span className="font-mono font-medium tabular-nums">
                {traceValue(row, row.after)}
              </span>
            </div>
          ))}
        </div>
      )}

      <div className="flex items-center gap-1.5">
        {verdict === "intended" ? (
          <>
            {expect}
            <Button type="button" size="xs" variant="quiet" onClick={fix}>
              {t("Fix instead")}
            </Button>
          </>
        ) : (
          <>
            <NovaButton disabled={!data && !why.isError} onClick={fix}>
              {t("Fix with Nova")}
            </NovaButton>
            {expect}
          </>
        )}
        <span className="flex-1" />
        <Tooltip>
          <TooltipTrigger
            render={
              <span className="text-foreground-subtle cursor-default text-2xs">
                {t("vs {0}", baseline)}
              </span>
            }
          />
          <TooltipContent>
            {recordedBy ? t("Recorded by {0}", recordedBy) : t("Who recorded it is not known")}
          </TooltipContent>
        </Tooltip>
      </div>
    </div>
  );
}
