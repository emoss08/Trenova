import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { MemoryRouter } from "react-router";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type {
  AgentRunRow,
  AgentRunTranscript,
  AgentRunTranscriptMessage,
} from "@/lib/graphql/agent-activity-tables";
import { queries } from "@/lib/queries";
import { AgentRunPanel } from "../agent-run-sheet";
import { transcriptBlocks } from "../run-transcript";
import { RunTranscriptView } from "../run-transcript-view";

const mocks = vi.hoisted(() => ({ download: vi.fn() }));

vi.mock("@/services/agent-run", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/services/agent-run")>()),
  downloadAgentRunTranscript: mocks.download,
}));

afterEach(cleanup);

const RUN = "arun_01";

function message(fields: Partial<AgentRunTranscriptMessage>): AgentRunTranscriptMessage {
  return {
    role: "Assistant",
    kind: "Message",
    content: "",
    reasoning: "",
    toolCalls: [],
    toolCallId: "",
    toolName: "",
    toolFailed: false,
    toolVerdict: "",
    toolSummary: "",
    agentDefinitionId: "",
    delegateCallId: "",
    omitted: false,
    createdAt: 1_790_000_000,
    ...fields,
  };
}

function deniedRun(): AgentRunTranscript {
  return {
    omittedMessages: 0,
    omittedAt: 0,
    messages: [
      message({
        content: "Checking the driver first.",
        reasoning: "The move needs a driver.",
        toolCalls: [
          { id: "c1", name: "update_worker", arguments: { id: "wrk_1" } },
          { id: "c2", name: "get_shipment", arguments: { id: "shp_1" } },
        ],
      }),
      message({
        role: "Tool",
        toolCallId: "c1",
        toolName: "update_worker",
        toolFailed: true,
        toolVerdict: "denied",
        content: 'Tool "update_worker" is not permitted.',
      }),
      message({
        role: "Tool",
        toolCallId: "c2",
        toolName: "get_shipment",
        toolSummary: "S-1001",
        toolVerdict: "ran",
        content: "{}",
      }),
      message({ content: "I could not reassign the driver." }),
    ],
  };
}

function wrap(children: ReactNode, client = new QueryClient()) {
  return (
    <QueryClientProvider client={client}>
      <MemoryRouter>{children}</MemoryRouter>
    </QueryClientProvider>
  );
}

/**
 * A background run keeps what its model said and the tools it called, so
 * the run reads back the way a conversation does: each turn with its calls
 * folded under it, refusals in their own words, and the stretch the record
 * left out counted where it fell.
 */
describe("transcriptBlocks", () => {
  it("folds each call and its result under the turn that asked for it", () => {
    const blocks = transcriptBlocks(RUN, deniedRun());

    expect(blocks.map((block) => block.kind)).toEqual(["turn", "turn"]);
    const first = blocks[0];
    if (first.kind !== "turn") throw new Error("expected a turn");
    expect(first.entry.tools.map((tool) => tool.result?.toolVerdict)).toEqual(["denied", "ran"]);
    expect(first.clipped).toBe(false);
  });

  it("counts the left-out middle where it fell and never pairs across it", () => {
    const blocks = transcriptBlocks(RUN, {
      omittedMessages: 5,
      omittedAt: 1,
      messages: [
        message({
          content: "Started.",
          toolCalls: [{ id: "c1", name: "get_shipment", arguments: {} }],
        }),
        message({ role: "Tool", toolCallId: "c9", toolName: "get_worker", content: "{}" }),
        message({ content: "Finished." }),
      ],
    });

    expect(blocks.map((block) => block.kind)).toEqual(["turn", "gap", "turn"]);
    const [before, gap, after] = blocks;
    if (before.kind !== "turn" || gap.kind !== "gap" || after.kind !== "turn") {
      throw new Error("unexpected blocks");
    }
    expect(gap.count).toBe(5);
    expect(before.entry.tools[0].result).toBeNull();
    expect(after.entry.tools.map((tool) => tool.call.id)).toEqual(["c9"]);
    expect(after.entry.tools[0].orphan).toBe(true);
  });

  it("marks a turn part of which was too large to keep", () => {
    const blocks = transcriptBlocks(RUN, {
      omittedMessages: 0,
      omittedAt: 0,
      messages: [
        message({ toolCalls: [{ id: "c1", name: "run_report", arguments: null }] }),
        message({ role: "Tool", toolCallId: "c1", toolName: "run_report", omitted: true }),
      ],
    });

    expect(blocks).toHaveLength(1);
    expect(blocks[0].kind === "turn" && blocks[0].clipped).toBe(true);
  });

  it("leaves out a role this reader does not know", () => {
    const blocks = transcriptBlocks(RUN, {
      omittedMessages: 0,
      omittedAt: 0,
      messages: [message({ role: "System", content: "internal" }), message({ content: "Done." })],
    });

    expect(blocks).toHaveLength(1);
  });
});

