import { PLANNED_PREVIEW_DEBOUNCE_MS, usePlannedPreview } from "@/hooks/use-planned-preview";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { useForm } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import { z } from "zod";

const schema = z.object({ amountMinor: z.number().int().positive(), memo: z.string() });
type Values = z.infer<typeof schema>;
type PlanInput = { submissionId: string; amountMinor: number };
type Plan = { amountMinor: number };

const fetchPlan = vi.fn((input: PlanInput): Promise<Plan> =>
  Promise.resolve({ amountMinor: input.amountMinor }),
);

afterEach(() => {
  vi.clearAllMocks();
});

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

function useFundingPreview(defaults: Values) {
  const form = useForm<Values>({ defaultValues: defaults });
  const preview = usePlannedPreview({
    control: form.control,
    schema,
    toInput: (values: Values) =>
      values.memo === "skip" ? null : { submissionId: "facsub_1", amountMinor: values.amountMinor },
    query: (input: PlanInput) => ({
      queryKey: ["plan", input],
      queryFn: () => fetchPlan(input),
    }),
  });
  return { form, preview };
}

describe("usePlannedPreview", () => {
  it("asks for no plan while the form fails its schema or makes no request", async () => {
    const { result } = renderHook(() => useFundingPreview({ amountMinor: 0, memo: "" }), {
      wrapper,
    });

    expect(result.current.preview.hasInput).toBe(false);
    act(() => result.current.form.setValue("amountMinor", 5_000));
    act(() => result.current.form.setValue("memo", "skip"));
    await new Promise((resolve) => setTimeout(resolve, PLANNED_PREVIEW_DEBOUNCE_MS + 100));

    expect(result.current.preview.hasInput).toBe(false);
    expect(result.current.preview.previewing).toBe(false);
    expect(fetchPlan).not.toHaveBeenCalled();
  });

  it("plans the values the form opens with, then only the last of a quick run of edits", async () => {
    const { result } = renderHook(() => useFundingPreview({ amountMinor: 1_000, memo: "" }), {
      wrapper,
    });

    expect(result.current.preview.hasInput).toBe(true);

    await waitFor(() => expect(result.current.preview.plan.data).toEqual({ amountMinor: 1_000 }));
    expect(result.current.preview.previewing).toBe(true);
    expect(fetchPlan).toHaveBeenCalledExactlyOnceWith({
      submissionId: "facsub_1",
      amountMinor: 1_000,
    });

    act(() => result.current.form.setValue("amountMinor", 2_000));
    act(() => result.current.form.setValue("amountMinor", 3_000));

    expect(result.current.preview.previewing).toBe(false);
    expect(result.current.preview.plan.data).toEqual({ amountMinor: 1_000 });
    await new Promise((resolve) => setTimeout(resolve, PLANNED_PREVIEW_DEBOUNCE_MS / 2));
    expect(fetchPlan).toHaveBeenCalledTimes(1);

    await waitFor(() => expect(result.current.preview.plan.data).toEqual({ amountMinor: 3_000 }));
    expect(result.current.preview.previewing).toBe(true);
    expect(result.current.preview.plan.isPlaceholderData).toBe(false);
    expect(fetchPlan.mock.calls.map(([input]) => input.amountMinor)).toEqual([1_000, 3_000]);
  });
});
