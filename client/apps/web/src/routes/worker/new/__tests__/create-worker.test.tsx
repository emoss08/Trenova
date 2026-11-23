import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { workerFormValues } from "../../[id]/_components/profile/worker-form-values";
import { mariaRecord } from "../../[id]/_components/__tests__/worker-fixtures";
import { WorkerCreatePage } from "../page";

const mocks = vi.hoisted(() => ({
  create: vi.fn(),
  navigate: vi.fn(),
  closePopout: vi.fn(),
  isPopout: false,
}));

vi.mock("@/services/worker", () => ({ workerService: { create: mocks.create } }));
vi.mock("@tanstack/react-router", () => ({
  useNavigate: () => mocks.navigate,
  useBlocker: () => ({ status: "idle" }),
}));
vi.mock("@trenova/shared/components/link", () => ({
  Link: ({ children }: { children?: ReactNode }) => <a href="/hr/workers">{children}</a>,
}));
vi.mock("@/hooks/popout-window/use-popout-window", () => ({
  usePopoutWindow: () => ({ isPopout: mocks.isPopout, closePopout: mocks.closePopout }),
}));
vi.mock("@/components/navigation/sidebar-layout", () => ({
  PageLayout: ({ children }: { children: ReactNode }) => <div>{children}</div>,
}));
vi.mock("../../[id]/_components/profile/worker-profile-cards", () => ({
  WorkerProfileCards: ({ note }: { note?: ReactNode }) => <div>{note}</div>,
  NewWorkerNote: () => <p>the onboarding checklist starts</p>,
}));
vi.mock("../../[id]/_components/profile/worker-form-values", async (importOriginal) => {
  const actual =
    await importOriginal<typeof import("../../[id]/_components/profile/worker-form-values")>();
  const { mariaRecord: record } = await import("../../[id]/_components/__tests__/worker-fixtures");
  const filled = actual.workerFormValues(record());
  return {
    ...actual,
    NEW_WORKER_VALUES: { ...filled, id: undefined, version: undefined },
  };
});

let client: QueryClient;

beforeEach(() => {
  client = new QueryClient();
  mocks.create.mockReset().mockResolvedValue({ ...workerFormValues(mariaRecord()), id: "wrk_9" });
  mocks.navigate.mockReset();
  mocks.closePopout.mockReset();
  mocks.isPopout = false;
});
afterEach(cleanup);

function renderPage() {
  render(
    <QueryClientProvider client={client}>
      <WorkerCreatePage />
    </QueryClientProvider>,
  );
}

describe("New worker", () => {
  it("creates the worker and opens their page in place of the form", async () => {
    renderPage();

    fireEvent.click(screen.getByRole("button", { name: "Create worker" }));

    await waitFor(() => expect(mocks.create).toHaveBeenCalledTimes(1));
    expect(mocks.create.mock.calls[0]?.[0]).toMatchObject({
      firstName: "Maria",
      lastName: "Alvarez",
    });
    await waitFor(() =>
      expect(mocks.navigate).toHaveBeenCalledWith({ to: "/hr/workers/wrk_9", replace: true }),
    );
  });

  it("closes the popout instead of navigating when it was opened in one", async () => {
    mocks.isPopout = true;
    renderPage();

    fireEvent.click(screen.getByRole("button", { name: "Create worker" }));

    await waitFor(() => expect(mocks.closePopout).toHaveBeenCalledTimes(1));
    expect(mocks.navigate).not.toHaveBeenCalled();
  });

  it("says what creating the worker starts", () => {
    renderPage();

    expect(screen.getByText("the onboarding checklist starts")).toBeTruthy();
  });
});
