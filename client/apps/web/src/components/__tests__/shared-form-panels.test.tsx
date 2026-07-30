import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, waitFor } from "@testing-library/react";
import { useForm, type UseFormReturn } from "react-hook-form";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";

vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }));

type ChargeForm = {
  code: string;
  description: string;
  status: string;
};

const PRISTINE: ChargeForm = { code: "", description: "", status: "Active" };
const ROW = {
  id: "acc_1",
  code: "DET",
  description: "Detention",
  status: "Inactive",
} as const;

// Mirrors the shape every route using these panels has: ONE useForm, handed to whichever
// panel `mode` selects. See routes/accessorial-charge/_components/accessorial-charge-panel.
function SharedFormPanels({
  mode,
  onForm,
}: {
  mode: "create" | "edit";
  onForm: (form: UseFormReturn<ChargeForm>) => void;
}) {
  const form = useForm<ChargeForm>({ defaultValues: PRISTINE });
  onForm(form);

  if (mode === "edit") {
    return (
      <FormEditPanel<ChargeForm, typeof ROW & Record<string, unknown>>
        open
        onOpenChange={() => undefined}
        row={ROW as unknown as typeof ROW & Record<string, unknown>}
        form={form}
        url="/accessorial-charges/"
        queryKey="accessorial-charge-list"
        title="Accessorial Charge"
        formComponent={null}
      />
    );
  }

  return (
    <FormCreatePanel<ChargeForm, typeof ROW>
      open
      onOpenChange={() => undefined}
      form={form}
      url="/accessorial-charges/"
      queryKey="accessorial-charge-list"
      title="Accessorial Charge"
      formComponent={null}
    />
  );
}

function wrapper(client: QueryClient) {
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
}

describe("create and edit panels sharing one form", () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
  });

  afterEach(() => {
    queryClient.clear();
    vi.clearAllMocks();
  });

  // The reported defect: open an edit panel, then open the create panel, and the create
  // form is pre-filled with the record that was just edited. reset(row) in the edit panel
  // REPLACES the form's defaultValues, so the create panel's bare reset() restored the
  // record instead of a blank form.
  it("opens the create panel blank after the edit panel loaded a record", async () => {
    let form!: UseFormReturn<ChargeForm>;
    const capture = (f: UseFormReturn<ChargeForm>) => {
      form = f;
    };

    const view = render(<SharedFormPanels mode="edit" onForm={capture} />, {
      wrapper: wrapper(queryClient),
    });

    await waitFor(() => expect(form.getValues().code).toBe("DET"));

    view.rerender(<SharedFormPanels mode="create" onForm={capture} />);

    await waitFor(() => expect(form.getValues()).toEqual(PRISTINE));
  });

  it("still loads the record into the edit panel", async () => {
    let form!: UseFormReturn<ChargeForm>;

    render(
      <SharedFormPanels
        mode="edit"
        onForm={(f) => {
          form = f;
        }}
      />,
      { wrapper: wrapper(queryClient) },
    );

    // reset(row) puts the whole record in, id included, so match rather than equal.
    await waitFor(() =>
      expect(form.getValues()).toMatchObject({
        code: "DET",
        description: "Detention",
        status: "Inactive",
      }),
    );
  });

  // isDirty in the edit panel must keep meaning "changed since this record loaded", which
  // is why the edit panel goes on replacing defaultValues rather than preserving them.
  it("keeps the edit panel undirty immediately after loading a record", async () => {
    let form!: UseFormReturn<ChargeForm>;

    render(
      <SharedFormPanels
        mode="edit"
        onForm={(f) => {
          form = f;
        }}
      />,
      { wrapper: wrapper(queryClient) },
    );

    await waitFor(() => expect(form.getValues().code).toBe("DET"));
    expect(form.formState.isDirty).toBe(false);
  });

  it("opens the create panel blank when no edit panel ran first", async () => {
    let form!: UseFormReturn<ChargeForm>;

    render(
      <SharedFormPanels
        mode="create"
        onForm={(f) => {
          form = f;
        }}
      />,
      { wrapper: wrapper(queryClient) },
    );

    await waitFor(() => expect(form.getValues()).toEqual(PRISTINE));
  });
});
