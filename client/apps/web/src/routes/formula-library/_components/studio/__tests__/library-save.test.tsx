import type { FormulaLibraryRecord } from "@/lib/graphql/formula-library";
import { useEditorPreferenceStore } from "@/stores/editor-preference-store";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import { rulesSync } from "@trenova/shared/rules/runtime";
import { tariffFormat } from "@trenova/shared/rules/tariff";
import type { FormulaLibraryFormValues } from "@trenova/shared/types/formula-library";
import type { ReactNode } from "react";
import { FormProvider, useForm, type UseFormReturn } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import { LibraryStudioProvider, useLibraryStudio } from "../library-studio-context";

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

const UNFORMATTED =
  'library "org.fees"\n\nfn flatFee(fee: Money = $150) -> Money {\n  return fee+$1\n}\n';
const CHANGED =
  'library "org.fees"\n\nfn flatFee(fee: Money = $150) -> Money {\n  return fee+$2\n}\n';

function savedLibrary(source: string): FormulaLibraryRecord {
  return { id: "fl_1", name: "fees", source } as FormulaLibraryRecord;
}

function renderStudio({
  library,
  source,
}: {
  library: FormulaLibraryRecord | null;
  source: string;
}) {
  const onSave = vi.fn();
  const form: { current: UseFormReturn<FormulaLibraryFormValues> | null } = { current: null };
  const queryClient = new QueryClient();
  function Wrapper({ children }: { children: ReactNode }) {
    const methods = useForm<FormulaLibraryFormValues>({
      defaultValues: { name: "fees", description: "", source },
    });
    form.current = methods;
    return (
      <QueryClientProvider client={queryClient}>
        <FormProvider {...methods}>
          <LibraryStudioProvider
            mode={library ? "edit" : "create"}
            library={library}
            isSubmitting={false}
            onSave={onSave}
          >
            {children}
          </LibraryStudioProvider>
        </FormProvider>
      </QueryClientProvider>
    );
  }
  const { result } = renderHook(() => useLibraryStudio(), { wrapper: Wrapper });
  return { result, onSave, source: () => form.current?.getValues("source") };
}

function canonical(source: string): string {
  const { formatted } = tariffFormat(rulesSync(), source);
  if (formatted === null || formatted === source) {
    throw new Error("the fixture must be valid and not yet in the canonical style");
  }
  return formatted;
}

describe("library studio save with format on save", () => {
  afterEach(() => {
    act(() => useEditorPreferenceStore.getState().setFormatOnSave(false));
  });

  it("formats a new library's source, then saves", () => {
    act(() => useEditorPreferenceStore.getState().setFormatOnSave(true));
    const { result, onSave, source } = renderStudio({ library: null, source: UNFORMATTED });

    act(() => result.current.save());

    expect(source()).toBe(canonical(UNFORMATTED));
    expect(onSave).toHaveBeenCalledTimes(1);
  });

  it("formats an edited source", () => {
    act(() => useEditorPreferenceStore.getState().setFormatOnSave(true));
    const { result, onSave, source } = renderStudio({
      library: savedLibrary(UNFORMATTED),
      source: CHANGED,
    });

    act(() => result.current.save());

    expect(source()).toBe(canonical(CHANGED));
    expect(onSave).toHaveBeenCalledTimes(1);
  });

  it("leaves the saved source as written when only the description changed", () => {
    act(() => useEditorPreferenceStore.getState().setFormatOnSave(true));
    const { result, onSave, source } = renderStudio({
      library: savedLibrary(UNFORMATTED),
      source: UNFORMATTED,
    });

    act(() => result.current.save());

    expect(source()).toBe(UNFORMATTED);
    expect(onSave).toHaveBeenCalledTimes(1);
  });

  it("saves the source as written while format on save is off", () => {
    const { result, onSave, source } = renderStudio({ library: null, source: UNFORMATTED });

    act(() => result.current.save());

    expect(source()).toBe(UNFORMATTED);
    expect(onSave).toHaveBeenCalledTimes(1);
  });
});
