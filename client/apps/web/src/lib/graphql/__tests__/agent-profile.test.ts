import {
  AddAgentOrganizationProfileGuidanceDocument,
  AddMyAgentProfileGuidanceDocument,
  AgentOrganizationProfileDocument,
  MyAgentProfileDocument,
  RegenerateAgentOrganizationProfileDocument,
  RegenerateMyAgentProfileDocument,
  RetireMyAgentProfileGuidanceDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  AGENT_PROFILE_KEY,
  addAgentProfileGuidance,
  agentProfileHistoryQueryKey,
  agentProfileQueryKey,
  fetchAgentProfile,
  regenerateAgentProfile,
  retireAgentProfileGuidance,
} from "../agent-profile";

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: vi.fn(),
}));

const requestGraphQLMock = vi.mocked(requestGraphQL);

const version = {
  __typename: "AgentProfileVersion",
  id: "aprf_1",
  subject: "Organization",
  status: "Current",
  trigger: "Scheduled",
  content: "Acme Freight runs a regional fleet.",
  facts: [{ __typename: "AgentProfileFact", key: "fleet.tractors", text: "About 45 tractors." }],
  guidanceIds: ["aprg_1"],
  model: "model",
  errorMessage: null,
  checkedAt: 1_790_000_000,
  createdAt: 1_789_990_000,
};

const guidance = {
  __typename: "AgentProfileGuidance",
  id: "aprg_1",
  subject: "Organization",
  content: "The Laredo lane is seasonal.",
  status: "Active",
  source: "Person",
  createdById: "usr_1",
  createdByName: "Avery Lane",
  threadId: null,
  retiredAt: null,
  createdAt: 1_789_000_000,
};

describe("agent profile", () => {
  beforeEach(() => {
    requestGraphQLMock.mockReset();
  });

  it("reads the organization's profile with its version and guidance unmasked", async () => {
    requestGraphQLMock.mockResolvedValueOnce({
      agentOrganizationProfile: {
        __typename: "AgentProfile",
        subject: "Organization",
        current: version,
        latestFailure: null,
        guidance: [guidance],
      },
    } as never);

    const profile = await fetchAgentProfile("organization");

    expect(requestGraphQLMock).toHaveBeenCalledWith(
      expect.objectContaining({
        document: AgentOrganizationProfileDocument,
        operationName: "AgentOrganizationProfile",
      }),
    );
    expect(profile.current?.content).toBe("Acme Freight runs a regional fleet.");
    expect(profile.current?.facts).toEqual([version.facts[0]]);
    expect(profile.latestFailure).toBeNull();
    expect(profile.guidance).toEqual([guidance]);
  });

  it("reads the person's own profile through the self-scoped query", async () => {
    requestGraphQLMock.mockResolvedValueOnce({
      myAgentProfile: {
        __typename: "AgentProfile",
        subject: "User",
        current: null,
        latestFailure: null,
        guidance: [],
      },
    } as never);

    const profile = await fetchAgentProfile("mine");

    expect(requestGraphQLMock).toHaveBeenCalledWith(
      expect.objectContaining({
        document: MyAgentProfileDocument,
        operationName: "MyAgentProfile",
      }),
    );
    expect(profile.current).toBeNull();
    expect(profile.guidance).toEqual([]);
  });

  it("sends guidance trimmed, to the mutation for whose profile it is", async () => {
    requestGraphQLMock.mockResolvedValue({
      addAgentOrganizationProfileGuidance: guidance,
      addMyAgentProfileGuidance: { ...guidance, subject: "User" },
    } as never);

    await addAgentProfileGuidance("organization", "  The Laredo lane is seasonal.  ");
    await addAgentProfileGuidance("mine", "I only handle the Midwest accounts.");

    expect(requestGraphQLMock).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({
        document: AddAgentOrganizationProfileGuidanceDocument,
        variables: { input: { content: "The Laredo lane is seasonal." } },
      }),
    );
    expect(requestGraphQLMock).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({
        document: AddMyAgentProfileGuidanceDocument,
        variables: { input: { content: "I only handle the Midwest accounts." } },
      }),
    );
  });

  it("retires and regenerates the person's own profile, never the organization's", async () => {
    requestGraphQLMock.mockResolvedValue({
      retireMyAgentProfileGuidance: { ...guidance, status: "Retired", retiredAt: 1_790_000_100 },
      regenerateMyAgentProfile: true,
    } as never);

    const retired = await retireAgentProfileGuidance("mine", "aprg_1");
    await regenerateAgentProfile("mine");

    expect(retired.status).toBe("Retired");
    expect(requestGraphQLMock).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({
        document: RetireMyAgentProfileGuidanceDocument,
        variables: { id: "aprg_1" },
      }),
    );
    expect(requestGraphQLMock).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({ document: RegenerateMyAgentProfileDocument }),
    );
    expect(requestGraphQLMock).not.toHaveBeenCalledWith(
      expect.objectContaining({ document: RegenerateAgentOrganizationProfileDocument }),
    );
  });

  it("keeps every profile query under the root its realtime resource refreshes", () => {
    expect(agentProfileQueryKey("organization")[0]).toBe(AGENT_PROFILE_KEY);
    expect(agentProfileQueryKey("mine")[0]).toBe(AGENT_PROFILE_KEY);
    expect(agentProfileHistoryQueryKey[0]).toBe(AGENT_PROFILE_KEY);
  });
});
