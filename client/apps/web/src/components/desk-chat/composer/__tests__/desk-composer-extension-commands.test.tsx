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
 * GET /assistant/agents/:agentId/commands/ as the server answers it: the
 * extension commands the person can send that agent.
 */
const server = vi.hoisted(() => ({
  commands: new Map<string, string[]>(),
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
        const agentId = /^\/assistant\/agents\/([^/]+)\/commands\/$/.exec(url)?.[1];
        if (agentId === undefined) {
          throw new Error(`unexpected request ${url}`);
        }
        return { commands: server.commands.get(decodeURIComponent(agentId)) ?? [] };
      }),
    },
  };
});

afterEach(() => {
  cleanup();
  server.commands.clear();
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

const CASES = [
  {
    name: "browser",
    prefix: "/bro",
    ask: "/browser check the 9am slot at Acme DC",
    description: "Do it in the browser, on one of your portal sites",
  },
  {
    name: "search",
    prefix: "/sea",
    ask: "/search FMCSA hours of service changes this year",
    description: "Search the web and answer from what it finds",
  },
] as const;

describe.each(CASES)("DeskComposer /$name", ({ name, prefix, ask, description }) => {
  const option = new RegExp(`/${name}`);

  it("is offered when the server says the agent can honour it", async () => {
    server.commands.set("agdef_ready", [name]);
    renderComposer({ agentId: "agdef_ready", initial: prefix, onSend: vi.fn() });

    expect(await screen.findByRole("option", { name: option })).toBeInTheDocument();
    expect(server.asked).toEqual(["/assistant/agents/agdef_ready/commands/"]);
  });

  it("is hidden when the server says the agent cannot", async () => {
    const { queryClient } = renderComposer({
      agentId: "agdef_plain",
      initial: prefix,
      onSend: vi.fn(),
    });

    await waitFor(() =>
      expect(queryClient.getQueryData(assistant.commands("agdef_plain").queryKey)).toEqual({
        commands: [],
      }),
    );
    expect(
      screen.getByText(`No command called “${prefix}”. Press Esc to send it as text.`),
    ).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: option })).toBeNull();
    expect(server.asked).toEqual(["/assistant/agents/agdef_plain/commands/"]);
  });

  it("sends the command with the person's words once the slot is filled", async () => {
    server.commands.set("agdef_ready", [name]);
    const onSend = vi.fn();
    const { box } = renderComposer({ agentId: "agdef_ready", initial: ask, onSend });

    expect(await screen.findByText(description)).toBeInTheDocument();
    fireEvent.keyDown(box, { key: "Enter" });

    expect(onSend).toHaveBeenCalledWith(ask, { attachments: [], mentions: [] });
  });

  it("waits for the reply under way rather than steering it", async () => {
    server.commands.set("agdef_ready", [name]);
    const onSteer = vi.fn();
    const { box } = renderComposer({
      agentId: "agdef_ready",
      initial: ask,
      busy: true,
      onSend: vi.fn(),
      onSteer,
    });

    await screen.findByText(description);
    fireEvent.keyDown(box, { key: "Enter" });

    expect(onSteer).toHaveBeenCalledWith(ask, { attachments: [], mentions: [] }, "queue");
  });
});

describe("DeskComposer extension commands", () => {
  it("offers only the commands the server named, browser before search", async () => {
    server.commands.set("agdef_both", ["search", "browser"]);
    renderComposer({ agentId: "agdef_both", initial: "/", onSend: vi.fn() });

    await screen.findByRole("option", { name: /\/search/ });
    const labels = screen
      .getAllByRole("option")
      .map((row) => /\/(browser|search)/.exec(row.textContent ?? "")?.[1])
      .filter((label) => label !== undefined);
    expect(labels).toEqual(["browser", "search"]);
  });

  it("offers /search alone when web research is the only extension on", async () => {
    server.commands.set("agdef_web", ["search"]);
    renderComposer({ agentId: "agdef_web", initial: "/", onSend: vi.fn() });

    expect(await screen.findByRole("option", { name: /\/search/ })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: /\/browser/ })).toBeNull();
  });
});
