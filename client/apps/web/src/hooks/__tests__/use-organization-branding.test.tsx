import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useOrganizationBranding } from "../use-organization-branding";

const mocks = vi.hoisted(() => ({ getBranding: vi.fn() }));

vi.mock("@/services/organization", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/services/organization")>()),
  organizationService: { getBranding: mocks.getBranding },
}));

const signed = (name: string, signature: string) =>
  `https://storage.example.com/org_1/${name}.webp?X-Amz-Signature=${signature}`;

/** A screen that draws the logo from its signed address, as the sidebar and the settings do. */
function Logo({ enabled }: { enabled?: boolean }) {
  const { branding, refresh } = useOrganizationBranding("org_1", enabled);
  const url = branding?.logoUrl;

  return url ? <img src={url} alt="Logo" onError={() => refresh(url)} /> : <span>No logo</span>;
}

function renderLogo(props: { enabled?: boolean } = {}) {
  const client = new QueryClient();
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  return render(<Logo {...props} />, { wrapper });
}

afterEach(() => {
  vi.clearAllMocks();
});

describe("an organization's branding addresses", () => {
  it("are read again when a picture fails to load, and the picture is drawn from the new one", async () => {
    mocks.getBranding
      .mockResolvedValueOnce({ logoUrl: signed("logo", "lapsed"), iconUrl: "" })
      .mockResolvedValueOnce({ logoUrl: signed("logo", "fresh"), iconUrl: "" });
    renderLogo();

    fireEvent.error(await screen.findByRole("img", { name: "Logo" }));

    await waitFor(() =>
      expect(screen.getByRole("img", { name: "Logo" })).toHaveAttribute(
        "src",
        signed("logo", "fresh"),
      ),
    );
    expect(mocks.getBranding).toHaveBeenCalledTimes(2);
  });

  it("are read once more for an address that failed, however often its picture says so", async () => {
    mocks.getBranding.mockResolvedValue({ logoUrl: signed("logo", "gone"), iconUrl: "" });
    renderLogo();
    const logo = await screen.findByRole("img", { name: "Logo" });

    fireEvent.error(logo);
    await waitFor(() => expect(mocks.getBranding).toHaveBeenCalledTimes(2));
    fireEvent.error(logo);
    fireEvent.error(logo);

    await new Promise((resolve) => setTimeout(resolve, 20));
    expect(mocks.getBranding).toHaveBeenCalledTimes(2);
  });

  it("are not read for an organization with no pictures", () => {
    renderLogo({ enabled: false });

    expect(screen.getByText("No logo")).toBeInTheDocument();
    expect(mocks.getBranding).not.toHaveBeenCalled();
  });
});
