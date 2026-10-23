import type { PageAccess, PageAccessScreen, PageAccessSubject } from "@/lib/graphql/page-access";
import { toRoleDraft } from "@/lib/role-draft";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { RoleScreenPreview } from "../role-screen-preview";

const fetchPageAccessScreens = vi.fn<() => Promise<PageAccessScreen[]>>();
const fetchPageAccessPreview =
  vi.fn<(location: string, subject: PageAccessSubject) => Promise<PageAccess | null>>();

vi.mock("@/lib/graphql/page-access", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/page-access")>()),
  fetchPageAccessScreens: () => fetchPageAccessScreens(),
  fetchPageAccessPreview: (location: string, subject: PageAccessSubject) =>
    fetchPageAccessPreview(location, subject),
}));

// The screen picker is the shared Select; this stands in for it so a test can
// choose a screen without driving its popup.
vi.mock("@trenova/shared/components/ui/select", () => ({
  Select: ({
    items,
    onValueChange,
    children,
  }: {
    items: { value: string; label: string }[];
    onValueChange: (value: string) => void;
    children: ReactNode;
  }) => (
    <div>
      {items.map((item) => (
        <button key={item.value} type="button" onClick={() => onValueChange(item.value)}>
          {item.label}
        </button>
      ))}
      {children}
    </div>
  ),
  SelectTrigger: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SelectValue: ({ placeholder }: { placeholder: string }) => <span>{placeholder}</span>,
  SelectContent: () => null,
  SelectItem: () => null,
}));

afterEach(() => {
  cleanup();
  fetchPageAccessScreens.mockReset();
  fetchPageAccessPreview.mockReset();
});

const preview: PageAccess = {
  path: "/shipments/:id",
  name: "",
  operations: [],
  requires: [
    {
      resource: "shipment",
      resourceName: "Shipment",
      operation: "read",
      operationName: "View",
      allowed: false,
    },
  ],
  endpoints: [],
  kinds: [],
  withheld: 0,
  blocked: true,
};

function renderPreview(draft = toRoleDraft("internal", [])) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <RoleScreenPreview draft={draft} />
    </QueryClientProvider>,
  );
}

describe("RoleScreenPreview", () => {
  it("asks for nothing until a screen is chosen", async () => {
    fetchPageAccessScreens.mockResolvedValue([{ path: "/shipments", name: "Shipments" }]);
    renderPreview();

    expect(await screen.findByRole("button", { name: "Shipments" })).toBeInTheDocument();
    expect(
      screen.getByText("Choose a screen to see what this role would need and see there."),
    ).toBeInTheDocument();
    expect(fetchPageAccessPreview).not.toHaveBeenCalled();
  });

  it("previews the chosen screen as the role the form holds", async () => {
    fetchPageAccessScreens.mockResolvedValue([
      { path: "/shipments/:id", name: "" },
      { path: "/shipments", name: "Shipments" },
    ]);
    fetchPageAccessPreview.mockResolvedValue(preview);
    const draft = toRoleDraft("restricted", [{ resource: "customer", operations: ["read"] }]);
    renderPreview(draft);

    (await screen.findByRole("button", { name: "/shipments/:id" })).click();

    await waitFor(() =>
      expect(fetchPageAccessPreview).toHaveBeenCalledWith("/shipments/:id", { draft }),
    );
    expect(await screen.findByText("View on Shipment")).toBeInTheDocument();
    expect(screen.getByText(/This screen would not open/)).toBeInTheDocument();
  });
});
