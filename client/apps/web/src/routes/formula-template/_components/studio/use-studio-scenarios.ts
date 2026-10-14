import { describeApiError } from "@/lib/api-error-message";
import { formulaTestRequest } from "@/lib/formula-test-request";
import { formulaTemplate } from "@/lib/queries/formula-template";
import { apiService } from "@/services/api";
import type {
  FormulaTemplateFormValues,
  FormulaTestCase,
  FormulaTestCaseInput,
  TestCaseCandidate,
} from "@trenova/shared/types/formula-template";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { nanoid } from "nanoid";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";

const SCENARIO_DEBOUNCE_MS = 600;
/** A cent: what a scenario the engine priced should match exactly. */
export const SCENARIO_TOLERANCE = 0.01;

export type StudioScenario = {
  id: string;
  name: string;
  description: string;
  variables: Record<string, unknown>;
  expectedAmount: number;
  tolerance: number;
  recordedOnVersion: number | null;
  recordedBy: { id: string; name: string } | null;
  /** The saved test case; null while the scenario is held for a template not yet created. */
  saved: FormulaTestCase | null;
  /** For a held scenario, the editor content its expected amount was priced on. */
  recordedContent?: TestCaseCandidate;
};

export type ScenarioRun = {
  passed: boolean;
  actualAmount: number | null;
  error: string | null;
};

export type NewScenario = {
  name: string;
  description?: string;
  variables: Record<string, unknown>;
  expectedAmount: number;
};

export type StudioScenariosState = {
  scenarios: StudioScenario[];
  isLoading: boolean;
  runs: ReadonlyMap<string, ScenarioRun>;
  passed: number;
  total: number;
  /** A run is in flight. */
  isPending: boolean;
  /** The editor changed since the results were produced. */
  isStale: boolean;
  /** Ids added in this session, drawn as new. */
  freshIds: ReadonlySet<string>;
  add: (scenarios: NewScenario[]) => Promise<number>;
  remove: (id: string) => Promise<void>;
  /** Records the draft's amount as what the scenario expects now. */
  accept: (id: string, amount: number) => Promise<void>;
  /** The scenarios a template not yet created carries into its create request. */
  heldForCreate: () => FormulaTestCaseInput[];
  candidate: () => TestCaseCandidate;
};

function fromSaved(testCase: FormulaTestCase): StudioScenario {
  return {
    id: testCase.id,
    name: testCase.name,
    description: testCase.description,
    variables: testCase.variables,
    expectedAmount: testCase.expectedAmount,
    tolerance: testCase.tolerance,
    recordedOnVersion: testCase.recordedOnVersion ?? null,
    recordedBy: testCase.recordedBy
      ? { id: testCase.recordedBy.id, name: testCase.recordedBy.name }
      : null,
    saved: testCase,
  };
}

function amountOf(value: unknown): number | null {
  const amount = typeof value === "number" ? value : Number(value);
  return Number.isFinite(amount) ? amount : null;
}

/**
 * The template's scenarios, kept evaluated against what is in the editor.
 *
 * A saved template's scenarios live on the server and run through its
 * scenario runner. A template being created holds its scenarios here until
 * it is created, prices each one with the engine the same way, and hands
 * them to the create request so they are saved in the same transaction.
 */
