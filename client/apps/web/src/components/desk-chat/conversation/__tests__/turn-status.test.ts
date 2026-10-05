import type { TurnState } from "@/components/assistant/turn-stream";
import { describe, expect, it } from "vitest";
import { composerStatus } from "../turn-status";

const t = (text: string | null | undefined) => text ?? "";

function working(name: string): TurnState {
  return {
    status: "streaming",
    segments: [
      { kind: "tool", callId: "call_1", name, arguments: {}, status: "running", content: "" },
    ],
  } as unknown as TurnState;
}

describe("composer status while the agent works", () => {
  it("carries the memory mark while a memory is read or kept", () => {
    expect(composerStatus(working("recall_memory"), t)?.pose).toBe("memory");
    expect(composerStatus(working("remember"), t)?.pose).toBe("memory");
  });

  it("is ordinary work for any other step", () => {
    expect(composerStatus(working("get_shipment"), t)?.pose).toBe("work");
  });
});
