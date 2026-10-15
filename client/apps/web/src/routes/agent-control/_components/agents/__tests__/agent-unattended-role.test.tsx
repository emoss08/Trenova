import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AgentUnattendedRole } from "../agent-unattended-role";

const listRoles = vi.fn();

vi.mock("@/lib/role-api", () => ({ listRoles: (params: unknown) => listRoles(params) }));
vi.mock("@/lib/graphql/agent-access", () => ({ setAgentUnattendedRole: vi.fn() }));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: false, isLoading: false }),
}));

afterEach(() => {
  cleanup();
  listRoles.mockReset();
});

function renderFor(unattendedRoleId: string | null) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const agent = { id: "agdef_intake", name: "Intake desk", unattendedRoleId } as AgentDefinitionRow;
  render(
    <QueryClientProvider client={client}>
      <AgentUnattendedRole agent={agent} canUpdate={false} />
    </QueryClientProvider>,
  );
}

describe("AgentUnattendedRole", () => {
  it("says an agent with no role keeps the agent ceiling", () => {
    renderFor(null);

    expect(screen.getByText("The agent ceiling alone")).toBeTruthy();
    expect(listRoles).not.toHaveBeenCalled();
  });

  it("names the role an agent runs as", async () => {
    listRoles.mockResolvedValue({
      results: [{ id: "rol_intake", name: "Intake clerk" }],
      count: 1,
      next: "",
      previous: "",
    });
    renderFor("rol_intake");

    expect(await screen.findByText("Intake clerk")).toBeTruthy();
  });
});
