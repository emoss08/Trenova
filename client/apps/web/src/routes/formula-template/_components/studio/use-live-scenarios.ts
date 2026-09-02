import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type {
  FormulaTemplateFormValues,
  FormulaTestCase,
  RunTestCasesResponse,
  TestCaseCandidate,
} from "@trenova/shared/types/formula-template";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useCallback, useEffect, useRef, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";

const SCENARIO_DEBOUNCE_MS = 600;

export type LiveScenariosState = {
  scenarios: FormulaTestCase[] | undefined;
  isLoading: boolean;
  results: RunTestCasesResponse | null;
  /** A run is in flight. */
  isPending: boolean;
  /** The editor changed since the results were produced. */
  isStale: boolean;
  runNow: () => void;
};

/**
 * Keeps saved scenarios evaluated against what is in the editor right now.
 *
 * Scenarios are the approval gate, so a stale "3/3 passing" is worse than no
 * badge at all. Every edit that changes the candidate content re-runs them on
 * the same debounce the preview uses, and a scenario being added, edited, or
 * deleted re-runs them as soon as the list refetches.
 */
export function useLiveScenarios(templateId: string | null): LiveScenariosState {
  const { control } = useFormContext<FormulaTemplateFormValues>();

  const expression = useWatch({ control, name: "expression" });
  const variableDefinitions = useWatch({ control, name: "variableDefinitions" });
  const breakdownDefinitions = useWatch({ control, name: "breakdownDefinitions" });
  const minCharge = useWatch({ control, name: "minCharge" });
  const maxCharge = useWatch({ control, name: "maxCharge" });
  const roundingMode = useWatch({ control, name: "roundingMode" });
  const roundingPrecision = useWatch({ control, name: "roundingPrecision" });

  const { data: scenarios, isLoading } = useQuery({
    ...queries.formulaTemplate.testCases(templateId ?? ""),
    enabled: !!templateId,
  });

  const [results, setResults] = useState<RunTestCasesResponse | null>(null);
  const [isStale, setIsStale] = useState(false);

  const { mutate, isPending } = useMutation({
    mutationFn: (candidate: TestCaseCandidate) => {
      if (!templateId) throw new Error("Template not saved yet");
      return apiService.formulaTemplateService.runTestCases(templateId, candidate);
    },
    onSuccess: (response) => {
      setResults(response);
      setIsStale(false);
    },
    onError: () => {
      setIsStale(true);
    },
  });

  const buildCandidate = useCallback(
    (): TestCaseCandidate => ({
      expression: expression ?? "",
      variableDefinitions,
      breakdownDefinitions,
      minCharge: minCharge ?? null,
      maxCharge: maxCharge ?? null,
      roundingMode,
      roundingPrecision,
    }),
    [
      expression,
      variableDefinitions,
      breakdownDefinitions,
      minCharge,
      maxCharge,
      roundingMode,
      roundingPrecision,
    ],
  );

  const buildCandidateRef = useRef(buildCandidate);
  buildCandidateRef.current = buildCandidate;

  const canRun = !!templateId && (scenarios?.length ?? 0) > 0 && !!expression?.trim();

  const runNow = useCallback(() => {
    if (!canRun) return;
    mutate(buildCandidateRef.current());
  }, [canRun, mutate]);

  useEffect(() => {
    if (!templateId || !scenarios || scenarios.length === 0) {
      setResults(null);
      setIsStale(false);
      return;
    }
    if (!expression?.trim()) {
      return;
    }

    setIsStale(true);
    const timer = window.setTimeout(() => {
      mutate(buildCandidate());
    }, SCENARIO_DEBOUNCE_MS);

    return () => window.clearTimeout(timer);
  }, [templateId, scenarios, expression, buildCandidate, mutate]);

  return {
    scenarios,
    isLoading,
    results,
    isPending,
    isStale,
    runNow,
  };
}
