import { render, screen, waitFor } from "@testing-library/react";
import { FakeBroadcastChannel } from "@trenova/shared/testing/fake-broadcast-channel";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

const navigate = vi.hoisted(() => vi.fn());

vi.mock("react-router", async (importActual) => ({
  ...(await importActual<typeof import("react-router")>()),
  useNavigate: () => navigate,
}));

vi.stubGlobal("BroadcastChannel", FakeBroadcastChannel);

const { DashLoginPage } = await import("./login");

describe("DashLoginPage", () => {
  it("goes into the portal when the person finishes signing in in another tab", async () => {
    render(
      <MemoryRouter>
        <DashLoginPage />
      </MemoryRouter>,
    );
    await screen.findByLabelText(/email/i);

    new FakeBroadcastChannel("trenova-session-context").postMessage({ kind: "signed-in" });

    await waitFor(() => expect(navigate).toHaveBeenCalledWith("/dash", { replace: true }));
  });
});
