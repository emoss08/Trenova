import { DeskTrailerCardDocument } from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { fetchDeskTrailer } from "../trailer-card";

vi.mock("@trenova/shared/lib/graphql", () => ({ requestGraphQL: vi.fn() }));

const requestGraphQLMock = vi.mocked(requestGraphQL);

beforeEach(() => {
  requestGraphQLMock.mockReset();
});

describe("fetchDeskTrailer", () => {
  it("reads the trailer by id and passes the signal on", async () => {
    requestGraphQLMock.mockResolvedValue({ trailer: { id: "trl_1", code: "TL-501" } } as never);
    const controller = new AbortController();

    await expect(fetchDeskTrailer("trl_1", { signal: controller.signal })).resolves.toEqual({
      id: "trl_1",
      code: "TL-501",
    });
    const call = requestGraphQLMock.mock.calls.at(-1)?.[0] as unknown as {
      document: unknown;
      variables: unknown;
      signal?: AbortSignal;
    };
    expect(call.document).toBe(DeskTrailerCardDocument);
    expect(call.variables).toEqual({ id: "trl_1" });
    expect(call.signal).toBe(controller.signal);
  });

  it("treats a trailer the server does not return as an error", async () => {
    requestGraphQLMock.mockResolvedValue({ trailer: null } as never);
    await expect(fetchDeskTrailer("trl_gone")).rejects.toThrow("Trailer not found");
  });
});
