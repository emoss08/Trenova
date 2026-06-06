import { beforeEach, describe, expect, it, vi } from "vitest";
import { listShipmentCommentsGraphQL } from "@/lib/graphql/shipment";
import type { ShipmentComment } from "@/types/shipment-comment";
import { ShipmentCommentService } from "./shipment-comment";

vi.mock("@/lib/graphql/shipment", () => ({
  createShipmentCommentGraphQL: vi.fn(),
  deleteShipmentCommentGraphQL: vi.fn(),
  getShipmentCommentCountGraphQL: vi.fn(),
  listShipmentCommentsGraphQL: vi.fn(),
  updateShipmentCommentGraphQL: vi.fn(),
}));

const listShipmentCommentsGraphQLMock = vi.mocked(listShipmentCommentsGraphQL);

describe("ShipmentCommentService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("accepts GraphQL-normalized pagination fields when listing comments", async () => {
    const comment: ShipmentComment = {
      id: "comment_1",
      shipmentId: "shp_1",
      userId: null,
      comment: "Follow up with dispatch.",
      type: "Internal",
      visibility: "Operations",
      priority: "Normal",
      source: "User",
      metadata: {},
      editedAt: null,
      version: 1,
      createdAt: 1_800_000_000,
      updatedAt: 1_800_000_100,
      mentionedUserIds: [],
    };

    listShipmentCommentsGraphQLMock.mockResolvedValueOnce({
      results: [comment],
      count: 1,
      next: null,
      prev: null,
      pageInfo: {
        mode: "cursor",
        hasNextPage: false,
        endCursor: null,
        totalCount: 1,
      },
    });

    const response = await new ShipmentCommentService().list("shp_1", {
      limit: 20,
      offset: 0,
    });

    expect(listShipmentCommentsGraphQLMock).toHaveBeenCalledWith({
      shipmentId: "shp_1",
      limit: 20,
      offset: 0,
    });
    expect(response).toEqual({
      results: [comment],
      count: 1,
      next: null,
      prev: null,
      pageInfo: {
        mode: "cursor",
        hasNextPage: false,
        endCursor: null,
        totalCount: 1,
      },
    });
  });
});
