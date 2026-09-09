import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { useController, type Control } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";

// The tractor panel saves over REST. Whatever the fuel and IFTA controls put in
// form state has to reach the request body, or the columns keep their database
// defaults and the save looks like it silently did nothing.

type PickerProps = { control: Control; name: string; label: string };

function pickerStub() {
  return function Picker({ control, name, label }: PickerProps) {
    const { field } = useController({ control, name });
    return (
      <input
        aria-label={label}
        value={(field.value as string) ?? ""}
        onChange={(event) => field.onChange(event.target.value)}
      />
    );
  };
}

vi.mock("@/components/autocomplete-fields", () => ({
  EquipmentManufacturerAutocompleteField: pickerStub(),
  EquipmentTypeAutocompleteField: pickerStub(),
  FleetCodeAutocompleteField: pickerStub(),
  IftaFuelTypeAutocompleteField: pickerStub(),
  UsStateAutocompleteField: pickerStub(),
  WorkerAutocompleteField: pickerStub(),
}));

vi.mock("@/components/custom-fields-section", () => ({ CustomFieldsSection: () => null }));
vi.mock("@/components/documents/documents-tab", () => ({ default: () => null }));
vi.mock("../tractor-inspections-tab", () => ({ default: () => null }));

const post = vi.fn(async (_url: string, _body?: unknown) => ({ id: "trac_1" }));
const put = vi.fn(async (_url: string, _body?: unknown) => ({ id: "trac_77" }));

vi.mock("@trenova/shared/lib/api", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return { ...actual, api: { post, put, get: vi.fn(async () => ({})) } };
});

const { TractorPanel, buildTractorDefaults } = await import("../tractor-panel");

function newQueryClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 }, mutations: { retry: false } },
  });
}

describe("TractorPanel fuel and IFTA fields", () => {
  afterEach(() => {
    cleanup();
    post.mockClear();
    put.mockClear();
  });

  it("sends the chosen fuel type and IFTA flag when creating", async () => {
    const user = userEvent.setup();
    render(
      <QueryClientProvider client={newQueryClient()}>
        <TractorPanel open mode="create" row={null} onOpenChange={() => {}} />
      </QueryClientProvider>,
    );

    await user.type(screen.getByPlaceholderText("Code"), "TRC-900");
    await user.type(screen.getByLabelText(/^Equipment Type/), "et_1");
    await user.type(screen.getByLabelText(/^Equip. Manufacturer/), "em_1");
    await user.type(screen.getByLabelText(/^Primary Worker/), "wrk_1");
    await user.clear(screen.getByLabelText(/^Fuel Type/));
    await user.type(screen.getByLabelText(/^Fuel Type/), "Gasoline");
    await user.click(screen.getByRole("switch", { name: /IFTA qualified/i }));

    await user.click(screen.getByRole("button", { name: /save/i }));

    await waitFor(() => {
      expect(post).toHaveBeenCalled();
    });

    const [, body] = post.mock.calls[0];
    expect(body).toMatchObject({ fuelType: "Gasoline", iftaQualified: false });
  });

  it("sends a changed fuel type and IFTA flag when editing", async () => {
    const user = userEvent.setup();
    const row = {
      ...buildTractorDefaults(),
      id: "trac_77",
      version: 4,
      code: "TRC-103",
      equipmentTypeId: "et_1",
      equipmentManufacturerId: "em_1",
      primaryWorkerId: "wrk_1",
      fuelType: "Diesel",
      iftaQualified: true,
    } as never;

    render(
      <NuqsTestingAdapter>
        <QueryClientProvider client={newQueryClient()}>
          <TractorPanel open mode="edit" row={row} onOpenChange={() => {}} />
        </QueryClientProvider>
      </NuqsTestingAdapter>,
    );

    await user.clear(screen.getByLabelText(/^Fuel Type/));
    await user.type(screen.getByLabelText(/^Fuel Type/), "Biodiesel");
    await user.click(screen.getByRole("switch", { name: /IFTA qualified/i }));

    const [saveButton] = screen.getAllByRole("button", { name: /save & close/i });
    await user.click(saveButton);

    await waitFor(() => {
      expect(put).toHaveBeenCalled();
    });

    const [url, body] = put.mock.calls[0];
    expect(url).toContain("trac_77");
    expect(body).toMatchObject({ fuelType: "Biodiesel", iftaQualified: false });
  });
});
