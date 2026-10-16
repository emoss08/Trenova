import type { PricedScenario } from "@/types/page-draft";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { Beaker02Icon, CheckIcon, PlusIcon } from "@trenova/shared/components/icons";
import { useState } from "react";
import { toast } from "@trenova/shared/components/ui/toast";
import { formatMoney, formatStudioValue } from "../model/format";
import { useStudio } from "../studio-context";
import { scenarioAmount } from "./proposed-scenarios";

export function scenarioInputsLine(variables: Record<string, unknown>, names?: readonly string[]) {
  const entries = Object.entries(variables).filter(([name]) => !names || names.includes(name));
  return entries.map(([name, value]) => `${name} ${formatStudioValue(value)}`).join(" · ");
}

/**
 * Loads Nova proposed to test the formula, each priced by the engine on the
 * editor's formula. Adding one saves it as a scenario that keeps running as
 * the formula changes; on a template being created it is held until create.
 */
export function ProposedScenariosCard({ scenarios: proposed }: { scenarios: PricedScenario[] }) {
  const t = useT();
  const { scenarios, mode } = useStudio();
  const [added, setAdded] = useState<ReadonlySet<string>>(() => new Set());
  const [busy, setBusy] = useState(false);
  const pending = proposed.filter(
    (scenario) => scenarioAmount(scenario) !== null && !added.has(scenario.name),
  );

  const add = async (list: PricedScenario[]) => {
    setBusy(true);
    try {
      await scenarios.add(
        list.map((scenario) => ({
          name: scenario.name,
          description: scenario.description,
          variables: scenario.variables,
          expectedAmount: scenarioAmount(scenario) ?? 0,
        })),
      );
      setAdded((previous) => new Set([...previous, ...list.map((scenario) => scenario.name)]));
    } catch (error) {
      toast.error(t("Could not add the scenario"), {
        description: error instanceof Error ? error.message : undefined,
      });
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="bg-card flex flex-col overflow-hidden rounded-lg border">
      <div className="flex items-center gap-2 border-b px-3 py-2">
        <Beaker02Icon className="text-foreground-subtle size-3.5" />
        <span className="text-sm font-semibold">{t("Proposed scenarios")}</span>
        <span className="flex-1" />
        {pending.length > 1 && (
          <Button
            type="button"
            variant="ghost"
            size="xs"
            className="gap-1"
            disabled={busy}
            onClick={() => void add(pending)}
          >
            <PlusIcon className="size-3" />
            {t("Add all")}
          </Button>
        )}
      </div>
      {proposed.map((scenario) => {
        const amount = scenarioAmount(scenario);
        const isAdded = added.has(scenario.name);
        return (
          <div
            key={scenario.name}
            className="flex items-center gap-2 border-b px-3 py-2 last:border-b-0"
          >
            <div className="flex min-w-0 flex-1 flex-col gap-0.5">
              <span className="truncate text-sm font-medium">{scenario.name}</span>
              <span className="text-foreground-subtle truncate font-mono text-2xs">
                {scenarioInputsLine(scenario.variables)}
              </span>
            </div>
            {amount !== null ? (
              <span className="font-mono text-sm tabular-nums">{formatMoney(amount)}</span>
            ) : (
              <span className="text-danger max-w-28 truncate text-xs" title={scenario.error}>
                {scenario.error || t("Could not be priced")}
              </span>
            )}
            <Button
              type="button"
              variant={isAdded ? "ghost" : "outline"}
              size="xs"
              className="gap-1"
              disabled={amount === null || isAdded || busy}
              onClick={() => void add([scenario])}
            >
              {isAdded ? <CheckIcon className="size-3" /> : <PlusIcon className="size-3" />}
              {isAdded ? t("Added") : t("Add")}
            </Button>
          </div>
        );
      })}
      <p className="text-foreground-subtle bg-sunken px-3 py-2 text-2xs">
        {mode === "create"
          ? t(
              "Expected amounts are the engine's, from the formula in the editor. They are saved with the template when you create it.",
            )
          : t(
              "Expected amounts are the engine's, from the formula in the editor. Added scenarios keep running as you edit.",
            )}
      </p>
    </div>
  );
}
