import { describeApiError } from "@/lib/api-error-message";
import { apiService } from "@/services/api";
import type { PricedScenario } from "@/types/page-draft";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { NovaMark } from "@trenova/shared/components/ui/nova-mark";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  CheckIcon,
  InfoCircleIcon,
  PlusIcon,
  Trash01Icon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import { useMutation } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";
import { NovaButton } from "../ai/nova-button";
import { ProposedScenariosCard, scenarioInputsLine } from "../ai/proposed-scenarios-card";
import { formatMoney, previewAmounts } from "../model/format";
import { useStudio } from "../studio-context";
import type { StudioScenario } from "../use-studio-scenarios";
import { ScenarioWhy } from "./scenario-why";

function AddScenarioForm({ onDone }: { onDone: () => void }) {
  const t = useT();
  const { preview, scenarios } = useStudio();
  const [name, setName] = useState("");
  const [saving, setSaving] = useState(false);
  const amounts = previewAmounts(preview.result);
  const changed = Object.keys(preview.overrides).length;

  const save = async () => {
    if (!name.trim() || !amounts) return;
    setSaving(true);
    try {
      await scenarios.add([
        { name: name.trim(), variables: preview.loadValues(), expectedAmount: amounts.billed },
      ]);
      onDone();
    } catch (error) {
      toast.error(t("Could not save the scenario"), {
        description: error instanceof Error ? error.message : undefined,
      });
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="bg-card flex flex-col gap-2 rounded-lg border p-3">
      <Input
        autoFocus
        value={name}
        maxLength={100}
        onChange={(event) => setName(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === "Enter") {
            event.preventDefault();
            void save();
          }
          if (event.key === "Escape") onDone();
        }}
        aria-label={t("Scenario name")}
        placeholder={t("Scenario name, e.g. Short haul hits the minimum")}
      />
      <div className="text-foreground-muted flex items-center justify-between gap-2 text-xs">
        <span>
          {changed > 0
            ? t("Uses the preview's values ({0} changed) and expects", changed)
            : t("Uses the preview's values and expects")}
        </span>
        <span className="text-foreground font-mono tabular-nums">
          {amounts ? formatMoney(amounts.billed) : "—"}
        </span>
      </div>
      <div className="flex justify-end gap-1.5">
        <Button type="button" variant="quiet" size="xs" onClick={onDone}>
          {t("Cancel")}
        </Button>
        <Button
          type="button"
          size="xs"
          disabled={!name.trim() || !amounts}
          isLoading={saving}
          onClick={() => void save()}
        >
          {t("Save scenario")}
        </Button>
      </div>
    </div>
  );
}

function ScenarioRow({ scenario, onDelete }: { scenario: StudioScenario; onDelete: () => void }) {
  const t = useT();
  const { scenarios, analysis } = useStudio();
  const [whyOpen, setWhyOpen] = useState(false);
  const run = scenarios.runs.get(scenario.id);
  const passed = run?.passed ?? false;
  const known = run !== undefined;
  const inputs = scenarioInputsLine(scenario.variables, analysis.reads) || t("Sample load");

  const expect = async (amount: number) => {
    setWhyOpen(false);
    try {
      await scenarios.accept(scenario.id, amount);
      toast(t("“{0}” now expects {1}", scenario.name, formatMoney(amount)));
    } catch (error) {
      toast.error(t("Could not update the scenario"), { description: describeApiError(error) });
    }
  };

  return (
    <div
      className={cn(
        "group flex items-start gap-2.5 border-b px-3 py-2.5 last:border-b-0",
        scenarios.freshIds.has(scenario.id) && "animate-rise",
        scenarios.isStale && "opacity-70",
      )}
    >
      <span
        aria-hidden
        className={cn(
          "mt-0.5 flex size-4 shrink-0 items-center justify-center rounded-full",
          !known && "bg-surface-hover",
          known && passed && "bg-success-subtle text-success-subtle-foreground",
          known && !passed && "bg-danger-subtle text-danger-subtle-foreground",
        )}
      >
        {known &&
          (passed ? (
            <CheckIcon className="size-2.5" strokeWidth={2.4} />
          ) : (
            <XCloseIcon className="size-2.5" strokeWidth={2.4} />
          ))}
      </span>
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span className="truncate text-sm font-medium">{scenario.name}</span>
        <span className="text-foreground-subtle truncate font-mono text-2xs">{inputs}</span>
      </div>
      <div className="flex shrink-0 flex-col items-end gap-0.5">
        {!known || passed ? (
          <span className="font-mono text-sm tabular-nums">
            {formatMoney(run?.actualAmount ?? scenario.expectedAmount)}
          </span>
        ) : (
          <>
            <span className="text-danger font-mono text-sm tabular-nums">
              {run.actualAmount === null ? "—" : formatMoney(run.actualAmount)}
            </span>
            <s className="text-foreground-subtle font-mono text-2xs tabular-nums">
              {formatMoney(scenario.expectedAmount)}
            </s>
          </>
        )}
        <span className="flex items-center gap-1">
          {known && !passed && scenario.saved && (
            <Popover open={whyOpen} onOpenChange={setWhyOpen}>
              <PopoverTrigger
                render={
                  <Button
                    type="button"
                    variant="outline"
                    size="xxs"
                    className="fs-nova-btn gap-1"
                    aria-label={t("Why did {0} break?", scenario.name)}
                  >
                    <NovaMark className="fs-nova-ring size-2.75" />
                    {t("Why?")}
                  </Button>
                }
              />
              <PopoverContent align="end" className="w-85 p-3">
                <ScenarioWhy
                  scenario={scenario}
                  run={run}
                  onExpect={(amount) => void expect(amount)}
                  onClose={() => setWhyOpen(false)}
                />
              </PopoverContent>
            </Popover>
          )}
          {known && !passed && !scenario.saved && run.actualAmount !== null && (
            <Button
              type="button"
              variant="quiet"
              size="xxs"
              onClick={() => void expect(run.actualAmount ?? 0)}
            >
              {t("Expect {0}", formatMoney(run.actualAmount))}
            </Button>
          )}
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            className="opacity-0 transition-opacity group-hover:opacity-100 focus-visible:opacity-100"
            aria-label={t("Delete scenario {0}", scenario.name)}
            onClick={onDelete}
          >
            <Trash01Icon className="size-3" />
          </Button>
        </span>
      </div>
    </div>
  );
}

