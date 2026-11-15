import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { DocumentStorageStatus } from "@/types/document-control";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DocumentStorageAlert } from "../document-storage-alert";

const mocks = vi.hoisted(() => ({
  granted: new Set<string>(),
  status: null as DocumentStorageStatus | null,
  fetched: 0,
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (resource: string, operation: string) => ({
    allowed: mocks.granted.has(`${resource}:${operation}`),
    isLoading: false,
  }),
}));

vi.mock("@/lib/queries/document-control", () => ({
  documentControl: {
    storageStatus: () => ({
      queryKey: ["documentControl", "storage-status"],
      queryFn: async () => {
        mocks.fetched++;
        return mocks.status;
      },
    }),
  },
}));

const GIB = 1024 * 1024 * 1024;

function status(overrides: Partial<DocumentStorageStatus>): DocumentStorageStatus {
  return {
    managed: true,
    used: 0,
    included: 50 * GIB,
    overage: false,
    warning: false,
    stopped: false,
    ...overrides,
  };
}

function grantAdministrator() {
  mocks.granted.add(`${Resource.Organization}:${Operation.Update}`);
}

afterEach(() => {
  cleanup();
  mocks.granted.clear();
  mocks.status = null;
  mocks.fetched = 0;
});

function renderAlert() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <QueryClientProvider client={client}>
      <DocumentStorageAlert />
    </QueryClientProvider>,
  );
}

describe("DocumentStorageAlert", () => {
  it("warns an administrator past four fifths with the used and included amounts", async () => {
    grantAdministrator();
    mocks.status = status({ used: 41 * GIB, warning: true });

    renderAlert();

    expect(await screen.findByText("Document storage is nearly full")).toBeInTheDocument();
    expect(
      screen.getByText("41.0 GB of the 50.0 GB your plan includes is used."),
    ).toBeInTheDocument();
  });

  it("says uploads stopped and offers both ways out at the limit with overage off", async () => {
    grantAdministrator();
    mocks.status = status({ used: 50 * GIB, warning: true, stopped: true });

    renderAlert();

    expect(await screen.findByText("Document storage is full")).toBeInTheDocument();
    expect(
      screen.getByText(
        "Turn on overage from your Trenova account to keep uploading, or delete documents you no longer need.",
      ),
    ).toBeInTheDocument();
  });

  it("asks nothing of the server for someone who is not an administrator", () => {
    mocks.status = status({ used: 50 * GIB, warning: true, stopped: true });

    const { container } = renderAlert();

    expect(container).toBeEmptyDOMElement();
    expect(mocks.fetched).toBe(0);
  });

  it("renders nothing below the warning share or on an instance that is not managed", async () => {
    grantAdministrator();
    mocks.status = status({ managed: false });

    const { container } = renderAlert();

    await vi.waitFor(() => expect(mocks.fetched).toBe(1));
    expect(container).toBeEmptyDOMElement();
  });
});
