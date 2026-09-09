import { beforeEach, describe, expect, it, vi } from "vitest";
import { resetUserPassword } from "../user-api";

const mocks = vi.hoisted(() => ({ post: vi.fn() }));

vi.mock("@trenova/shared/lib/api", () => ({
  api: { post: mocks.post },
}));

describe("resetUserPassword", () => {
  beforeEach(() => {
    mocks.post.mockClear();
    mocks.post.mockResolvedValue({ message: "sent" });
  });

  // Pins the path against the Go route, which is registered as
  // "/:userID/reset-password/" on the /users group. The slash is not cosmetic: a POST
  // to the unslashed path relies on Gin's 307 redirect being replayed with its body.
  it("posts to the route the server actually registers", async () => {
    await resetUserPassword("usr_123");

    expect(mocks.post).toHaveBeenCalledWith("/users/usr_123/reset-password/", {});
  });
});
