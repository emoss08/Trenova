import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, within } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { emailMessageListSchema } from "@trenova/shared/types/email";
import { EmailLogsPage } from "../page";

const listLogs = vi.hoisted(() => vi.fn());

vi.mock("@/services/email", () => ({
  emailService: { listLogs },
}));
vi.mock("@/components/navigation/sidebar-layout", () => ({
  PageLayout: ({ children }: { children: ReactNode }) => <main>{children}</main>,
}));

// GET /email-logs/: what the server stores per message (internal/modules/workspace/domain/email).
function message(overrides: Record<string, unknown>) {
  return {
    id: "emlmsg_1",
    profileId: "",
    purpose: "Billing",
    kind: "CustomerFacing",
    sender: "Organization",
    provider: "Resend",
    idempotencyKey: "key",
    status: "Delivered",
    attempts: 1,
    fromEmail: "billing@acme.test",
    fromName: "Acme Billing",
    toRecipients: ["ap@customer.test"],
    subject: "Invoice INV-1001",
    createdAt: 1_790_000_000,
    updatedAt: 1_790_000_000,
    ...overrides,
  };
}

afterEach(() => {
  cleanup();
  listLogs.mockReset();
});

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <EmailLogsPage />
    </QueryClientProvider>,
  );
}

describe("EmailLogsPage", () => {
  it("names who sent each message: Trenova, or the organization's email profile", async () => {
    listLogs.mockResolvedValue(
      emailMessageListSchema.parse({
        count: 3,
        next: "",
        prev: "",
        results: [
          message({
            id: "emlmsg_1",
            subject: "Reset your password",
            kind: "System",
            sender: "Trenova",
            fromEmail: "no-reply@trenova.app",
            fromName: "Trenova",
          }),
          message({
            id: "emlmsg_2",
            subject: "Invoice INV-1001",
            profileId: "emlprof_1",
            profile: {
              id: "emlprof_1",
              name: "Accounts receivable",
              senderName: "Acme Billing",
              senderEmail: "billing@acme.test",
            },
          }),
          message({ id: "emlmsg_3", subject: "Invoice INV-1002", fromName: "Acme Dispatch" }),
        ],
      }),
    );

    renderPage();

    const row = async (subject: string) =>
      (await screen.findByText(subject)).closest("tr") as HTMLElement;
    expect(within(await row("Reset your password")).getByText("Sent by Trenova")).toBeVisible();
    expect(within(await row("Invoice INV-1001")).getByText("Accounts receivable")).toBeVisible();
    expect(within(await row("Invoice INV-1002")).getByText("Acme Dispatch")).toBeVisible();
    expect(screen.getByRole("columnheader", { name: "Sender" })).toBeInTheDocument();
  });
});
