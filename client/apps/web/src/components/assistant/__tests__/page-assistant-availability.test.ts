import { ApiRequestError } from "@trenova/shared/lib/api";
import { describe, expect, it } from "vitest";
import { pageAssistantAvailability } from "../page-assistant-availability";

/*
 * Opening a page conversation needs assistant:create before anything else
 * (the route's own check, answered 403), then access to the page's agent
 * (MayUseAgent, 403 naming the agent), then the agent turned on (422 saying
 * so). Each refusal is told apart, and the server's own words are kept.
 */
describe("pageAssistantAvailability", () => {
  it("waits for the person's permissions before deciding anything", () => {
    expect(
      pageAssistantAvailability({ permissionsLoading: true, canAsk: false, error: null }),
    ).toEqual({ state: "checking" });
  });

  it("is unavailable without assistant create, without asking the server", () => {
    expect(
      pageAssistantAvailability({ permissionsLoading: false, canAsk: false, error: null }),
    ).toEqual({ state: "no-permission" });
  });

  it("is ready when the conversation opened", () => {
    expect(
      pageAssistantAvailability({ permissionsLoading: false, canAsk: true, error: null }),
    ).toEqual({ state: "ready" });
  });

  it("says the person has no access to the agent, in the server's words", () => {
    const error = new ApiRequestError(403, {
      type: "authorization-error",
      title: "Forbidden",
      status: 403,
      detail:
        "You do not have access to Formula assistant. An administrator can give one of your roles access to it.",
    });

    expect(pageAssistantAvailability({ permissionsLoading: false, canAsk: true, error })).toEqual({
      state: "no-access",
      message:
        "You do not have access to Formula assistant. An administrator can give one of your roles access to it.",
    });
  });

  it("passes on why the agent cannot be used, such as it being turned off", () => {
    const error = new ApiRequestError(422, {
      type: "business-error",
      title: "Unprocessable",
      status: 422,
      detail:
        "Shipment import assistant is turned off. An administrator can turn it on in AI Control.",
    });

    expect(pageAssistantAvailability({ permissionsLoading: false, canAsk: true, error })).toEqual({
      state: "unavailable",
      message:
        "Shipment import assistant is turned off. An administrator can turn it on in AI Control.",
    });
  });

  it("says something even when the failure carries no words", () => {
    const result = pageAssistantAvailability({
      permissionsLoading: false,
      canAsk: true,
      error: new Error(""),
    });

    expect(result.state).toBe("unavailable");
    expect(result.state === "unavailable" && result.message).not.toBe("");
  });
});
