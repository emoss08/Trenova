import { RequestTimeoutError } from "@trenova/shared/lib/request-errors";
import { afterEach, describe, expect, it, vi } from "vitest";
import { handleMutationError } from "../use-api-mutation";

const toastError = vi.hoisted(() => vi.fn());
vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: { error: toastError, success: vi.fn() },
}));

describe("a write that passed its deadline", () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it("is not called a failure, because the change may have landed", () => {
    handleMutationError({ error: new RequestTimeoutError() });

    expect(toastError).toHaveBeenCalledTimes(1);
    expect(toastError).toHaveBeenCalledWith(
      "This may not have saved",
      expect.objectContaining({
        description:
          "The server didn't answer in time. Check whether the change went through before trying again.",
      }),
    );
  });
});

describe("a failure two surfaces catch", () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it("is only told to the person once", () => {
    const error = new Error("Refused");

    handleMutationError({ error });
    handleMutationError({ error });

    expect(toastError).toHaveBeenCalledTimes(1);
  });
});
