import type { PageRequest } from "@/components/assistant/use-thread-model";
import type { TariffDefinitionTarget } from "@/components/formula-editor/tariff-service";
import { tariffReadiness } from "@/components/tariff-studio/readiness";
import {
  useTariffEditorBinding,
  useTariffWorkbench,
  type TariffEditorBinding,
  type TariffWorkbench,
} from "@/components/tariff-studio/use-tariff-workbench";
import { useStudioRail } from "@/components/tariff-studio/use-studio-rail";
import { useMediaQuery } from "@/hooks/use-media-query";
import { useEditorPreferenceStore } from "@/stores/editor-preference-store";
import { useTariffCheckContext } from "@/hooks/use-tariff-check-context";
import type { FormulaLibraryRecord } from "@/lib/graphql/formula-library";
import { useRules } from "@trenova/shared/hooks/use-rules";
import { useT } from "@trenova/shared/i18n/use-t";
import type { RulesRuntime } from "@trenova/shared/rules/runtime";
import type { FormulaLibraryFormValues } from "@trenova/shared/types/formula-library";
import type { ReadinessCheck } from "@trenova/shared/types/formula-template";
import type { TariffContext, TariffRange } from "@trenova/shared/types/tariff";
import { nanoid } from "nanoid";
import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react";
import { useFormContext, useFormState, useWatch } from "react-hook-form";

export type LibraryStudioMode = "create" | "edit";
export type LibraryStudioTab = "tests" | "reference" | "usage" | "history" | "nova";

export type LibraryStudioProps = {
  mode: LibraryStudioMode;
  library: FormulaLibraryRecord | null;
  isSubmitting: boolean;
  onSave: () => void;
};

type LibraryStudioValue = LibraryStudioProps &
  TariffWorkbench &
  TariffEditorBinding & {
    rules: RulesRuntime;
    context: TariffContext | null;
    contextError: boolean;
    readiness: ReadinessCheck[];
    dirty: boolean;
    /** The unsaved changes make a new version, so saving an approved library returns it to draft. */
    material: boolean;
    /** The name was changed from the saved one, which changes the path templates import. */
    renamed: boolean;
    tab: LibraryStudioTab;
    setTab: (tab: LibraryStudioTab) => void;
    stacked: boolean;
    railOpen: boolean;
    setRailOpen: (open: boolean) => void;
    novaRequest: PageRequest | null;
    askNova: (text?: string) => void;
    clearNovaRequest: (key: string) => void;
    novaBusy: boolean;
    setNovaBusy: (busy: boolean) => void;
    unsavedNovaKey: string;
    highlight: TariffRange | null;
    /** Saves the library, formatting a changed source first when the person asked for that. */
    save: () => void;
    setHighlight: (range: TariffRange | null) => void;
    definition: TariffDefinitionTarget | null;
    openDefinition: (target: TariffDefinitionTarget) => void;
    closeDefinition: () => void;
  };

const LibraryStudioContext = createContext<LibraryStudioValue | null>(null);

export function useLibraryStudio(): LibraryStudioValue {
  const value = useContext(LibraryStudioContext);
  if (!value) throw new Error("useLibraryStudio must be used inside the library studio");
  return value;
}

/**
 * Everything the library studio knows: the check and tests of the source in the editor, the
 * live readiness, and the rail's tab. It suspends until the rules module has loaded.
 */
export function LibraryStudioProvider({
  children,
  ...props
}: LibraryStudioProps & { children: ReactNode }) {
  const t = useT();
  const rules = useRules();
  const { mode, library, isSubmitting, onSave } = props;
  const form = useFormContext<FormulaLibraryFormValues>();
  const { isDirty } = useFormState({ control: form.control });
  const [name, description, source] = useWatch({
    control: form.control,
    name: ["name", "description", "source"],
  });
  const text = source ?? "";

  const { context, tablesStamp, isError: contextError } = useTariffCheckContext();
  const workbench = useTariffWorkbench({
    rules,
    source: text,
    kind: "library",
    context,
    tablesStamp,
  });
  const binding = useTariffEditorBinding({
    read: () => form.getValues("source") ?? "",
    write: (next) => form.setValue("source", next, { shouldDirty: true, shouldValidate: true }),
  });

  const { formatSource } = binding;
  const save = useCallback(() => {
    const changed = library === null || (form.getValues("source") ?? "") !== library.source;
    if (useEditorPreferenceStore.getState().formatOnSave && changed) formatSource(rules);
    onSave();
  }, [library, form, formatSource, rules, onSave]);

  const renamed = mode === "edit" && library !== null && (name ?? "").trim() !== library.name;
  const material =
    mode === "edit" && isDirty && library !== null && (library.source !== text || renamed);

  const readiness = useMemo(
    () =>
      tariffReadiness(
        {
          mode,
          name: name ?? "",
          description: description ?? "",
          sourceEmpty: text.trim() === "",
          errors: workbench.errors,
          warnings: workbench.warnings,
          tests: workbench.tests,
          coverage: workbench.coverage,
          dirty: isDirty,
          testsRequired: true,
        },
        t,
      ),
    [mode, name, description, text, workbench, isDirty, t],
  );

  const stacked = useMediaQuery("(max-width: 899px)");
  const [railOpen, setRailOpen] = useStudioRail(stacked);
  const [tab, setTabState] = useState<LibraryStudioTab>("tests");
  const setTab = useCallback(
    (next: LibraryStudioTab) => {
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
  const [unsavedNovaKey] = useState(() => nanoid());

  const [highlight, setHighlight] = useState<TariffRange | null>(null);
  const [definition, setDefinition] = useState<TariffDefinitionTarget | null>(null);
  const openDefinition = useCallback((target: TariffDefinitionTarget) => setDefinition(target), []);
  const closeDefinition = useCallback(() => setDefinition(null), []);

  const value = useMemo<LibraryStudioValue>(
    () => ({
      mode,
      library,
      isSubmitting,
      onSave,
      ...workbench,
      ...binding,
      rules,
      context,
      contextError,
      readiness,
      dirty: isDirty,
      material,
      renamed,
      tab,
      setTab,
      stacked,
      railOpen,
      setRailOpen,
      novaRequest,
      askNova,
      clearNovaRequest,
      novaBusy,
      setNovaBusy,
      unsavedNovaKey,
      highlight,
      save,
      setHighlight,
      definition,
      openDefinition,
      closeDefinition,
    }),
    [
      mode,
      library,
      isSubmitting,
      onSave,
      workbench,
      binding,
      rules,
      context,
      contextError,
      readiness,
      isDirty,
      material,
      renamed,
      tab,
      setTab,
      stacked,
      railOpen,
      setRailOpen,
      novaRequest,
      askNova,
      clearNovaRequest,
      novaBusy,
      unsavedNovaKey,
      highlight,
      save,
      definition,
      openDefinition,
      closeDefinition,
    ],
  );

  return <LibraryStudioContext.Provider value={value}>{children}</LibraryStudioContext.Provider>;
}
