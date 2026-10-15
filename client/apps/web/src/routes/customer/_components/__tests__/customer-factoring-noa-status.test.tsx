import type { FactoringNoa } from "@/lib/graphql/factoring-noa";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { Customer } from "@trenova/shared/types/customer";
import type { ReactNode } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CustomerFactoringNoa } from "../customer-factoring-noa";

/*
The customer form shows where the customer's notice of assignment stands for
every customer with a factor, but the dialog that sends one is wanted rarely.
Its code (a form, a document picker and the notice mutations) is read when
the pointer or focus reaches Send NOA, never with the customers page.
*/

const mocks = vi.hoisted(() => ({
  loaded: [] as string[],
  listFactoringNoas: vi.fn(),
}));

vi.mock("@/components/factoring/factoring-noa-dialogs", async (importOriginal) => {
  mocks.loaded.push("noa-dialogs");
  return importOriginal();
});
vi.mock("@/lib/graphql/factoring-noa", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/factoring-noa")>()),
  listFactoringNoas: mocks.listFactoringNoas,
}));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

const draft: FactoringNoa = {
  id: "facnoa_1",
  customerId: "cus_1",
  factoringCompanyId: "fac_rts",
  status: "Draft",
  sentAt: null,
  sentTo: null,
  acknowledgedAt: null,
  signedDocumentId: null,
  expiresAt: null,
  releasedAt: null,
  releaseReason: null,
  notes: null,
  version: 1,
  createdAt: 1_789_000_000,
  updatedAt: 1_789_000_000,
  customer: { id: "cus_1", code: "ACME", name: "Acme Foods" },
  factoringCompany: { id: "fac_rts", code: "RTS", name: "RTS Financial" },
};

function CustomerForm({ children }: { children: ReactNode }) {
  const form = useForm<Customer>({
    defaultValues: {
      id: "cus_1",
      billingProfile: { factoringCompanyId: "fac_rts" },
    } as Customer,
  });
  return <FormProvider {...form}>{children}</FormProvider>;
}

describe("CustomerFactoringNoa", () => {
  it("reads the send dialog when the pointer reaches Send NOA, then opens it on the click", async () => {
    mocks.listFactoringNoas.mockResolvedValue([draft]);
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <MemoryRouter>
          <CustomerForm>
            <CustomerFactoringNoa />
          </CustomerForm>
        </MemoryRouter>
      </QueryClientProvider>,
    );

    const send = await screen.findByRole("button", { name: "Send NOA" });
    expect(screen.getByText("Drafted, not sent yet")).toBeInTheDocument();
    expect(mocks.listFactoringNoas).toHaveBeenCalledWith(
      { customerId: "cus_1", factoringCompanyId: "fac_rts" },
      expect.objectContaining({ signal: expect.any(AbortSignal) }),
    );
    expect(mocks.loaded).toEqual([]);

    await userEvent.hover(send);
    await waitFor(() => expect(mocks.loaded).toEqual(["noa-dialogs"]));
    await userEvent.click(send);

    const dialog = await screen.findByRole("dialog", {}, { timeout: 10_000 });
    expect(
      within(dialog).getByRole("heading", { name: "Send notice of assignment" }),
    ).toBeInTheDocument();
  }, 20_000);
});
