import type { PageRequest } from "@/components/assistant/use-thread-model";
import type { TariffDefinitionTarget } from "@/components/formula-editor/tariff-service";
import { tariffReadiness } from "@/components/tariff-studio/readiness";
import type { TariffTestsState } from "@/components/tariff-studio/use-tariff-tests";
import {
  useTariffEditorBinding,
  useTariffWorkbench,
} from "@/components/tariff-studio/use-tariff-workbench";
import { useMediaQuery } from "@/hooks/use-media-query";
import { useTariffCheckContext } from "@/hooks/use-tariff-check-context";
import { formulaTemplate } from "@/lib/queries/formula-template";
import type { EditorView } from "@codemirror/view";
import { useQuery } from "@tanstack/react-query";
import { useRules } from "@trenova/shared/hooks/use-rules";
import { useT } from "@trenova/shared/i18n/use-t";
import type { RulesRuntime } from "@trenova/shared/rules/runtime";
import type {
  FormulaDraftContent,
  FormulaTemplate,
  FormulaTemplateFormValues,
  ReadinessCheck,
  TemplateUsageResponse,
} from "@trenova/shared/types/formula-template";
import type {
  TariffCheckResult,
  TariffContext,
  TariffFix,
  TariffInputValues,
  TariffRange,
  TariffTestStatus,
} from "@trenova/shared/types/tariff";
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
import {
  useForm,
  useFormContext,
  useFormState,
  useWatch,
  type UseFormReturn,
} from "react-hook-form";
import { useLivePreview, type LivePreviewState } from "./use-live-preview";
import { useStudioRail } from "@/components/tariff-studio/use-studio-rail";

export type StudioMode = "create" | "edit";
export type StudioTab = "preview" | "tests" | "reference" | "nova";
export type StudioSection = "details" | "source" | "inputs" | "tests";

export const STUDIO_SECTIONS: readonly StudioSection[] = ["details", "source", "inputs", "tests"];

export const sectionDomId = (section: StudioSection) => `studio-section-${section}`;

/**
 * The sample input values the preview rates with, held apart from the template itself. The
 * values are wire values; the form types them loosely because react-hook-form's path types do
 * not terminate on a recursive value type.
 */
export type PreviewInputsForm = { inputs: Record<string, unknown> };

export type StudioProps = {
  mode: StudioMode;
  template: FormulaTemplate | null;
  isSubmitting: boolean;
  onSave: () => void;
  onTemplateChanged?: (template: FormulaTemplate) => void;
};

/**
 * What changes as the person works: the check, the tests, the preview and the tab. A consumer
 * of this re-renders on every keystroke in the editor.
 */
type StudioStateValue = {
  /** The organisation's context the source is checked against; null while it loads. */
  context: TariffContext | null;
  contextError: boolean;
  check: TariffCheckResult;
  errors: number;
  warnings: number;
  tests: TariffTestsState;
  /** Each test's verdict from the last run, by name. */
  testStatus: ReadonlyMap<string, TariffTestStatus>;
  preview: LivePreviewState;
  usage: TemplateUsageResponse | undefined;
  usageTotal: number;
  readiness: ReadinessCheck[];
  /** The editor holds unsaved changes. */
  dirty: boolean;
  /** The unsaved changes alter what the template bills, so saving demotes it. */
  material: boolean;
  tab: StudioTab;
  /** The screen is too narrow for the panel beside the form, so it floats over it. */
  stacked: boolean;
  /** The floating panel of preview, tests, reference and Nova is showing. */
  railOpen: boolean;
  novaRequest: PageRequest | null;
  /** Nova is writing a reply in the rail. */
  novaBusy: boolean;
  activeSection: StudioSection;
  /** The source range something outside the editor points at, lit in the editor. */
  highlight: TariffRange | null;
  /** The imported library declaration go to definition opened, if any. */
  definition: TariffDefinitionTarget | null;
};

/**
 * What stays the same while the person works: the page's props, the callbacks and the refs. A
 * consumer that needs only these never re-renders for a keystroke in the editor.
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
  /** The source editor's view, once it exists. */
  setEditorView: (view: EditorView | null) => void;
  /** Selects a source range in the editor and brings the editor into view. */
  reveal: (range: TariffRange) => void;
  /** Lights a range while something points at it; null clears it. */
  setHighlight: (range: TariffRange | null) => void;
  /** Applies a fix's edits through the editor, so they can be undone. */
  applyFix: (fix: Pick<TariffFix, "changes">) => void;
  /** Appends a `test` block to the source, through the editor so it can be undone. */
  appendToSource: (text: string) => void;
  openDefinition: (target: TariffDefinitionTarget) => void;
  closeDefinition: () => void;
  /** The editor as it is, saved or not, in the shape Nova's calls read. */
  draftContent: () => FormulaDraftContent;
  save: () => void;
  /** The preview's sample input values. */
  previewInputs: UseFormReturn<PreviewInputsForm>;
  /** Names Nova's conversation about a template not yet created, for the life of the page. */
  unsavedNovaKey: string;
  /** The loaded rules module, for work done outside a render (applying a proposal). */
  rules: RulesRuntime;
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

