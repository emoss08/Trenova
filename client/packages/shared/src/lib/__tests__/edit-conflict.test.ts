import { describe, expect, it } from "vitest";
import { ApiRequestError } from "../api";
import { editConflictOf } from "../edit-conflict";
import { GraphQLRequestError } from "../graphql";

const conflict = {
  version: 8,
  updatedById: "usr_1",
  updatedByName: "Sarah Alvarez",
  updatedAt: 1_800_000_000,
  changes: [{ field: "instructions", label: "Instructions" }],
};

describe("editConflictOf", () => {
  it("reads the conflict a REST save came back with", () => {
    const error = new ApiRequestError(409, {
      type: "resource-conflict",
      title: "Conflict",
      status: 409,
      conflict,
    });

    expect(editConflictOf(error)).toEqual(conflict);
  });

  it("is null for anything else", () => {
    expect(editConflictOf(new Error("boom"))).toBeNull();
    expect(
      editConflictOf(new ApiRequestError(409, { type: "resource-conflict", title: "x", status: 409 })),
    ).toBeNull();
  });

  it("reads the conflict a GraphQL save came back with", () => {
    const error = new GraphQLRequestError({
      kind: "graphql",
      message: "Conflict",
      status: 200,
      graphQLErrors: [{ message: "Conflict", extensions: { code: "VERSION_MISMATCH", conflict } }],
    });

    expect(editConflictOf(error)).toEqual(conflict);
  });
});
