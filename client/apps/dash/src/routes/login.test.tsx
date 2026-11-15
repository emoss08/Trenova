import { TestRouter } from "@trenova/shared/testing/test-router";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { FakeBroadcastChannel } from "@trenova/shared/testing/fake-broadcast-channel";
import { afterEach, describe, expect, it, vi } from "vitest";

const navigate = vi.hoisted(() => vi.fn());

vi.mock("@tanstack/react-router", async (importActual) => ({
  ...(await importActual<typeof import("@tanstack/react-router")>()),
  useNavigate: () => navigate,
}));

vi.stubGlobal("BroadcastChannel", FakeBroadcastChannel);

const { DashLoginPage } = await import("./login");

describe("DashLoginPage", () => {
  afterEach(cleanup);

  it("goes into the portal when the person finishes signing in in another tab", async () => {
    render(
      <TestRouter>
        <DashLoginPage />
      </TestRouter>,
    );
    await screen.findByLabelText(/email/i);

    new FakeBroadcastChannel("trenova-session-context").postMessage({ kind: "signed-in" });

    await waitFor(() => expect(navigate).toHaveBeenCalledWith({ to: "/dash", replace: true }));
  });

  it("tells a driver that the app and what it shows come from their carrier, and not to use it while driving", async () => {
    render(
      <TestRouter>
        <DashLoginPage />
      </TestRouter>,
    );

    expect(
      await screen.findByText(
        "Dash is provided by your carrier. Routes, assignments and times shown here come from your carrier, not from Trenova. Don't use Dash while your vehicle is moving.",
      ),
    ).toBeInTheDocument();
  });
});