export function StudioProvider({ children, ...props }: StudioProps & { children: ReactNode }) {
  const t = useT();
  // Everything below checks the source synchronously with the rules module, so the studio
  // suspends here, behind its skeleton, until the module has loaded (main.tsx preloads it).
  const rules = useRules();
  const { mode, template, onSave } = props;
  const form = useFormContext<FormulaTemplateFormValues>();
  const { isDirty } = useFormState({ control: form.control });
  const [name, description, source] = useWatch({
    control: form.control,
    name: ["name", "description", "source"],
  });
  const text = source ?? "";

  const { context, isError: contextError } = useTariffCheckContext();
  const { check, errors, warnings, tests, testStatus } = useTariffWorkbench({
    rules,
    source: text,
    kind: "tariff",
    context,
  });

  const previewInputs = useForm<PreviewInputsForm>({ defaultValues: { inputs: {} } });
  const sampleInputs = useWatch({ control: previewInputs.control, name: "inputs" });
  const preview = useLivePreview({
    source: text,
    inputs: sampleInputs as TariffInputValues | undefined,
    errors,
  });

  const { data: usage } = useQuery({
    ...formulaTemplate.usage(template?.id ?? ""),
    enabled: mode === "edit" && !!template?.id,
    staleTime: 60_000,
  });
  const usageTotal = usage?.usages.reduce((sum, item) => sum + item.count, 0) ?? 0;

  const material = mode === "edit" && isDirty && template !== null && template.source !== text;

  const readiness = useMemo(
    () =>
      tariffReadiness(
        {
          mode,
          name: name ?? "",
          description: description ?? "",
          sourceEmpty: text.trim() === "",
          errors,
          warnings,
          tests,
          dirty: isDirty,
        },
        t,
      ),
    [mode, name, description, text, errors, warnings, tests, isDirty, t],
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
    (question?: string) => {
      setTab("nova");
      const trimmed = question?.trim();
      if (trimmed) setNovaRequest({ key: nanoid(), text: trimmed });
    },
    [setTab],
  );
  const clearNovaRequest = useCallback((key: string) => {
    setNovaRequest((current) => (current?.key === key ? null : current));
  }, []);

  const [novaBusy, setNovaBusy] = useState(false);
  const [activeSection, setActiveSection] = useState<StudioSection>("source");
  const [highlight, setHighlight] = useState<TariffRange | null>(null);
  const [definition, setDefinition] = useState<TariffDefinitionTarget | null>(null);
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

  const { setEditorView, reveal, applyFix, appendToSource } = useTariffEditorBinding({
    read: () => form.getValues("source") ?? "",
    write: (next) => form.setValue("source", next, { shouldDirty: true, shouldValidate: true }),
  });

  const openDefinition = useCallback((target: TariffDefinitionTarget) => setDefinition(target), []);
  const closeDefinition = useCallback(() => setDefinition(null), []);

  const draftContent = useCallback((): FormulaDraftContent => {
    const current = form.getValues();
    return {
      ...(template?.id ? { templateId: template.id } : {}),
      name: current.name ?? "",
      description: current.description ?? "",
      source: current.source ?? "",
    };
  }, [form, template]);

  const save = useCallback(() => onSave(), [onSave]);
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
      setEditorView,
      reveal,
      setHighlight,
      applyFix,
      appendToSource,
      openDefinition,
      closeDefinition,
      draftContent,
      save,
      previewInputs,
      unsavedNovaKey,
      rules,
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
      setEditorView,
      reveal,
      applyFix,
      appendToSource,
      openDefinition,
      closeDefinition,
      draftContent,
      save,
      previewInputs,
      unsavedNovaKey,
      rules,
    ],
  );

  const state = useMemo<StudioStateValue>(
    () => ({
      context,
      contextError,
      check,
      errors,
      warnings,
      tests,
      testStatus,
      preview,
      usage,
      usageTotal,
      readiness,
      dirty: isDirty,
      material,
      tab,
      stacked,
      railOpen,
      novaRequest,
      novaBusy,
      activeSection,
      highlight,
      definition,
    }),
    [
      context,
      contextError,
      check,
      errors,
      warnings,
      tests,
      testStatus,
      preview,
      usage,
      usageTotal,
      readiness,
      isDirty,
      material,
      tab,
      stacked,
      railOpen,
      novaRequest,
      novaBusy,
      activeSection,
      highlight,
      definition,
    ],
  );

  return (
    <StudioActionsContext.Provider value={actions}>
      <StudioStateContext.Provider value={state}>{children}</StudioStateContext.Provider>
    </StudioActionsContext.Provider>
  );
}