export function useStudioScenarios({
  templateId,
  author,
}: {
  templateId: string | null;
  author: { id: string; name: string } | null;
}): StudioScenariosState {
  const { control, getValues } = useFormContext<FormulaTemplateFormValues>();
  const queryClient = useQueryClient();

  const expression = useWatch({ control, name: "expression" });
  const schemaId = useWatch({ control, name: "schemaId" });
  const variableDefinitions = useWatch({ control, name: "variableDefinitions" });
  const breakdownDefinitions = useWatch({ control, name: "breakdownDefinitions" });
  const minCharge = useWatch({ control, name: "minCharge" });
  const maxCharge = useWatch({ control, name: "maxCharge" });
  const roundingMode = useWatch({ control, name: "roundingMode" });
  const roundingPrecision = useWatch({ control, name: "roundingPrecision" });

  const savedQuery = useQuery({
    ...formulaTemplate.testCases(templateId ?? ""),
    enabled: !!templateId,
  });
  const [held, setHeld] = useState<StudioScenario[]>([]);
  const [freshIds, setFreshIds] = useState<ReadonlySet<string>>(() => new Set());
  const [runs, setRuns] = useState<ReadonlyMap<string, ScenarioRun>>(() => new Map());
  const [isStale, setIsStale] = useState(false);

  const scenarios = useMemo(
    () => (templateId ? (savedQuery.data ?? []).map(fromSaved) : held),
    [templateId, savedQuery.data, held],
  );

  const candidate = useCallback((): TestCaseCandidate => {
    const values = getValues();
    return {
      expression: values.expression ?? "",
      variableDefinitions: values.variableDefinitions,
      breakdownDefinitions: values.breakdownDefinitions,
      minCharge: values.minCharge ?? null,
      maxCharge: values.maxCharge ?? null,
      roundingMode: values.roundingMode,
      roundingPrecision: values.roundingPrecision,
    };
  }, [getValues]);

  const runMutation = useMutation({
    mutationFn: async (list: StudioScenario[]): Promise<Map<string, ScenarioRun>> => {
      const content = candidate();
      if (templateId) {
        const response = await apiService.formulaTemplateService.runTestCases(templateId, content);
        return new Map(
          response.results.map((result) => [
            result.testCaseId,
            {
              passed: result.passed,
              actualAmount: result.error ? null : result.actualAmount,
              error: result.error ?? null,
            },
          ]),
        );
      }
      const values = getValues();
      const priced = await Promise.all(
        list.map(async (scenario) => {
          const request = formulaTestRequest(values, {
            sampleValues: {},
            overrides: scenario.variables,
          });
          if (!request) {
            return [scenario.id, { passed: false, actualAmount: null, error: null }] as const;
          }
          const response = await apiService.formulaTemplateService.test(request);
          const actual = response.valid ? amountOf(response.result) : null;
          return [
            scenario.id,
            {
              passed:
                actual !== null &&
                Math.abs(actual - scenario.expectedAmount) <= scenario.tolerance + 1e-9,
              actualAmount: actual,
              error: response.valid ? null : (response.error ?? response.message),
            },
          ] as const;
        }),
      );
      return new Map(priced);
    },
    onSuccess: (next) => {
      setRuns(next);
      setIsStale(false);
    },
    onError: () => setIsStale(true),
  });
  const { mutate: run } = runMutation;

  const scenariosRef = useRef(scenarios);
  scenariosRef.current = scenarios;

  useEffect(() => {
    if (scenarios.length === 0) {
      setRuns(new Map());
      setIsStale(false);
      return;
    }
    if (!expression?.trim()) {
      return;
    }
    setIsStale(true);
    const timer = window.setTimeout(() => run(scenariosRef.current), SCENARIO_DEBOUNCE_MS);
    return () => window.clearTimeout(timer);
  }, [
    scenarios,
    expression,
    schemaId,
    variableDefinitions,
    breakdownDefinitions,
    minCharge,
    maxCharge,
    roundingMode,
    roundingPrecision,
    run,
  ]);

  const invalidate = useCallback(async () => {
    if (!templateId) return;
    await queryClient.invalidateQueries({
      queryKey: formulaTemplate.testCases(templateId).queryKey,
    });
  }, [queryClient, templateId]);

  const markFresh = useCallback((ids: string[]) => {
    setFreshIds((current) => {
      const next = new Set(current);
      for (const id of ids) next.add(id);
      return next;
    });
  }, []);

  const add = useCallback(
    async (list: NewScenario[]): Promise<number> => {
      if (list.length === 0) return 0;
      if (!templateId) {
        const recordedContent = candidate();
        const additions = list.map((scenario): StudioScenario => ({
          id: `held_${nanoid(10)}`,
          name: scenario.name,
          description: scenario.description ?? "",
          variables: scenario.variables,
          expectedAmount: scenario.expectedAmount,
          tolerance: SCENARIO_TOLERANCE,
          recordedOnVersion: null,
          recordedBy: author,
          saved: null,
          recordedContent,
        }));
        setHeld((current) => [...current, ...additions]);
        markFresh(additions.map((scenario) => scenario.id));
        return additions.length;
      }
      const content = candidate();
      const created: string[] = [];
      try {
        for (const scenario of list) {
          const saved = await apiService.formulaTemplateService.createTestCase(templateId, {
            name: scenario.name,
            description: scenario.description ?? "",
            variables: scenario.variables,
            expectedAmount: scenario.expectedAmount,
            tolerance: SCENARIO_TOLERANCE,
            candidate: content,
          });
          created.push(saved.id);
        }
      } catch (error) {
        throw new Error(describeApiError(error, "The scenario could not be saved."), {
          cause: error,
        });
      } finally {
        markFresh(created);
        await invalidate();
      }
      return created.length;
    },
    [author, candidate, invalidate, markFresh, templateId],
  );

  const remove = useCallback(
    async (id: string) => {
      if (!templateId) {
        setHeld((current) => current.filter((scenario) => scenario.id !== id));
        return;
      }
      await apiService.formulaTemplateService.deleteTestCase(templateId, id);
      await invalidate();
    },
    [invalidate, templateId],
  );

  const accept = useCallback(
    async (id: string, amount: number) => {
      if (!templateId) {
        setHeld((current) =>
          current.map((scenario) =>
            scenario.id === id
              ? {
                  ...scenario,
                  expectedAmount: amount,
                  recordedContent: candidate(),
                  recordedBy: author,
                }
              : scenario,
          ),
        );
        return;
      }
      const scenario = scenariosRef.current.find((item) => item.id === id);
      if (!scenario?.saved) return;
      await apiService.formulaTemplateService.updateTestCase(templateId, id, {
        name: scenario.name,
        description: scenario.description,
        variables: scenario.variables,
        expectedAmount: amount,
        tolerance: scenario.tolerance,
        candidate: candidate(),
        version: scenario.saved.version,
      });
      await invalidate();
    },
    [author, candidate, invalidate, templateId],
  );

  const heldForCreate = useCallback(
    (): FormulaTestCaseInput[] =>
      held.map((scenario) => ({
        name: scenario.name,
        description: scenario.description,
        variables: scenario.variables,
        expectedAmount: scenario.expectedAmount,
        tolerance: scenario.tolerance,
        candidate: scenario.recordedContent ?? candidate(),
      })),
    [candidate, held],
  );

  const passed = scenarios.filter((scenario) => runs.get(scenario.id)?.passed).length;

  return {
    scenarios,
    isLoading: !!templateId && savedQuery.isLoading,
    runs,
    passed,
    total: scenarios.length,
    isPending: runMutation.isPending,
    isStale,
    freshIds,
    add,
    remove,
    accept,
    heldForCreate,
    candidate,
  };
}
