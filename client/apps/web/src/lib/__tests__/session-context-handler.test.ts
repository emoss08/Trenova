import { useSessionContextStore } from "@/stores/session-context-store";
import { beforeEach, describe, expect, it, vi } from "vitest";

const leaveEndedSession = vi.hoisted(() => vi.fn(async () => undefined));

vi.mock("@/lib/session-exit", () => ({ leaveEndedSession }));

const { handleSessionContextStatus } = await import("@/lib/session-context-handler");

describe("handleSessionContextStatus", () => {
  beforeEach(() => {
    leaveEndedSession.mockClear();
    useSessionContextStore.setState({ organizationChanged: false, keptOpen: false });
  });

  it("prompts for a reload when another tab moved the session to another organization", () => {
    handleSessionContextStatus("organization-changed");

    expect(useSessionContextStore.getState().organizationChanged).toBe(true);
    expect(leaveEndedSession).not.toHaveBeenCalled();
  });

  it("withdraws the prompt and the bar when the session comes back to this tab's organization", () => {
    handleSessionContextStatus("organization-changed");
    useSessionContextStore.getState().keepPageOpen();
    handleSessionContextStatus("current");

    expect(useSessionContextStore.getState()).toMatchObject({
      organizationChanged: false,
      keptOpen: false,
    });
  });

  it("does not bring the prompt back once the person chose to keep the page open", () => {
    handleSessionContextStatus("organization-changed");
    useSessionContextStore.getState().keepPageOpen();
    handleSessionContextStatus("organization-changed");

    expect(useSessionContextStore.getState()).toMatchObject({
      organizationChanged: true,
      keptOpen: true,
    });
  });

  it("leaves for the new person's session at once when someone else signs in", () => {
    handleSessionContextStatus("user-changed");

    expect(leaveEndedSession).toHaveBeenCalledWith({ destination: "/", pastUnsavedWork: true });
    expect(useSessionContextStore.getState().organizationChanged).toBe(false);
  });

  it("leaves for the sign-in page, saying why, when the person signs out in another tab", () => {
    handleSessionContextStatus("signed-out");

    expect(leaveEndedSession).toHaveBeenCalledWith({
      destination: "/login?signed_out=elsewhere",
      pastUnsavedWork: true,
    });
  });
});
