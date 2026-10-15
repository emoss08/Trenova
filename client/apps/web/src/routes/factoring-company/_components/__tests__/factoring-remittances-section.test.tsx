import type { FactoringRemittance } from "@/lib/graphql/factoring-remittance";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { FactoringRemittancesSection } from "../factoring-remittances-section";

const mocks = vi.hoisted(() => ({
  loaded: [] as string[],
  listFactoringRemittances: vi.fn(),
  granted: new Set<number>(),
}));

vi.mock("@/lib/graphql/factoring-remittance", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/factoring-remittance")>()),
  listFactoringRemittances: mocks.listFactoringRemittances,
}));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: (_resource: string, operation: number) => ({
    allowed: mocks.granted.has(operation),
    isLoading: false,
  }),
}));
vi.mock("@/components/factoring/import-remittance-dialog", () => {
  mocks.loaded.push("import-dialog");
  return {
    ImportRemittanceDialog: ({ factoringCompanyId }: { factoringCompanyId?: string }) => (
      <div role="dialog" aria-label="Import remittance">
        {factoringCompanyId}
      </div>
    ),
  };
});

const READ = 1 << 0;
const CREATE = 1 << 1;

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  mocks.granted.clear();
});

function remittance(index: number): FactoringRemittance {
  return {
    id: `facrem_${index}`,
    factoringCompanyId: "fac_rts",
    fileName: `rts-${index}.csv`,
    status: "Applied",
    remittedOn: null,
    createdAt: 1_800_000_000,
    lineCount: index,
    totalCashMinor: 102_345,
    currencyCode: "USD",
  } as FactoringRemittance;
}

function renderSection() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <FactoringRemittancesSection factoringCompanyId="fac_rts" />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("FactoringRemittancesSection", () => {
  it("lists the factor's five newest remittances, each linking to its review", async () => {
    mocks.granted.add(READ);
    mocks.listFactoringRemittances.mockResolvedValue({
      remittances: [1, 2, 3, 4, 5, 6].map(remittance),
      totalCount: 6,
    });

    renderSection();

    const first = await screen.findByRole("link", { name: "rts-1.csv" });
    expect(first.getAttribute("href")).toBe("/billing/factoring/remittances/facrem_1");
    expect(screen.getAllByRole("link")).toHaveLength(5);
    expect(screen.queryByText("rts-6.csv")).toBeNull();
    expect(screen.getAllByText("$1,023.45")).toHaveLength(5);
    expect(mocks.listFactoringRemittances).toHaveBeenCalledWith(
      { factoringCompanyId: "fac_rts" },
      expect.anything(),
    );
    expect(screen.queryByRole("button", { name: "Import remittance" })).toBeNull();
  });

  it("offers an import to someone who may create remittances", async () => {
    mocks.granted.add(READ);
    mocks.granted.add(CREATE);
    mocks.listFactoringRemittances.mockResolvedValue({ remittances: [], totalCount: 0 });

    renderSection();

    expect(
      await screen.findByText(
        "No remittance from this factor yet. Import one, or forward the factor's emails to your inbox.",
      ),
    ).toBeTruthy();
    expect(screen.getByRole("button", { name: "Import remittance" })).toBeTruthy();
  });

  it("reads the import dialog when the pointer reaches Import, and opens it for this factor", async () => {
    mocks.granted.add(READ);
    mocks.granted.add(CREATE);
    mocks.listFactoringRemittances.mockResolvedValue({ remittances: [], totalCount: 0 });

    renderSection();

    const importButton = await screen.findByRole("button", { name: "Import remittance" });
    expect(mocks.loaded).toEqual([]);

    await userEvent.hover(importButton);
    await waitFor(() => expect(mocks.loaded).toEqual(["import-dialog"]));
    await userEvent.click(importButton);

    expect((await screen.findByRole("dialog", { name: "Import remittance" })).textContent).toBe(
      "fac_rts",
    );
  });

  it("draws the list's own skeleton while the remittances are read", async () => {
    mocks.granted.add(READ);
    mocks.listFactoringRemittances.mockImplementation(() => new Promise<never>(() => {}));

    renderSection();

    expect(await screen.findByLabelText("Loading remittances")).toHaveAttribute(
      "aria-busy",
      "true",
    );
    expect(screen.getByRole("region", { name: "Remittances" })).toBeTruthy();
  });

  it("shows nothing and reads nothing for someone who cannot see remittances", () => {
    const { container } = renderSection();

    expect(container.textContent).toBe("");
    expect(mocks.listFactoringRemittances).not.toHaveBeenCalled();
  });
});