describe("RunTranscriptView", () => {
  it("reads the run back with the conversation's pieces", () => {
    render(wrap(<RunTranscriptView runId={RUN} transcript={deniedRun()} />));

    expect(screen.getByText("Checking the driver first.")).toBeTruthy();
    expect(screen.getByText("I could not reassign the driver.")).toBeTruthy();
    expect(screen.getByText("Thought it through")).toBeTruthy();
    expect(screen.getByText("Not permitted").className).toContain("text-warning");
  });

  it("says where the record left messages out", () => {
    render(
      wrap(
        <RunTranscriptView
          runId={RUN}
          transcript={{ ...deniedRun(), omittedMessages: 3, omittedAt: 1 }}
        />,
      ),
    );

    expect(screen.getByRole("note").textContent).toBe(
      "3 messages from the middle of the run were left out to keep the record bounded",
    );
  });
});

describe("AgentRunPanel", () => {
  beforeEach(() => {
    mocks.download.mockReset();
  });

  const row = {
    id: RUN,
    agentType: "AssistantChat",
    agentDefinitionId: "",
    status: "Completed",
    trigger: "Schedule",
    summary: "I could not reassign the driver.",
    modelIdentifier: "test-model",
    errorMessage: "",
    traceUrl: null,
    handedBy: null,
    startedAt: 1_790_000_000,
    completedAt: 1_790_000_042,
    createdAt: 1_790_000_000,
  } as unknown as AgentRunRow;

  function client(transcript: AgentRunTranscript | null, run: AgentRunRow = row) {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false, staleTime: Infinity } },
    });
    queryClient.setQueryData(queries.agentRun.detail(run.id).queryKey, { run, transcript });
    return queryClient;
  }

  function renderPanel(queryClient: QueryClient, run: AgentRunRow = row) {
    return render(
      <NuqsTestingAdapter>
        {wrap(<AgentRunPanel open onOpenChange={() => {}} mode="edit" row={run} />, queryClient)}
      </NuqsTestingAdapter>,
    );
  }

  it("shows what the run did as soon as it opens", async () => {
    renderPanel(client(deniedRun()));

    const sheet = await screen.findByRole("complementary", {
      name: "I could not reassign the driver.",
    });
    expect(within(sheet).getByText("What it did")).toBeTruthy();
    expect(await within(sheet).findByText("Checking the driver first.")).toBeTruthy();
    expect(within(sheet).getByText("Not permitted")).toBeTruthy();
    expect(within(sheet).getByText("test-model")).toBeTruthy();
  });

  it("downloads the run's transcript from its sheet", async () => {
    const user = userEvent.setup();
    renderPanel(client(deniedRun()));

    await user.click(await screen.findByRole("button", { name: "Download transcript" }));

    expect(mocks.download).toHaveBeenCalledExactlyOnceWith(RUN);
  });

  it("says so when the run kept no transcript, and offers no download", async () => {
    renderPanel(client(null));

    expect(await screen.findByText(/This run kept no transcript/)).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Download transcript" })).toBeNull();
  });

  it("shows why a failed run failed and leads to the providers", async () => {
    const failed = {
      ...row,
      status: "Failed",
      errorMessage: "dial tcp 127.0.0.1:8000: connection refused",
    } as AgentRunRow;
    renderPanel(client(null, failed), failed);

    expect(await screen.findByText("dial tcp 127.0.0.1:8000: connection refused")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Check providers" })).toBeTruthy();
  });
});
