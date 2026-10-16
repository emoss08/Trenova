import type { PageRequest } from "@/components/assistant/use-thread-model";
import {
  reconcileBreakdown,
  type BreakdownReconciliation,
} from "@/components/formula-editor/breakdown-reconcile";
import type { KnownIdentifiers } from "@/components/formula-editor/known-identifiers";
import { useKnownIdentifiers } from "@/hooks/use-formula-schema";
import { useMediaQuery } from "@/hooks/use-media-query";
import { hasMaterialChange } from "@/lib/formula-template-material";
import { formulaTemplate } from "@/lib/queries/formula-template";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useT } from "@trenova/shared/i18n/use-t";
import type {
  FormulaDraftContent,
  FormulaTemplate,
  FormulaTemplateFormValues,
  FormulaTestCaseInput,
  ReadinessCheck,
  TemplateUsageResponse,
  VariableValueType,
} from "@trenova/shared/types/formula-template";
import { useQuery } from "@tanstack/react-query";
import { nanoid } from "nanoid";
import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useRef,
  useState,
  type ReactNode,
  type RefObject,
} from "react";
import { useFormContext, useFormState, useWatch } from "react-hook-form";
import {
  analyzeExpression,
  inferInputType,
  type ExpressionAnalysis,
} from "./model/expression-analysis";
import { studioReadiness } from "./model/readiness";
import { useLivePreview, type LivePreviewState } from "./use-live-preview";
import { useStudioScenarios, type StudioScenariosState } from "./use-studio-scenarios";
import { useStudioRail } from "./use-studio-rail";

export type StudioMode = "create" | "edit";
export type StudioTab = "preview" | "scenarios" | "reference" | "nova";
export type StudioSection = "details" | "expression" | "inputs" | "policy" | "breakdown";

export const STUDIO_SECTIONS: readonly StudioSection[] = [
  "details",
  "expression",
  "inputs",
  "policy",
  "breakdown",
];

export const sectionDomId = (section: StudioSection) => `studio-section-${section}`;

/** What saving hands the page: the scenarios a template being created carries with it. */
export type StudioSaveExtras = { testCases: FormulaTestCaseInput[] };

export type StudioProps = {
  mode: StudioMode;
  template: FormulaTemplate | null;
  isSubmitting: boolean;
  onSave: (extras: StudioSaveExtras) => void;
  onTemplateChanged?: (template: FormulaTemplate) => void;
};

/**
 * What changes as the person works: the analysis, the preview, the scenarios
 * and the tab. A consumer of this re-renders on every keystroke in the editor.
 */
type StudioStateValue = {
  known: KnownIdentifiers;
  analysis: ExpressionAnalysis;
  preview: LivePreviewState;
  scenarios: StudioScenariosState;
  usage: TemplateUsageResponse | undefined;
  usageTotal: number;
  readiness: ReadinessCheck[];
  reconciliation: BreakdownReconciliation | null;
  /** The editor holds unsaved changes. */
  dirty: boolean;
  /** The unsaved changes alter what the template computes, so saving demotes it. */
  material: boolean;
  tab: StudioTab;
  /** The screen is too narrow for the panel beside the form, so it floats over it. */
  stacked: boolean;
  /** The floating panel of preview, scenarios, reference and Nova is showing. */
  railOpen: boolean;
  novaRequest: PageRequest | null;
  /** Nova is writing a reply in the rail. */
  novaBusy: boolean;
  activeSection: StudioSection;
  /** The input whose row should take focus, once, after it was declared. */
  focusInput: string | null;
};

/**
 * What stays the same while the person works: the page's props, the
 * callbacks and the refs. A consumer that needs only these never re-renders
 * for a keystroke in the editor.
 */
type StudioActionsValue = StudioProps & {
  /** Shows a pane, opening the floating panel when it is closed. */
  setTab: (tab: StudioTab) => void;
  setRailOpen: (open: boolean) => void;
  /** Sends a question to Nova in the rail; without text it only opens the tab. */
  askNova: (text?: string) => void;
  clearNovaRequest: (key: string) => void;
  setNovaBusy: (busy: boolean) => void;
  setActiveSection: (section: StudioSection) => void;
  jumpTo: (section: StudioSection) => void;
  formViewport: RefObject<HTMLDivElement | null>;
  /** The editor as it is, saved or not, in the shape Nova's calls read. */
  draftContent: () => FormulaDraftContent;
  save: () => void;
  /** Adds an input for a name the formula reads, and takes the person to its row. */
  declareInput: (name: string, type?: VariableValueType) => void;
  clearFocusInput: () => void;
  /** Names Nova's conversation about a template not yet created, for the life of the page. */
  unsavedNovaKey: string;
};

type StudioContextValue = StudioStateValue & StudioActionsValue;

