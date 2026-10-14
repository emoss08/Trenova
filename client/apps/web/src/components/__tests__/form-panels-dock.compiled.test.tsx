import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { Controller, useForm } from "react-hook-form";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";

// Runs under the React Compiler, as the app is built: the panels' save dock and change
// count must follow the form as it is edited.

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

function DescriptionField() {
  return (
    <Controller<ChargeForm>
      name="description"
      render={({ field }) => (
        <input
          aria-label="Description"
          name={field.name}
          value={String(field.value ?? "")}
          onChange={field.onChange}
          onBlur={field.onBlur}
          ref={field.ref}
        />
      )}
    />
  );
}

function Panel({ mode, useDock }: { mode: "create" | "edit"; useDock: boolean }) {
  const form = useForm<ChargeForm>({ defaultValues: PRISTINE });

  if (mode === "edit") {
    return (
      <FormEditPanel<ChargeForm, typeof ROW & Record<string, unknown>>
        open
        onOpenChange={() => undefined}
        row={ROW as unknown as typeof ROW & Record<string, unknown>}
        form={form}
        url="/accessorial-charges/"
        queryKey="accessorial-charge-list"
        title="Accessorial charge"
        formComponent={<DescriptionField />}
        useDock={useDock}
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
      title="Accessorial charge"
      formComponent={<DescriptionField />}
      useDock={useDock}
    />
  );
}

function dockSave() {
  return screen.queryByRole("button", { name: /^Save/ });
}

describe("form panels under the React Compiler", () => {
  let queryClient: QueryClient;

  function renderPanel(mode: "create" | "edit", useDock: boolean) {
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
    return render(<Panel mode={mode} useDock={useDock} />, { wrapper });
  }

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
  });

  afterEach(() => {
    queryClient.clear();
    vi.clearAllMocks();
  });

  it("opens the create panel's save dock once a field is changed", async () => {
    renderPanel("create", true);
    const description = await screen.findByLabelText("Description");
    expect(dockSave()).toBeNull();

    fireEvent.change(description, { target: { value: "Layover" } });

    expect(await screen.findByRole("button", { name: /^Save/ })).toBeInTheDocument();
  });

  it("opens the edit panel's save dock once the loaded record is changed", async () => {
    renderPanel("edit", true);
    const description = await screen.findByLabelText("Description");
    await waitFor(() => expect(description).toHaveValue("Detention"));
    expect(dockSave()).toBeNull();

    fireEvent.change(description, { target: { value: "Detention (driver)" } });

    expect(await screen.findByRole("button", { name: /^Save/ })).toBeInTheDocument();
  });

  it("closes the edit panel's save dock when the change is typed back", async () => {
    renderPanel("edit", true);
    const description = await screen.findByLabelText("Description");
    await waitFor(() => expect(description).toHaveValue("Detention"));

    fireEvent.change(description, { target: { value: "Detention (driver)" } });
    await screen.findByRole("button", { name: /^Save/ });
    fireEvent.change(description, { target: { value: "Detention" } });

    await waitFor(() => expect(dockSave()).toBeNull());
  });

  it("counts the edit panel's unsaved changes in its footer", async () => {
    renderPanel("edit", false);
    const description = await screen.findByLabelText("Description");
    await waitFor(() => expect(description).toHaveValue("Detention"));
    expect(screen.getByText("No changes")).toBeInTheDocument();

    fireEvent.change(description, { target: { value: "Detention (driver)" } });

    expect(await screen.findByRole("button", { name: "1 unsaved change" })).toBeInTheDocument();
  });
});
