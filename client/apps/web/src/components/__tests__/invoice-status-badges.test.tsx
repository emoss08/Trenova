import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import {
  InvoiceDisputeCaseStatusBadge,
  InvoiceEdiSendStatusBadge,
  InvoiceStatusBadge,
  PlainInvoiceDisputeBadge,
  PlainInvoiceScopeBadge,
  PlainInvoiceStatusBadge,
} from "@trenova/shared/components/status-badge";

afterEach(cleanup);

describe("invoice status badges", () => {
  it("names a voided invoice in both badge styles", () => {
    render(
      <>
        <PlainInvoiceStatusBadge status="Voided" />
        <InvoiceStatusBadge status="Voided" />
      </>,
    );
    expect(screen.getAllByText("Voided")).toHaveLength(2);
  });

  it("names a standalone memo's scope", () => {
    render(<PlainInvoiceScopeBadge scope="Memo" />);
    expect(screen.getByText("Memo")).toBeInTheDocument();
  });

  it("reads a dispute case status and the invoice's disputed flag", () => {
    render(
      <>
        <InvoiceDisputeCaseStatusBadge status="Open" />
        <InvoiceDisputeCaseStatusBadge status="Withdrawn" />
        <PlainInvoiceDisputeBadge disputeStatus="Disputed" />
        <PlainInvoiceDisputeBadge disputeStatus="None" />
      </>,
    );
    expect(screen.getByText("Open")).toBeInTheDocument();
    expect(screen.getByText("Withdrawn")).toBeInTheDocument();
    expect(screen.getAllByText("Disputed")).toHaveLength(1);
  });

  it("reads every EDI send status", () => {
    render(
      <>
        <InvoiceEdiSendStatusBadge status="NotConfigured" />
        <InvoiceEdiSendStatusBadge status="DeadLettered" />
        <InvoiceEdiSendStatusBadge status="Sent" />
      </>,
    );
    expect(screen.getByText("Not configured")).toBeInTheDocument();
    expect(screen.getByText("Dead-lettered")).toBeInTheDocument();
    expect(screen.getByText("Sent")).toBeInTheDocument();
  });
});
