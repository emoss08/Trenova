import { DeskServiceFailureCardDocument } from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { fetchDeskServiceFailure } from "../service-failure-card";

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: vi.fn(),
}));

const requestGraphQLMock = vi.mocked(requestGraphQL);

beforeEach(() => {
  requestGraphQLMock.mockReset();
});

describe("fetchDeskServiceFailure", () => {
  it("reads one failure by id, passing the signal on", async () => {
    requestGraphQLMock.mockResolvedValue({
      serviceFailure: { id: "sf_1", number: "SF-1" },
    } as never);
    const controller = new AbortController();

    const failure = await fetchDeskServiceFailure("sf_1", { signal: controller.signal });

    expect(failure).toEqual({ id: "sf_1", number: "SF-1" });
    expect(requestGraphQLMock).toHaveBeenCalledWith({
      document: DeskServiceFailureCardDocument,
      operationName: "DeskServiceFailureCard",
      variables: { id: "sf_1" },
      signal: controller.signal,
    });
  });

  it("says a failure the server does not return could not be found", async () => {
    requestGraphQLMock.mockResolvedValue({ serviceFailure: null } as never);

    await expect(fetchDeskServiceFailure("sf_999")).rejects.toThrow("Service failure not found");
  });
});
