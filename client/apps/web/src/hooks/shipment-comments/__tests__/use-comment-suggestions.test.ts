import { beforeEach, describe, expect, it, vi } from "vitest";

const fetchGraphQLSelectOptions = vi.hoisted(() => vi.fn());

vi.mock("@/lib/graphql/select-options", () => ({ fetchGraphQLSelectOptions }));

const { fetchMentionCandidates } = await import("../use-comment-suggestions");

describe("fetchMentionCandidates", () => {
  beforeEach(() => {
    fetchGraphQLSelectOptions.mockReset();
  });

  it("searches the organization's users and shows each by name, with an email when it adds one", async () => {
    fetchGraphQLSelectOptions.mockResolvedValue({
      count: 2,
      next: "2",
      results: [
        { id: "usr_1", label: "Dana Dispatcher", description: "dana@example.com", meta: null },
        { id: "usr_2", label: "sam@example.com", description: "sam@example.com", meta: null },
      ],
    });
    const controller = new AbortController();

    const page = await fetchMentionCandidates("da", 2, { signal: controller.signal });

    expect(fetchGraphQLSelectOptions).toHaveBeenCalledWith(
      { resource: "USER", query: "da", page: 2, initialLimit: 10 },
      { signal: controller.signal },
    );
    expect(page).toEqual({
      items: [
        { id: "usr_1", label: "Dana Dispatcher", description: "dana@example.com" },
        { id: "usr_2", label: "sam@example.com", description: null },
      ],
      hasMore: true,
    });
  });
});
