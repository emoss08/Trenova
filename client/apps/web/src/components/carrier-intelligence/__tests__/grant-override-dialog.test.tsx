import type { CarrierIntelFinding } from "@/lib/graphql/carrier-intelligence";
import {
  MAX_OVERRIDE_SECONDS,
  SECONDS_PER_DAY,
  createOverrideFormSchema,
} from "@/lib/carrier-intelligence";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { GrantOverrideDialog } from "../grant-override-dialog";

const mocks = vi.hoisted(() => ({
  grantCarrierIntelOverride: vi.fn(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}));

vi.mock("@/lib/graphql/carrier-intelligence", () => ({
  grantCarrierIntelOverride: mocks.grantCarrierIntelOverride,
}));
vi.mock("sonner", () => ({ toast: { success: mocks.toastSuccess, error: mocks.toastError } }));

afterEach(() => {
  vi.clearAllMocks();
});

const FINDING: CarrierIntelFinding = {
  code: "insurance.bipd_below_minimum",
  category: "Insurance",
  action: "Block",
  severity: "High",
  message: "BIPD coverage on file is below the required minimum",
  unverifiable: false,
  unconfirmed: false,
  overridden: false,
  overrideId: null,
  overrideExpiresAt: null,
};

function renderDialog() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const onOpenChange = vi.fn();
  const onGranted = vi.fn();
  render(
    <QueryClientProvider client={client}>
      <GrantOverrideDialog
        carrierId="car_1"
        finding={FINDING}
        ruleLabel="BIPD minimum"
        open
        onOpenChange={onOpenChange}
        onGranted={onGranted}
      />
    </QueryClientProvider>,
  );
  return { onOpenChange, onGranted };
}

describe("createOverrideFormSchema", () => {
  const now = 1_800_000_000;
  const schema = createOverrideFormSchema(now);

  it("requires a reason", () => {
    const result = schema.safeParse({ reason: "   ", expiresAt: now + SECONDS_PER_DAY });
    expect(result.success).toBe(false);
    expect(result.error?.issues.map((issue) => issue.message)).toContain(
      "A reason is required to override a finding",
    );
  });

  it("requires an expiry", () => {
    const result = schema.safeParse({ reason: "Certificate received", expiresAt: null });
    expect(result.success).toBe(false);
    expect(result.error?.issues.map((issue) => issue.message)).toContain(
      "Choose when the override expires",
    );
  });

  it("rejects an expiry in the past", () => {
    const result = schema.safeParse({ reason: "Certificate received", expiresAt: now - 60 });
    expect(result.success).toBe(false);
    expect(result.error?.issues.map((issue) => issue.message)).toContain(
      "An override must expire in the future",
    );
  });

  it("accepts exactly 90 days and rejects anything longer", () => {
    expect(
      schema.safeParse({ reason: "Certificate received", expiresAt: now + MAX_OVERRIDE_SECONDS })
        .success,
    ).toBe(true);

    const result = schema.safeParse({
      reason: "Certificate received",
      expiresAt: now + MAX_OVERRIDE_SECONDS + 1,
    });
    expect(result.success).toBe(false);
    expect(result.error?.issues.map((issue) => issue.message)).toContain(
      "An override cannot last longer than 90 days",
    );
  });

  it("rejects a reason over 2000 characters", () => {
    const result = schema.safeParse({
      reason: "x".repeat(2001),
      expiresAt: now + SECONDS_PER_DAY,
    });
    expect(result.success).toBe(false);
  });
});

describe("GrantOverrideDialog", () => {
  it("refuses to grant without a reason", async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByRole("button", { name: "Grant override" }));

    expect(
      await screen.findByText("A reason is required to override a finding"),
    ).toBeInTheDocument();
    expect(mocks.grantCarrierIntelOverride).not.toHaveBeenCalled();
  });

  it("grants a time-boxed override for the finding's rule", async () => {
    const user = userEvent.setup();
    const expiresAt = Math.floor(Date.now() / 1000) + 30 * SECONDS_PER_DAY;
    mocks.grantCarrierIntelOverride.mockResolvedValue({
      id: "cio_1",
      carrierId: "car_1",
      ruleCode: FINDING.code,
      reason: "Renewal certificate on file",
      grantedById: "usr_1",
      grantedAt: Math.floor(Date.now() / 1000),
      expiresAt,
      revokedById: null,
      revokedAt: null,
      revokeReason: null,
      active: true,
      version: 1,
      createdAt: Math.floor(Date.now() / 1000),
    });
    const { onOpenChange, onGranted } = renderDialog();

    await user.type(screen.getByLabelText(/Reason/), "  Renewal certificate on file  ");
    await user.click(screen.getByRole("button", { name: "30 days" }));
    await user.click(screen.getByRole("button", { name: "Grant override" }));

    await waitFor(() => expect(mocks.grantCarrierIntelOverride).toHaveBeenCalledTimes(1));
    const input = mocks.grantCarrierIntelOverride.mock.calls[0]?.[0];
    expect(input).toMatchObject({
      carrierId: "car_1",
      ruleCode: FINDING.code,
      reason: "Renewal certificate on file",
    });
    expect(Math.abs(input.expiresAt - expiresAt)).toBeLessThanOrEqual(5);
    await waitFor(() => expect(onGranted).toHaveBeenCalledTimes(1));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("offers presets that never exceed the 90-day limit", () => {
    renderDialog();

    const presets = screen.getAllByRole("button", { name: /^\d+ days$/ });
    const days = presets.map((button) => Number.parseInt(button.textContent ?? "0", 10));
    expect(Math.max(...days)).toBe(90);
  });
});