const StudioStateContext = createContext<StudioStateValue | null>(null);
const StudioActionsContext = createContext<StudioActionsValue | null>(null);

/** The studio's callbacks and refs alone; stable, so a keystroke never reaches the consumer. */
export function useStudioActions(): StudioActionsValue {
  const context = useContext(StudioActionsContext);
  if (!context) {
    throw new Error("useStudioActions must be used inside the formula studio");
  }
  return context;
}

/** Everything the studio knows; the consumer follows every change in the editor. */
export function useStudio(): StudioContextValue {
  const state = useContext(StudioStateContext);
  const actions = useStudioActions();
  if (!state) {
    throw new Error("useStudio must be used inside the formula studio");
  }
  return useMemo(() => ({ ...state, ...actions }), [state, actions]);
}

function stringAmount(value: number | string | null | undefined): string | null {
  if (value === null || value === undefined || value === "") return null;
  return String(value);
}

export function StudioProvider({ children, ...props }: StudioProps & { children: ReactNode }) {
  const t = useT();
  const { mode, template, onSave } = props;
  const form = useFormContext<FormulaTemplateFormValues>();
  const { isDirty } = useFormState({ control: form.control });
  const user = useAuthStore((state) => state.user);

  const [
    name,
    description,
    expression,
    schemaId,
    variableDefinitions,
    breakdownDefinitions,
    type,
    minCharge,
    maxCharge,
    roundingMode,
    roundingPrecision,
  ] = useWatch({
    control: form.control,
    name: [
      "name",
      "description",
      "expression",
      "schemaId",
      "variableDefinitions",
      "breakdownDefinitions",
      "type",
      "minCharge",
      "maxCharge",
      "roundingMode",
      "roundingPrecision",
    ],
  });

  const known = useKnownIdentifiers(schemaId || "shipment", variableDefinitions ?? []);
  const analysis = useMemo(() => analyzeExpression(expression ?? "", known), [expression, known]);

  const preview = useLivePreview();
  const author = useMemo(
    () => (user?.id ? { id: user.id, name: user.name } : null),
    [user?.id, user?.name],
  );
  const scenarios = useStudioScenarios({ templateId: template?.id ?? null, author });

  const { data: usage } = useQuery({
    ...formulaTemplate.usage(template?.id ?? ""),
    enabled: mode === "edit" && !!template?.id,
    staleTime: 60_000,
  });
  const usageTotal = usage?.usages.reduce((sum, item) => sum + item.count, 0) ?? 0;

  const result = preview.result;
  const reconciliation = useMemo(() => {
    if (!result?.valid || !result.breakdown || result.breakdown.length === 0) return null;
    const total = typeof result.result === "number" ? result.result : Number(result.result);
    if (!Number.isFinite(total)) return null;
    return reconcileBreakdown({
      total,
      rawAmount: result.receipt?.rawAmount ?? result.guardrail?.rawAmount ?? total,
      guardrailApplied: result.guardrail?.applied ?? false,
      lines: result.breakdown,
    });
  }, [result]);

  // Only the fields that decide materiality are compared, and only when one
  // of them changes: watching the whole form here re-ran the comparison, and
  // re-rendered every consumer, for each character typed into the name.
  const material = useMemo(
    () =>
      mode === "edit" && isDirty && template !== null
        ? hasMaterialChange(template, {
            expression,
            schemaId,
            type,
            minCharge,
            maxCharge,
            roundingMode,
            roundingPrecision,
            variableDefinitions,
            breakdownDefinitions,
          } as FormulaTemplateFormValues)
        : false,
    [
      mode,
      isDirty,
      template,
      expression,
      schemaId,
      type,
      minCharge,
      maxCharge,
      roundingMode,
      roundingPrecision,
      variableDefinitions,
      breakdownDefinitions,
    ],
  );

  const readiness = useMemo(
    () =>
      studioReadiness(
        {
          mode,
          name: name ?? "",
          description: description ?? "",
          expressionEmpty: analysis.empty,
          expressionErrors: analysis.errorCount,
          previewValid: result ? result.valid : null,
          unguardedFields: [...new Set((result?.warnings ?? []).map((warning) => warning.field))],
          scenarioTotal: scenarios.total,
          scenarioPassed: scenarios.passed,
          breakdownLines: (breakdownDefinitions ?? []).length,
          reconciliation,
          dirty: isDirty,
        },
        t,
      ),
    [
      mode,
      name,
      description,
      analysis,
      result,
      scenarios.total,
      scenarios.passed,
      breakdownDefinitions,
      reconciliation,
      isDirty,
      t,
    ],
  );

  const stacked = useMediaQuery("(max-width: 899px)");
  const [railOpen, setRailOpen] = useStudioRail(stacked);
  const [tab, setTabState] = useState<StudioTab>("preview");
  const setTab = useCallback(
    (next: StudioTab) => {
      setTabState(next);
      setRailOpen(true);
    },
    [setRailOpen],
  );
  const [novaRequest, setNovaRequest] = useState<PageRequest | null>(null);
  const askNova = useCallback(
    (text?: string) => {
      setTab("nova");
      const question = text?.trim();
      if (question) {
        setNovaRequest({ key: nanoid(), text: question });
      }
    },
    [setTab],
  );
  const clearNovaRequest = useCallback((key: string) => {
    setNovaRequest((current) => (current?.key === key ? null : current));
  }, []);

  const [novaBusy, setNovaBusy] = useState(false);
  const [activeSection, setActiveSection] = useState<StudioSection>("expression");
  const formViewport = useRef<HTMLDivElement | null>(null);
  const jumpTo = useCallback((section: StudioSection) => {
    const viewport = formViewport.current;
    const target = document.getElementById(sectionDomId(section));
    if (!target) return;
    if (viewport && viewport.contains(target)) {
      const offset = target.getBoundingClientRect().top - viewport.getBoundingClientRect().top - 24;
      viewport.scrollTo({ top: viewport.scrollTop + offset, behavior: "smooth" });
    } else {
      target.scrollIntoView({ behavior: "smooth", block: "start" });
    }
  }, []);

  const draftContent = useCallback((): FormulaDraftContent => {
    const current = form.getValues();
    return {
      ...(template?.id ? { templateId: template.id } : {}),
      name: current.name ?? "",
      description: current.description ?? "",
      type: current.type,
      schemaId: current.schemaId || "shipment",
      expression: current.expression ?? "",
      variableDefinitions: current.variableDefinitions ?? [],
      breakdownDefinitions: current.breakdownDefinitions ?? [],
      minCharge: stringAmount(current.minCharge),
      maxCharge: stringAmount(current.maxCharge),
      roundingMode: current.roundingMode,
      roundingPrecision: current.roundingPrecision,
    };
  }, [form, template?.id]);

  const [focusInput, setFocusInput] = useState<string | null>(null);
  const clearFocusInput = useCallback(() => setFocusInput(null), []);
  const declareInput = useCallback(
    (name: string, type?: VariableValueType) => {
      const current = form.getValues("variableDefinitions") ?? [];
      const existing = current.findIndex((variable) => variable.name === name);
      if (existing === -1) {
        const kind = type ?? inferInputType(name);
        form.setValue(
          "variableDefinitions",
          [
            ...current,
            {
              name,
              type: kind,
              description: "",
              required: false,
              defaultValue: kind === "Boolean" ? false : kind === "String" ? "" : undefined,
            },
          ],
          { shouldDirty: true, shouldValidate: true },
        );
      }
      setFocusInput(name);
      jumpTo("inputs");
    },
    [form, jumpTo],
  );

  const heldForCreate = scenarios.heldForCreate;
  const save = useCallback(() => {
    onSave({ testCases: mode === "create" ? heldForCreate() : [] });
  }, [heldForCreate, mode, onSave]);

  const [unsavedNovaKey] = useState(() => nanoid());

  const { isSubmitting, onTemplateChanged } = props;
  const actions = useMemo<StudioActionsValue>(
    () => ({
      mode,
      template,
      isSubmitting,
      onSave,
      onTemplateChanged,
      setTab,
      setRailOpen,
      askNova,
      clearNovaRequest,
      setNovaBusy,
      setActiveSection,
      jumpTo,
      formViewport,
      draftContent,
      save,
      declareInput,
      clearFocusInput,
      unsavedNovaKey,
    }),
    [
      mode,
      template,
      isSubmitting,
      onSave,
      onTemplateChanged,
      setTab,
      setRailOpen,
      askNova,
      clearNovaRequest,
      jumpTo,
      draftContent,
      save,
      declareInput,
      clearFocusInput,
      unsavedNovaKey,
    ],
  );

  const state = useMemo<StudioStateValue>(
    () => ({
      known,
      analysis,
      preview,
      scenarios,
      usage,
      usageTotal,
      readiness,
      reconciliation,
      dirty: isDirty,
      material,
      tab,
      stacked,
      railOpen,
      novaRequest,
      novaBusy,
      activeSection,
      focusInput,
    }),
    [
      known,
      analysis,
      preview,
      scenarios,
      usage,
      usageTotal,
      readiness,
      reconciliation,
      isDirty,
      material,
      tab,
      stacked,
      railOpen,
      novaRequest,
      novaBusy,
      activeSection,
      focusInput,
    ],
  );

  return (
    <StudioActionsContext.Provider value={actions}>
      <StudioStateContext.Provider value={state}>{children}</StudioStateContext.Provider>
    </StudioActionsContext.Provider>
  );
}
