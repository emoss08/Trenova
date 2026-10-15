import type { FactoringCompany } from "@/types/factoring-company";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { FactoringCompanyPanel } from "../factoring-company-panel";

/*
A factor's panel opens on its form. The five sections below it (agreements,
notices, delivery, reserve, remittances) bring their own dialogs and queries,
so each is read only once the panel opens, and while it is read the panel
draws that section's own outline in its place.
*/

vi.mock("../factoring-agreements-section", () => new Promise(() => {}));
vi.mock("../factoring-notices-section", () => new Promise(() => {}));
vi.mock("../factoring-delivery-section", () => new Promise(() => {}));
vi.mock("../factoring-reserve-section", () => new Promise(() => {}));
vi.mock("../factoring-remittances-section", () => new Promise(() => {}));
vi.mock("@/components/autocomplete-fields", () => ({ UsStateAutocompleteField: () => null }));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

afterEach(cleanup);

const factor = {
  id: "fac_rts",
  status: "Active",
  code: "RTS",
  name: "RTS Financial",
  addressLine1: "9300 Metcalf Ave",
  addressLine2: "",
  city: "Overland Park",
  stateId: "st_ks",
  postalCode: "66212",
  email: "schedules@rtsfinancial.com",
  phoneNumber: "",
  remittanceInstructions: "",
  noticeOfAssignment: "",
  notes: "",
  nonRecourse: false,
  version: 2,
  updatedAt: 1_789_000_000,
} as FactoringCompany;

describe("FactoringCompanyPanel", () => {
  it("draws each section's outline while the sections' code is read", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <NuqsTestingAdapter hasMemory>
          <MemoryRouter>
            <FactoringCompanyPanel open onOpenChange={vi.fn()} mode="edit" row={factor} />
          </MemoryRouter>
        </NuqsTestingAdapter>
      </QueryClientProvider>,
    );

    for (const label of [
      "Loading agreements",
      "Loading notices of assignment",
      "Loading delivery settings",
      "Loading the reserve",
      "Loading remittances",
    ]) {
      expect(await screen.findByRole("region", { name: label })).toHaveAttribute(
        "aria-busy",
        "true",
      );
    }
  });
});
