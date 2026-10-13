import type { PageRequest } from "@/components/assistant/use-thread-model";
import {
  reconcileBreakdown,
  type BreakdownReconciliation,
} from "@/components/formula-editor/breakdown-reconcile";
import type { KnownIdentifiers } from "@/components/formula-editor/known-identifiers";
import { useKnownIdentifiers } from "@/hooks/use-formula-schema";
import { hasMaterialChange } from "@/lib/formula-template-material";
import { queries } from "@/lib/queries";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useT } from "@trenova/shared/i18n/use-t";
import type {
  FormulaDraftContent,
  FormulaTemplate,
  FormulaTemplateFormValues,
  FormulaTestCaseInput,
  ReadinessCheck,
  TemplateUsageResponse,
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
import { analyzeExpression, type ExpressionAnalysis } from "./model/expression-analysis";
import { studioReadiness } from "./model/readiness";
import { useLivePreview, type LivePreviewState } from "./use-live-preview";
import { useStudioScenarios, type StudioScenariosState } from "./use-studio-scenarios";

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

type StudioContextValue = StudioProps & {
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
  setTab: (tab: StudioTab) => void;
  /** Sends a question to Nova in the rail; without text it only opens the tab. */
  askNova: (text?: string) => void;
  novaRequest: PageRequest | null;
  clearNovaRequest: (key: string) => void;
  /** Nova is writing a reply in the rail. */
  novaBusy: boolean;
  setNovaBusy: (busy: boolean) => void;
  activeSection: StudioSection;
  setActiveSection: (section: StudioSection) => void;
  jumpTo: (section: StudioSection) => void;
  formViewport: RefObject<HTMLDivElement | null>;
  /** The editor as it is, saved or not, in the shape Nova's calls read. */
  draftContent: () => FormulaDraftContent;
  save: () => void;
};

const StudioContext = createContext<StudioContextValue | null>(null);

export function useStudio(): StudioContextValue {
  const context = useContext(StudioContext);
  if (!context) {
    throw new Error("useStudio must be used inside the formula studio");
  }
  return context;
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

  const [name, description, expression, schemaId, variableDefinitions, breakdownDefinitions] =
    useWatch({
      control: form.control,
      name: [
        "name",
        "description",
        "expression",
        "schemaId",
        "variableDefinitions",
        "breakdownDefinitions",
      ],
    });

  const known = useKnownIdentifiers(schemaId || "shipment", variableDefinitions ?? []);
  const customNames = useMemo(
    () => new Set((variableDefinitions ?? []).map((variable) => variable.name).filter(Boolean)),
    [variableDefinitions],
  );
  const analysis = useMemo(
    () => analyzeExpression(expression ?? "", known, customNames),
    [expression, known, customNames],
  );

  const preview = useLivePreview();
  const author = useMemo(
    () => (user?.id ? { id: user.id, name: user.name } : null),
    [user?.id, user?.name],
  );
  const scenarios = useStudioScenarios({ templateId: template?.id ?? null, author });

  const { data: usage } = useQuery({
    ...queries.formulaTemplate.usage(template?.id ?? ""),
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

  const values = useWatch({ control: form.control });
  const material =
    mode === "edit" && isDirty && template !== null
      ? hasMaterialChange(template, values as FormulaTemplateFormValues)
      : false;

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

  const [tab, setTab] = useState<StudioTab>("preview");
  const [novaRequest, setNovaRequest] = useState<PageRequest | null>(null);
  const askNova = useCallback((text?: string) => {
    setTab("nova");
    const question = text?.trim();
    if (question) {
      setNovaRequest({ key: nanoid(), text: question });
    }
  }, []);
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

  const heldForCreate = scenarios.heldForCreate;
  const save = useCallback(() => {
    onSave({ testCases: mode === "create" ? heldForCreate() : [] });
  }, [heldForCreate, mode, onSave]);

  const value: StudioContextValue = {
    ...props,
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
    setTab,
    askNova,
    novaRequest,
    clearNovaRequest,
    novaBusy,
    setNovaBusy,
    activeSection,
    setActiveSection,
    jumpTo,
    formViewport,
    draftContent,
    save,
  };

  return <StudioContext.Provider value={value}>{children}</StudioContext.Provider>;
}