/**
 * The scenarios: loads with the charge they must keep producing, run against
 * the editor on every change. A failing one says why in a popover, grounded in
 * the engine's trace of the version it was recorded on against the draft.
 */
export function ScenariosPane() {
  const t = useT();
  const { scenarios, mode, analysis, preview, draftContent } = useStudio();
  const [adding, setAdding] = useState(false);
  const [suggestions, setSuggestions] = useState<PricedScenario[] | null>(null);
  const [pendingDelete, setPendingDelete] = useState<StudioScenario | null>(null);
  const amounts = previewAmounts(preview.result);
  const { total, passed } = scenarios;

  const suggest = useMutation({
    mutationFn: () =>
      apiService.formulaTemplateService.suggestScenarios(
        draftContent(),
        scenarios.scenarios.map((scenario) => ({
          name: scenario.name,
          variables: scenario.variables,
        })),
      ),
    onSuccess: setSuggestions,
    onError: (error) =>
      toast.error(t("Nova could not suggest scenarios"), { description: describeApiError(error) }),
  });

  const remove = useMutation({
    mutationFn: (scenario: StudioScenario) => scenarios.remove(scenario.id),
    onSuccess: () => {
      toast(t("Scenario deleted"));
      setPendingDelete(null);
    },
    onError: (error) =>
      toast.error(t("Failed to delete scenario"), { description: describeApiError(error) }),
  });

  return (
    <div className="flex flex-col gap-3 p-3">
      <div className="flex items-center gap-2">
        <div className="flex items-baseline gap-2">
          <span
            className={cn(
              "font-mono text-2xl tabular-nums",
              total > 0 && passed === total && "text-success",
              total > 0 && passed < total && "text-danger",
            )}
          >
            {passed}
            <span className="text-foreground-subtle text-base">/{total}</span>
          </span>
          <span className="text-foreground-muted text-xs">
            {total === 0
              ? t("No scenarios yet")
              : passed === total
                ? t("Every scenario passes")
                : t("{0} failing · approval is blocked", total - passed)}
          </span>
        </div>
        <span className="flex-1" />
        <NovaButton
          busy={suggest.isPending}
          disabled={analysis.empty || suggest.isPending}
          onClick={() => suggest.mutate()}
        >
          {suggest.isPending ? t("Thinking…") : t("Suggest edge cases")}
        </NovaButton>
        <Button
          type="button"
          variant="outline"
          size="xs"
          className="gap-1"
          disabled={!amounts}
          onClick={() => setAdding(true)}
        >
          <PlusIcon className="size-3" />
          {t("Add")}
        </Button>
      </div>

      {mode === "create" && (
        <Alert variant="info" size="sm">
          <InfoCircleIcon />
          <AlertDescription>
            {t("Scenarios you add here are saved with the template when you create it.")}
          </AlertDescription>
        </Alert>
      )}

      {adding && <AddScenarioForm onDone={() => setAdding(false)} />}

      {suggestions && suggestions.length > 0 && <ProposedScenariosCard scenarios={suggestions} />}

      {scenarios.isLoading ? (
        <div className="flex flex-col gap-2">
          <Skeleton className="h-12 w-full" />
          <Skeleton className="h-12 w-full" />
        </div>
      ) : total > 0 ? (
        <div className="bg-card overflow-hidden rounded-lg border">
          {scenarios.scenarios.map((scenario) => (
            <ScenarioRow
              key={scenario.id}
              scenario={scenario}
              onDelete={() =>
                scenario.saved ? setPendingDelete(scenario) : remove.mutate(scenario)
              }
            />
          ))}
        </div>
      ) : (
        <p className="text-foreground-muted rounded-lg border border-dashed px-3 py-4 text-center text-sm">
          {t(
            "Pin the preview or ask Nova for edge cases. Each scenario pins what the formula must keep charging; approval needs every one to pass.",
          )}
        </p>
      )}

      {total > 0 && (
        <p className="text-foreground-subtle text-2xs leading-relaxed">
          {t(
            "Scenarios re-run against the editor on every change, saved or not. A failing scenario blocks approval until it passes or is deleted.",
          )}
        </p>
      )}

      <AlertDialog
        open={pendingDelete !== null}
        onOpenChange={(open) => {
          if (!open) setPendingDelete(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {t("Delete scenario “{0}”?", pendingDelete?.name ?? "")}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t("This scenario will no longer gate approval of the template.")}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel variant="outline" size="default">
              {t("Cancel")}
            </AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              size="default"
              isLoading={remove.isPending}
              onClick={() => pendingDelete && remove.mutate(pendingDelete)}
            >
              {t("Delete")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
