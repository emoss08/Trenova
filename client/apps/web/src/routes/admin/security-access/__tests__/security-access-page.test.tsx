import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { User } from "@trenova/shared/types/user";
import { domMax, LazyMotion } from "motion/react";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { SecurityAccessPage } from "../page";

// Every read the console makes waits, so the test sees what is drawn before any data.
vi.mock("@/services/organization", () => ({
  organizationService: new Proxy({}, { get: () => () => new Promise(() => {}) }),
}));

vi.mock("@/components/navigation/sidebar-layout", () => ({
  PageLayout: ({
    pageHeaderProps,
    children,
  }: {
    pageHeaderProps: { title: string; description: string };
    children: React.ReactNode;
  }) => (
    <main>
      <h1>{pageHeaderProps.title}</h1>
      <p>{pageHeaderProps.description}</p>
      {children}
    </main>
  ),
}));

function signIn(currentOrganizationId: string | null = "org_01") {
  useAuthStore.setState({
    user: { currentOrganizationId, memberships: [] } as unknown as User,
    isAuthenticated: true,
  });
}

function renderPage(search = "") {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <TestRouter search={search}>
      <LazyMotion features={domMax}>
        <QueryClientProvider client={queryClient}>
          <SecurityAccessPage />
        </QueryClientProvider>
      </LazyMotion>
    </TestRouter>,
  );
}

// The page draws its body lazily; loading the module first keeps a cold first import from
// outlasting the queries' wait.
beforeAll(async () => {
  await import("../_components/security-access-body");
});

afterEach(() => {
  cleanup();
  clearPermissions();
  useAuthStore.setState({ user: null, isAuthenticated: false });
});

describe("the security and access page", () => {
  it("is headed Security & access", async () => {
    signIn();
    seedPermissions({ [Resource.AccessPolicy]: Operation.Read });

    renderPage();

    expect(screen.getByRole("heading", { name: "Security & access" })).toBeInTheDocument();
  });

  it("draws the console's own tab row with the parts a person may read", async () => {
    signIn();
    seedPermissions({
      [Resource.AccessPolicy]: Operation.Read,
      [Resource.AuthEvent]: Operation.Read,
    });

    renderPage();

    await screen.findByRole("tab", { name: "Policies" });
    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual([
      "Policies",
      "Activity",
    ]);
    expect(screen.getByRole("tab", { name: "Policies", selected: true })).toBeInTheDocument();
  });

  it("opens the tab the address names", async () => {
    signIn();
    seedPermissions({
      [Resource.AccessPolicy]: Operation.Read,
      [Resource.AuthEvent]: Operation.Read,
    });

    renderPage("?tab=activity");

    expect(
      await screen.findByRole("tab", { name: "Activity", selected: true }),
    ).toBeInTheDocument();
  });

  it("says so when the person may read none of it", async () => {
    signIn();
    seedPermissions({ [Resource.Shipment]: Operation.Read });

    renderPage();

    expect(
      await screen.findByText(
        "You don't have access to any of this organization's security settings.",
      ),
    ).toBeInTheDocument();
    expect(screen.queryByRole("tablist")).toBeNull();
  });

  it("says so when the session has no organization", async () => {
    signIn(null);
    seedPermissions({ [Resource.AccessPolicy]: Operation.Read });

    renderPage();

    expect(
      await screen.findByText("No active organization found for this session."),
    ).toBeInTheDocument();
  });
});
