import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { useForm } from "react-hook-form";
import { describe, expect, it, vi } from "vitest";
import { FormEditPanel } from "@/components/form-edit-panel";

vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }));

type CarrierForm = { name: string };

const ROW = { id: "car_1", name: "Swift Haulers" } as const;

function Panel({ withSubtitle }: { withSubtitle: boolean }) {
  const form = useForm<CarrierForm>({ defaultValues: { name: "" } });
  return (
    <FormEditPanel<CarrierForm, typeof ROW & Record<string, unknown>>
      open
      onOpenChange={() => undefined}
      row={ROW as unknown as typeof ROW & Record<string, unknown>}
      form={form}
      url="/carriers/"
      queryKey="carrier-list"
      title="Carrier"
      fieldKey="name"
      formComponent={null}
      subtitle={withSubtitle ? (record) => <p>Synced for {String(record.id)}</p> : undefined}
    />
  );
}

function renderPanel(withSubtitle: boolean) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <Panel withSubtitle={withSubtitle} />
    </QueryClientProvider>,
  );
}

describe("FormEditPanel subtitle", () => {
  it("renders the subtitle for the open record under its title", async () => {
    renderPanel(true);

    const title = await screen.findByText("Swift Haulers");
    const subtitle = screen.getByText("Synced for car_1");
    expect(title.parentElement).toContainElement(subtitle);
  });

  it("keeps the default title when no subtitle is given", async () => {
    renderPanel(false);

    expect(await screen.findByText("Swift Haulers")).toBeInTheDocument();
    expect(screen.queryByText(/Synced for/)).not.toBeInTheDocument();
  });
});
