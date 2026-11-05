import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { markExplained } from "@trenova/shared/lib/explained-errors";
import { reportFailure, setFailureReporter } from "@trenova/shared/lib/failure-reporter";
import { RequestTimeoutError, ResponseContractError } from "@trenova/shared/lib/request-errors";

const reporter = vi.fn();

function failed(status: number): ApiRequestError {
  return new ApiRequestError(status, { type: "about:blank", title: "Failed", status });
}

describe("reportFailure", () => {
  beforeEach(() => {
    setFailureReporter(reporter);
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("reports a server fault with what support searches by", () => {
    reportFailure(failed(500), { source: "query", operation: "shipment" });

    expect(reporter).toHaveBeenCalledWith(
      expect.objectContaining({ kind: "server", status: 500 }),
      { source: "query", operation: "shipment" },
    );
  });

  it("reports a response this build cannot read", () => {
    reportFailure(new ResponseContractError("Shipment"), { source: "query" });

    expect(reporter).toHaveBeenCalledWith(
      expect.objectContaining({ kind: "contract" }),
      expect.anything(),
    );
  });

  it("reports one failure once, however many surfaces saw it", () => {
    const error = new Error("broken");

    reportFailure(error, { source: "query" });
    reportFailure(error, { source: "boundary" });

    expect(reporter).toHaveBeenCalledTimes(1);
  });

  it("stays quiet about what is not Trenova's fault", () => {
    reportFailure(failed(403), { source: "query" });
    reportFailure(failed(404), { source: "query" });
    reportFailure(new TypeError("Failed to fetch"), { source: "query" });
    reportFailure(new RequestTimeoutError(), { source: "query" });

    expect(reporter).not.toHaveBeenCalled();
  });

  it("stays quiet about a cancellation and what a dialog already explained", () => {
    const explained = new Error("Plan limit");
    markExplained(explained);

    reportFailure(new DOMException("Aborted", "AbortError"), { source: "query" });
    reportFailure(explained, { source: "mutation" });

    expect(reporter).not.toHaveBeenCalled();
  });
});
