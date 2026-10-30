import { useEditorPreferenceStore } from "@/stores/editor-preference-store";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import type {
  FormulaTemplate,
  FormulaTemplateFormValues,
} from "@trenova/shared/types/formula-template";
import type { ReactNode } from "react";
import { FormProvider, useForm, type UseFormReturn } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import { StudioProvider, useStudioActions } from "../studio-context";

vi.mock("@/hooks/use-tariff-check-context", () => ({
  useTariffCheckContext: () => ({
    context: null,
    libraries: [],
    tablesStamp: "",
    isError: false,
    refetch: () => undefined,
  }),
}));

vi.mock("@/hooks/use-media-query", () => ({ useMediaQuery: () => false }));

vi.mock("@/lib/queries/formula-template", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/queries/formula-template")>();
  return {
    ...actual,
    formulaTemplate: {
      ...actual.formulaTemplate,
      usage: (templateId: string) => ({
        queryKey: ["formula-template-usage", templateId],
        queryFn: () => new Promise<never>(() => undefined),
      }),
    },
  };
});

vi.mock("../use-live-preview", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../use-live-preview")>();
  return {
    ...actual,
    useLivePreview: () => ({
      result: undefined,
      isPending: false,
      requestError: null,
      lastRunAt: null,
      source: "sample",
      setSource: () => undefined,
      shipmentId: "",
      setShipmentId: () => undefined,
      lastRequest: null,
      blocked: false,
      runNow: () => undefined,
    }),
  };
});

const UNFORMATTED = 'tariff "Old"\nlet a=1+2\n';

function activeTemplate(source: string): FormulaTemplate {
  return { id: "ft_1", name: "Old", source, status: "Active" } as FormulaTemplate;
}

function renderStudio({
  template,
  values,
}: {
  template: FormulaTemplate | null;
  values: Pick<FormulaTemplateFormValues, "name" | "source">;
}) {
  const onSave = vi.fn();
  const form: { current: UseFormReturn<FormulaTemplateFormValues> | null } = { current: null };
  const queryClient = new QueryClient();
  function Wrapper({ children }: { children: ReactNode }) {
    const methods = useForm<FormulaTemplateFormValues>({
      defaultValues: { description: "", ...values },
    });
    form.current = methods;
    return (
      <QueryClientProvider client={queryClient}>
        <FormProvider {...methods}>
          <StudioProvider
            mode={template ? "edit" : "create"}
            template={template}
            isSubmitting={false}
            onSave={onSave}
          >
            {children}
          </StudioProvider>
        </FormProvider>
      </QueryClientProvider>
    );
  }
  const { result } = renderHook(() => useStudioActions(), { wrapper: Wrapper });
  return { result, onSave, source: () => form.current?.getValues("source") };
}

describe("formula studio save with format on save", () => {
  afterEach(() => {
    act(() => useEditorPreferenceStore.getState().setFormatOnSave(false));
  });

  it("formats a new template's source, then saves", () => {
    act(() => useEditorPreferenceStore.getState().setFormatOnSave(true));
    const { result, onSave, source } = renderStudio({
      template: null,
      values: { name: "Old", source: UNFORMATTED },
    });

    act(() => result.current.save());

    expect(source()).toBe('tariff "Old"\nlet a = 1 + 2\n');
    expect(onSave).toHaveBeenCalledTimes(1);
  });

  it("formats an edit that changes what the template bills", () => {
    act(() => useEditorPreferenceStore.getState().setFormatOnSave(true));
    const { result, onSave, source } = renderStudio({
      template: activeTemplate(UNFORMATTED),
      values: { name: "Old", source: 'tariff "Old"\nlet a=1+3\n' },
    });

    act(() => result.current.save());

    expect(source()).toBe('tariff "Old"\nlet a = 1 + 3\n');
    expect(onSave).toHaveBeenCalledTimes(1);
  });

  it("leaves a rename of an approved template as written, so it stays approved", () => {
    act(() => useEditorPreferenceStore.getState().setFormatOnSave(true));
    const { result, onSave, source } = renderStudio({
      template: activeTemplate(UNFORMATTED),
      values: { name: "New", source: 'tariff "New"\nlet a=1+2\n' },
    });

    act(() => result.current.save());

    expect(source()).toBe('tariff "New"\nlet a=1+2\n');
    expect(onSave).toHaveBeenCalledTimes(1);
  });

  it("saves a source with syntax errors as written", () => {
    act(() => useEditorPreferenceStore.getState().setFormatOnSave(true));
    const { result, onSave, source } = renderStudio({
      template: null,
      values: { name: "Old", source: 'tariff "Old"\nlet a = 1 +\n' },
    });

    act(() => result.current.save());

    expect(source()).toBe('tariff "Old"\nlet a = 1 +\n');
    expect(onSave).toHaveBeenCalledTimes(1);
  });

  it("saves the source as written while format on save is off", () => {
    const { result, onSave, source } = renderStudio({
      template: null,
      values: { name: "Old", source: UNFORMATTED },
    });

    act(() => result.current.save());

    expect(source()).toBe(UNFORMATTED);
    expect(onSave).toHaveBeenCalledTimes(1);
  });
});
