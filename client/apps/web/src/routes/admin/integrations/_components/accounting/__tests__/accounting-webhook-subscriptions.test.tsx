import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { AccountingWebhookSubscriptions } from "../accounting-webhook-subscriptions";
import { businessCentralVendor } from "../accounting-vendors";

const base = {
  configured: true,
  active: 0,
  pending: 0,
  failed: 0,
  nextExpiryAt: null,
  lastError: null,
};

describe("AccountingWebhookSubscriptions", () => {
  it("shows the active subscriptions and when they renew", () => {
    render(
      <AccountingWebhookSubscriptions
        vendor={businessCentralVendor}
        summary={{ ...base, active: 7, nextExpiryAt: 1_780_000_000 }}
      />,
    );

    expect(screen.getByText("7 active")).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("explains that changes are polled when there is no public address", () => {
    render(
      <AccountingWebhookSubscriptions
        vendor={businessCentralVendor}
        summary={{ ...base, configured: false, failed: 7 }}
      />,
    );

    expect(screen.getByText(/no public https address/)).toBeInTheDocument();
    expect(screen.queryByText(/could not be kept/)).not.toBeInTheDocument();
  });

  it("names why subscriptions could not be kept", () => {
    render(
      <AccountingWebhookSubscriptions
        vendor={businessCentralVendor}
        summary={{ ...base, active: 5, failed: 2, lastError: "The handshake timed out." }}
      />,
    );

    expect(screen.getByText(/2 of Trenova's change subscriptions/)).toBeInTheDocument();
    expect(screen.getByText(/The handshake timed out\./)).toBeInTheDocument();
  });
});
