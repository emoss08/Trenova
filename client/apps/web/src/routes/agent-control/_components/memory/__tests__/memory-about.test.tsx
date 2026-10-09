import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { FormProvider, useForm, type UseFormReturn } from "react-hook-form";
import { describe, expect, it, vi } from "vitest";
import { MemoryAbout } from "../memory-form";
import type { MemoryFormValues } from "../memory-form-schema";

vi.mock("@/components/autocomplete-fields", () => ({
  CarrierAutocompleteField: () => null,
  CustomerAutocompleteField: () => null,
  LocationAutocompleteField: () => null,
  WorkerAutocompleteField: () => null,
}));

function renderAbout(defaults: Partial<MemoryFormValues>) {
  let form!: UseFormReturn<MemoryFormValues>;
  function Host({ children }: { children: ReactNode }) {
    form = useForm<MemoryFormValues>({
      defaultValues: {
        content: "Dock 4 closes at 15:00 on Fridays.",
        kind: "Fact",
        subjectType: null,
        subjectId: null,
        toolName: "",
        expiresAt: null,
        ...defaults,
      } as MemoryFormValues,
    });
    return <FormProvider {...form}>{children}</FormProvider>;
  }
  render(
    <QueryClientProvider client={new QueryClient()}>
      <Host>
        <MemoryAbout />
      </Host>
    </QueryClientProvider>,
  );
  return () => form.getValues();
}

describe("MemoryAbout", () => {
  it("picks what the memory is about from the shared select, not a native one", async () => {
    const values = renderAbout({});
    expect(document.querySelector("select")).toBeNull();

    await userEvent.click(screen.getByRole("button", { name: /Every agent/ }));
    await userEvent.click(await screen.findByRole("option", { name: /A customer/ }));

    await waitFor(() => expect(values().subjectType).toBe("Customer"));
    expect(values().subjectId).toBeNull();
  });

  it("goes back to every agent as no subject at all, and drops the record", async () => {
    const values = renderAbout({ subjectType: "Customer", subjectId: "cus_1" });

    await userEvent.click(screen.getByRole("button", { name: /A customer/ }));
    await userEvent.click(await screen.findByRole("option", { name: /Every agent/ }));

    await waitFor(() => expect(values().subjectType).toBeNull());
    expect(values().subjectId).toBeNull();
  });
});
