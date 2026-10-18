import type { ComposerPayload } from "@/components/assistant/composer-types";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { assistant } from "@/lib/queries/assistant";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { translate } from "@trenova/shared/i18n/runtime";
import { useState } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DeskComposer } from "../desk-composer";

/**
 * GET /assistant/agents/:agentId/browser/ as the server answers it: whether
 * the person can send that agent to the browser.
 */
const server = vi.hoisted(() => ({
  browserAgents: new Set<string>(),
  asked: [] as string[],
}));

vi.mock("@trenova/shared/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@trenova/shared/lib/api")>();
  return {
    ...actual,
    api: {
      ...actual.api,
      get: vi.fn(async (url: string) => {
        server.asked.push(url);
        const agentId = /^\/assistant\/agents\/([^/]+)\/browser\/$/.exec(url)?.[1];
        if (agentId === undefined) {
          throw new Error(`unexpected request ${url}`);
        }
        return { available: server.browserAgents.has(decodeURIComponent(agentId)) };
      }),
    },
  };
});

afterEach(() => {
  cleanup();
  server.browserAgents.clear();
  server.asked.length = 0;
});

function agentChoice(id: string): AgentChoice {
  return { id, name: "Dispatch", starters: [] } as unknown as AgentChoice;
}

function Harness({
  agentId,
  initial,
  busy = false,
  onSend,
  onSteer,
}: {
  agentId: string;
  initial: string;
  busy?: boolean;
  onSend: (content: string, payload: ComposerPayload) => void;
  onSteer?: (content: string, payload: ComposerPayload, mode: "steer" | "queue") => void;
}) {
  const [value, setValue] = useState(initial);

  return (
    <DeskComposer
      value={value}
      onChange={setValue}
      onSend={onSend}
      onSteer={onSteer}
      agent={agentChoice(agentId)}
      busy={busy}
    />
  );
}

function renderComposer(props: Parameters<typeof Harness>[0]) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <Harness {...props} />
      </MemoryRouter>
    </QueryClientProvider>,
  );

  return {
    box: screen.getByRole("textbox", { name: translate("Message {0}", "Dispatch") }),
    queryClient,
  };
}

describe("DeskComposer /browser", () => {
  it("is offered when the server says the agent can work in the browser", async () => {
    server.browserAgents.add("agdef_browser");
    renderComposer({ agentId: "agdef_browser", initial: "/bro", onSend: vi.fn() });

    expect(await screen.findByRole("option", { name: /\/browser/ })).toBeInTheDocument();
    expect(server.asked).toEqual(["/assistant/agents/agdef_browser/browser/"]);
  });

  it("is hidden when the server says the agent cannot", async () => {
    const { queryClient } = renderComposer({
      agentId: "agdef_plain",
      initial: "/bro",
      onSend: vi.fn(),
    });

    await waitFor(() =>
      expect(queryClient.getQueryData(assistant.browser("agdef_plain").queryKey)).toEqual({
        available: false,
      }),
    );
    expect(
      screen.getByText("No command called “/bro”. Press Esc to send it as text."),
    ).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: /\/browser/ })).toBeNull();
    expect(server.asked).toEqual(["/assistant/agents/agdef_plain/browser/"]);
  });

  it("sends the command with the person's words once the task is filled", async () => {
    server.browserAgents.add("agdef_browser");
    const onSend = vi.fn();
    const { box } = renderComposer({
      agentId: "agdef_browser",
      initial: "/browser check the 9am slot at Acme DC",
      onSend,
    });

    expect(
      await screen.findByText("Do it in the browser, on one of your portal sites"),
    ).toBeInTheDocument();
    fireEvent.keyDown(box, { key: "Enter" });

    expect(onSend).toHaveBeenCalledWith("/browser check the 9am slot at Acme DC", {
      attachments: [],
      mentions: [],
    });
  });

  it("waits for the reply under way rather than steering it", async () => {
    server.browserAgents.add("agdef_browser");
    const onSteer = vi.fn();
    const { box } = renderComposer({
      agentId: "agdef_browser",
      initial: "/browser check the 9am slot at Acme DC",
      busy: true,
      onSend: vi.fn(),
      onSteer,
    });

    await screen.findByText("Do it in the browser, on one of your portal sites");
    fireEvent.keyDown(box, { key: "Enter" });

    expect(onSteer).toHaveBeenCalledWith(
      "/browser check the 9am slot at Acme DC",
      { attachments: [], mentions: [] },
      "queue",
    );
  });
});
