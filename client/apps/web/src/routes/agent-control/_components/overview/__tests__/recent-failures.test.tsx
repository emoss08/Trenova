import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { RecentFailures, type UsageFailure } from "../recent-failures";

afterEach(cleanup);

function failure(overrides: Partial<UsageFailure> = {}): UsageFailure {
  return {
    providerId: "aiprv_1",
    providerName: "Gemini",
    model: "gemini-3.8-flash",
    task: "assistant_chat",
    errorClass: "provider_error",
    message: "provider request failed (status 400): Function call is missing a thought_signature",
    at: 1_790_000_000,
    ...overrides,
  };
}

/**
 * A failure count with no reason beside it sends an administrator to the
 * server log. The panel puts the provider's own words on the overview, so
 * "3 failed" reads as "Gemini refuses the request because..." without a
 * terminal.
 */
describe("RecentFailures", () => {
  it("names the provider, the model and the provider's message", () => {
    render(<RecentFailures failures={[failure()]} />);

    expect(screen.getByText("Gemini")).toBeInTheDocument();
    expect(screen.getByText("gemini-3.8-flash")).toBeInTheDocument();
    expect(screen.getByText(/missing a thought_signature/)).toBeInTheDocument();
  });

  it("says the kind of failure in words, not the class token", () => {
    render(
      <RecentFailures
        failures={[
          failure({ errorClass: "provider_unavailable" }),
          failure({ at: 1_789_999_000, errorClass: "timeout", message: "" }),
        ]}
      />,
    );

    expect(screen.getByText("Provider unavailable")).toBeInTheDocument();
    expect(screen.getByText("Timed out")).toBeInTheDocument();
    expect(screen.queryByText("provider_unavailable")).not.toBeInTheDocument();
  });

  it("renders nothing when there is nothing to report", () => {
    const { container } = render(<RecentFailures failures={[]} />);

    expect(container).toBeEmptyDOMElement();
  });
});
