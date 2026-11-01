import { describe, expect, it } from "vitest";
import { GraphQLRequestError } from "../graphql";

describe("GraphQLRequestError.getUsageStats", () => {
  it("returns the usage a conflict refusal counted", () => {
    const usageStats = {
      definitionId: "cfd_1",
      totalValueCount: 12,
      resourceCount: 4,
      optionUsage: [{ value: "rush", label: "Rush", usageCount: 12 }],
    };
    const error = new GraphQLRequestError({
      kind: "graphql",
      message: "Cannot delete custom field definition",
      status: 200,
      graphQLErrors: [
        {
          message: "Cannot delete custom field definition",
          extensions: {
            code: "RESOURCE_IN_USE",
            type: "https://api.test/problems/resource-conflict",
            usageStats,
          },
        },
      ],
    });

    expect(error.getUsageStats()).toEqual(usageStats);
  });

  it("is undefined when the refusal carried no usage", () => {
    const error = new GraphQLRequestError({
      kind: "graphql",
      message: "Busy",
      status: 200,
      graphQLErrors: [{ message: "Busy", extensions: { code: "RESOURCE_IN_USE" } }],
    });

    expect(error.getUsageStats()).toBeUndefined();
  });
});
