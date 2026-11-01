import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, waitFor } from "@testing-library/react";
import { useForm, type UseFormReturn } from "react-hook-form";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";

vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: { error: vi.fn(), success: vi.fn() },
}));

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
// panel `mode` selects. See components/accessorial-charge/accessorial-charge-panel.
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
        mutationFn={async (values) => values}
        queryKey="accessorial-charge-list"
        title="Accessorial charge"
        formComponent={null}
      />
    );
  }

  return (
    <FormCreatePanel<ChargeForm, typeof ROW>
      open
      onOpenChange={() => undefined}
      form={form}
      mutationFn={async (values) => values}
      queryKey="accessorial-charge-list"
      title="Accessorial charge"
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

  it("tells its caller once a record is created, with what the create returned", async () => {
    const onCreated = vi.fn();
    function Panel() {
      const form = useForm<ChargeForm>({ defaultValues: { ...PRISTINE, code: "FSC" } });
      return (
        <FormCreatePanel<ChargeForm, typeof ROW, ChargeForm, { id: string }>
          open
          onOpenChange={() => undefined}
          form={form}
          queryKey="accessorial-charge-list"
          title="Accessorial charge"
          formComponent={null}
          mutationFn={async () => ({ id: "acc_9" })}
          onCreated={onCreated}
        />
      );
    }
    render(<Panel />, { wrapper: wrapper(queryClient) });

    fireEvent.submit(document.getElementById("panel-create-form")!);

    await waitFor(() => expect(onCreated).toHaveBeenCalledWith({ id: "acc_9" }));
  });

  // Each save bumps the record's version and refuses a stale one. A panel left open
  // after Save must edit the saved version, or its next save is refused as out of date.
  it("sends the version the last save returned when saved again without closing", async () => {
    window.localStorage.setItem("panel-edit-default-action", JSON.stringify("save"));
    type VersionedForm = ChargeForm & { version: number };
    const row = { ...ROW, version: 3 };
    const sent: number[] = [];
    let form!: UseFormReturn<VersionedForm>;
    function Panel() {
      form = useForm<VersionedForm>({ defaultValues: { ...PRISTINE, version: 0 } });
      return (
        <FormEditPanel<VersionedForm, typeof row & Record<string, unknown>>
          open
          onOpenChange={() => undefined}
          row={row as typeof row & Record<string, unknown>}
          form={form}
          mutationFn={async (values) => {
            sent.push(values.version);
            return { ...values, version: values.version + 1 };
          }}
          queryKey="accessorial-charge-list"
          title="Accessorial charge"
          formComponent={null}
        />
      );
    }
    try {
      render(<Panel />, { wrapper: wrapper(queryClient) });
      await waitFor(() => expect(form.getValues().version).toBe(3));

      fireEvent.submit(document.getElementById("panel-edit-form")!);
      await waitFor(() => expect(form.getValues().version).toBe(4));
      expect(form.formState.isDirty).toBe(false);

      fireEvent.submit(document.getElementById("panel-edit-form")!);
      await waitFor(() => expect(sent).toEqual([3, 4]));
    } finally {
      window.localStorage.removeItem("panel-edit-default-action");
    }
  });
});
