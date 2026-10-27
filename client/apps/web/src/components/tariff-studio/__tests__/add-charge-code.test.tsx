import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { TariffDiagnostic } from "@trenova/shared/types/tariff";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AddChargeCode } from "../add-charge-code";

const permission = vi.hoisted(() => ({ allowed: true }));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: permission.allowed }),
}));
vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: { error: vi.fn(), success: vi.fn() },
}));

afterEach(() => {
  cleanup();
  permission.allowed = true;
});

const SOURCE = 'tariff "t"\ncharge "Fuel surcharge" as FSC = $75';

function finding(code = "unknown-code"): TariffDiagnostic {
  const from = SOURCE.indexOf("FSC");
  return {
    code,
    severity: "error",
    message: { message: "{0} isn't in the accessorial charge catalog", args: ["`FSC`"] },
    range: { from, to: from + 3 },
    fixes: [],
    related: [],
    module: null,
    path: null,
  } as unknown as TariffDiagnostic;
}

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

describe("AddChargeCode", () => {
  it("opens the accessorial charge panel with the code and the charge's label filled in", async () => {
    const user = userEvent.setup();
    render(<AddChargeCode diagnostic={finding()} source={SOURCE} />, { wrapper });

    await user.click(screen.getByRole("button", { name: "Add FSC to accessorial charges" }));

    expect(await screen.findByDisplayValue("FSC")).toBeInTheDocument();
    expect(screen.getByDisplayValue("Fuel surcharge")).toBeInTheDocument();
  });

  it("is hidden from someone who can't create accessorial charges", () => {
    permission.allowed = false;
    render(<AddChargeCode diagnostic={finding()} source={SOURCE} />, { wrapper });
    expect(screen.queryByRole("button")).toBeNull();
  });

  it("is hidden on any other finding", () => {
    render(<AddChargeCode diagnostic={finding("inactive-code")} source={SOURCE} />, { wrapper });
    expect(screen.queryByRole("button")).toBeNull();
  });
});
