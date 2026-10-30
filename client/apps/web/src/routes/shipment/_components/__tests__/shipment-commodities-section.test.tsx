import { recordPath } from "@/config/record-links";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { Shipment } from "@trenova/shared/types/shipment";
import { useEffect, useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import CommoditiesSection from "../shipment-commodities";

const { getUIPolicy, checkHazmatSegregation } = vi.hoisted(() => ({
  getUIPolicy: vi.fn(),
  checkHazmatSegregation: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiService: { shipmentService: { getUIPolicy, checkHazmatSegregation } },
}));
vi.mock("@/components/autocomplete-fields", () => ({
  CommodityAutocompleteField: () => null,
}));

// Two lines as the shipment read returns them: a hazardous, fragile commodity and a plain one.
const LINES = [
  {
    commodityId: "com_1",
    pieces: 2,
    weight: 1000,
    commodity: {
      id: "com_1",
      name: "Lithium batteries",
      hazardousMaterialId: "hm_1",
      stackable: false,
      fragile: true,
    },
  },
  {
    commodityId: "com_2",
    pieces: 1,
    weight: 500,
    commodity: { id: "com_2", name: "Paper rolls", stackable: true, fragile: false },
  },
] as Shipment["commodities"];

function Harness({
  lines = LINES,
  weightError,
}: {
  lines?: Shipment["commodities"];
  weightError?: string;
}) {
  const form = useForm<Shipment>({ defaultValues: { commodities: lines } as Shipment });
  const [client] = useState(
    () => new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } }),
  );
  useEffect(() => {
    if (weightError) form.setError("commodities.0.weight", { message: weightError });
  }, [form, weightError]);
  return (
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <FormProvider {...form}>
          <CommoditiesSection />
        </FormProvider>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

afterEach(cleanup);

beforeEach(() => {
  vi.clearAllMocks();
  getUIPolicy.mockResolvedValue({ maxShipmentWeightLimit: 45000, checkHazmatSegregation: true });
});

describe("CommoditiesSection", () => {
  it("names each line's commodity as a link to it, with its handling flags", () => {
    render(<Harness />);

    expect(screen.getByRole("link", { name: "Lithium batteries" })).toHaveAttribute(
      "href",
      recordPath("commodity", "com_1"),
    );
    expect(
      screen.getByRole("img", { name: "Commodity is classified as hazardous material." }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("img", { name: "Commodity is marked as fragile." }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("img", { name: "Commodity is marked as stackable." }),
    ).toBeInTheDocument();
  });

  it("totals the pieces and weight against the organisation's limit", async () => {
    render(<Harness />);

    expect(screen.getByText("3 pieces total")).toBeInTheDocument();
    expect(screen.getByText("1,500 lbs")).toBeInTheDocument();
    expect(await screen.findByText("/ 45,000 lbs")).toBeInTheDocument();
  });

  it("names each line's edit and remove buttons for its commodity, and removes the line", async () => {
    render(<Harness />);

    expect(screen.getByRole("button", { name: "Edit Paper rolls" })).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Remove Lithium batteries" }));

    expect(screen.queryByRole("link", { name: "Lithium batteries" })).not.toBeInTheDocument();
    expect(screen.getByText("1 piece total")).toBeInTheDocument();
  });

  it("marks a line with problems and says what they are on hover", async () => {
    render(<Harness weightError="Weight cannot be negative" />);

    const problems = await screen.findByRole("img", { name: "This commodity has problems" });
    await userEvent.hover(problems);

    expect(await screen.findByText("Weight cannot be negative")).toBeInTheDocument();
  });

  it("keeps the dialog open and saves nothing when the hazardous material check cannot run", async () => {
    checkHazmatSegregation.mockRejectedValue(new Error("offline"));
    render(<Harness />);
    await screen.findByText("/ 45,000 lbs");

    await userEvent.click(screen.getByRole("button", { name: "Edit Lithium batteries" }));
    const dialog = await screen.findByRole("dialog", { name: "Edit commodity" });
    await userEvent.click(within(dialog).getByRole("button", { name: "Save" }));

    expect(
      await within(dialog).findByText(
        "The hazardous material check could not run, so the commodity was not saved. Try again.",
      ),
    ).toBeInTheDocument();
    expect(checkHazmatSegregation).toHaveBeenCalledWith(["com_1", "com_2"]);
    expect(screen.getByRole("dialog", { name: "Edit commodity" })).toBeInTheDocument();
  });

  it("closes the dialog once the hazardous material check clears the lines", async () => {
    checkHazmatSegregation.mockResolvedValue(undefined);
    render(<Harness />);
    await screen.findByText("/ 45,000 lbs");

    await userEvent.click(screen.getByRole("button", { name: "Edit Lithium batteries" }));
    const dialog = await screen.findByRole("dialog", { name: "Edit commodity" });
    await userEvent.click(within(dialog).getByRole("button", { name: "Save" }));

    await waitFor(() =>
      expect(screen.queryByRole("dialog", { name: "Edit commodity" })).not.toBeInTheDocument(),
    );
  });

  it("removes a line added and then cancelled", async () => {
    render(<Harness />);

    await userEvent.click(screen.getByRole("button", { name: "Add commodity" }));
    const dialog = await screen.findByRole("dialog", { name: "Add commodity" });
    expect(screen.getByText("4 pieces total")).toBeInTheDocument();
    await userEvent.click(within(dialog).getByRole("button", { name: "Cancel" }));

    await waitFor(() => expect(screen.getByText("3 pieces total")).toBeInTheDocument());
  });
});
