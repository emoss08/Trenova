import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { useForm, type UseFormReturn } from "react-hook-form";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { GLAccountAutocompleteField } from "@/components/autocomplete-fields";

// Whether a picker can be emptied depends on the form's value and the field's props, not on
// what the server lists, so every request is answered with an empty list.
beforeEach(() => {
  vi.stubGlobal(
    "fetch",
    vi.fn(
      async () =>
        new Response(
          JSON.stringify({
            data: {
              selectOptions: {
                edges: [],
                pageInfo: { hasNextPage: false, endCursor: null },
                totalCount: 0,
              },
            },
          }),
          { headers: { "content-type": "application/json" } },
        ),
    ),
  );
});

afterEach(() => {
  vi.unstubAllGlobals();
});

afterEach(cleanup);

type Values = { accountId: string };

const queryClient = new QueryClient();

function Harness({
  accountId,
  required,
  clearable,
  onForm,
}: {
  accountId: string;
  required?: boolean;
  clearable?: boolean;
  onForm: (form: UseFormReturn<Values>) => void;
}) {
  const form = useForm<Values>({ defaultValues: { accountId } });
  onForm(form);

  return (
    <QueryClientProvider client={queryClient}>
      <GLAccountAutocompleteField
        control={form.control}
        name="accountId"
        label="Purchased transportation account"
        required={required}
        clearable={clearable}
      />
    </QueryClientProvider>
  );
}

function renderField(props: { accountId: string; required?: boolean; clearable?: boolean }) {
  const held: { form?: UseFormReturn<Values> } = {};
  render(<Harness {...props} onForm={(form) => (held.form = form)} />);
  return {
    value: () => held.form!.getValues("accountId"),
    canClear: () => screen.queryByTitle("Clear") !== null,
    clear: () => fireEvent.click(screen.getByTitle("Clear")),
  };
}

describe("a picker that is not required", () => {
  it("can be emptied once it holds a record", () => {
    const field = renderField({ accountId: "gla_1" });

    expect(field.canClear()).toBe(true);
    field.clear();

    expect(field.value()).toBeFalsy();
    expect(field.canClear()).toBe(false);
  });

  it("offers nothing to clear while it is empty", () => {
    expect(renderField({ accountId: "" }).canClear()).toBe(false);
  });

  it("cannot be emptied where the form says it must always hold a record", () => {
    expect(renderField({ accountId: "gla_1", clearable: false }).canClear()).toBe(false);
  });
});

describe("a required picker", () => {
  it("cannot be emptied", () => {
    expect(renderField({ accountId: "gla_1", required: true }).canClear()).toBe(false);
  });
});
