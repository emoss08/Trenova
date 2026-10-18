import type { AssistantArtifact } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { portalTaskFrom } from "../../../artifact-payloads";
import { portalTaskResultView } from "../portal-task";

function artifactOf(payload: Record<string, unknown>): AssistantArtifact {
  return {
    id: "aart_1",
    threadId: "athr_1",
    kind: "portal_task",
    status: "Ready",
    title: "Which dock appointments are open on October 20?",
    payload,
  } as unknown as AssistantArtifact;
}

describe("portalTaskFrom", () => {
  it("reads what read_portal published", () => {
    expect(
      portalTaskFrom(
        artifactOf({
          site: "Acme dock portal",
          question: "Which dock appointments are open on October 20?",
          answer: "Open slots: 8:00 AM, 1:30 PM.",
          pagesRead: ["https://portal.acme.com/appointments"],
          portalRunId: "aprn_1",
          retrievedOn: "2026-10-17",
          mode: "Read",
        }),
      ),
    ).toEqual({
      runId: "aprn_1",
      mode: "Read",
      site: "Acme dock portal",
      question: "Which dock appointments are open on October 20?",
      answer: "Open slots: 8:00 AM, 1:30 PM.",
      pagesRead: ["https://portal.acme.com/appointments"],
      awaitingSignInCode: false,
      portalSaid: "",
      retrievedOn: "2026-10-17",
    });
  });

  it("reads a read stopped on a sign-in code", () => {
    const task = portalTaskFrom(
      artifactOf({
        site: "Acme dock portal",
        question: "Which slots are open?",
        awaitingSignInCode: true,
        portalSaid: "SIGN-IN CODE NEEDED: a code was texted to the phone ending 42.",
        portalRunId: "aprn_1",
        mode: "Read",
      }),
    );

    expect(task.awaitingSignInCode).toBe(true);
    expect(task.portalSaid).toBe("SIGN-IN CODE NEEDED: a code was texted to the phone ending 42.");
    expect(task.answer).toBe("");
  });

  it("reads an approved action, and anything it does not promise as absent", () => {
    expect(
      portalTaskFrom(
        artifactOf({
          portalRunId: "aprn_2",
          mode: "Act",
          site: "Acme dock portal",
          written: "started",
          pagesRead: "not a list",
        }),
      ),
    ).toEqual({
      runId: "aprn_2",
      mode: "Act",
      site: "Acme dock portal",
      question: "",
      answer: "",
      pagesRead: [],
      awaitingSignInCode: false,
      portalSaid: "",
      retrievedOn: "",
    });
  });
});

describe("portalTaskResultView", () => {
  it("shows what the agent was told, with no way to act on the run", () => {
    expect(
      portalTaskResultView({
        runId: "aprn_1",
        mode: "Read",
        site: "Acme dock portal",
        question: "Which slots are open?",
        answer: "Open slots: 8:00 AM.",
        pagesRead: ["https://portal.acme.com/appointments"],
        awaitingSignInCode: false,
        portalSaid: "",
        retrievedOn: "2026-10-17",
      }),
    ).toEqual({
      runId: "aprn_1",
      status: null,
      answer: "Open slots: 8:00 AM.",
      failure: "",
      visitedUrls: ["https://portal.acme.com/appointments"],
      offSiteUrls: [],
      hasRecording: false,
      codePrompt: "",
      codeExpiresAt: null,
      codeAttempts: 0,
    });
  });
});
