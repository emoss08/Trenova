import type { EDICommunicationProfileRow } from "@/lib/graphql/edi-table";
import { ediService } from "@/services/edi";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AS2InboundURLSection } from "../panel/edi-as2-inbound-url";

const permission = vi.hoisted(() => ({ allowed: true }));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: permission.allowed, isLoading: false }),
}));

vi.mock("@/services/edi", () => ({
  ediService: { rotateInboundUrl: vi.fn() },
}));

const PATH = "/api/v1/edi/as2/inbound/PROFILEADDRESS/";

function profileRow(as2InboundPath: string | null): EDICommunicationProfileRow {
  return { id: "edicp_1", as2InboundPath } as EDICommunicationProfileRow;
}

function renderSection(profile: EDICommunicationProfileRow | null) {
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });

  return render(
    <QueryClientProvider client={client}>
      <AS2InboundURLSection profile={profile} />
    </QueryClientProvider>,
  );
}

describe("AS2InboundURLSection", () => {
  afterEach(() => {
    cleanup();
    permission.allowed = true;
    vi.clearAllMocks();
  });

  it("shows the full URL a partner posts to", () => {
    renderSection(profileRow(PATH));

    expect(screen.getByText(new RegExp(`${PATH}$`))).toBeTruthy();
  });

  it("asks for a save before a profile has a URL", () => {
    renderSection(null);

    expect(screen.getByText(/Save this AS2 profile/)).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Rotate URL" })).toBeNull();
  });

  it("offers no rotation to someone who cannot edit EDI", () => {
    permission.allowed = false;
    renderSection(profileRow(PATH));

    expect(screen.queryByRole("button", { name: "Rotate URL" })).toBeNull();
  });

  it("shows the new URL once a rotation is confirmed", async () => {
    vi.mocked(ediService.rotateInboundUrl).mockResolvedValue({
      as2InboundPath: "/api/v1/edi/as2/inbound/NEWADDRESS/",
    } as Awaited<ReturnType<typeof ediService.rotateInboundUrl>>);
    const user = userEvent.setup();
    renderSection(profileRow(PATH));

    await user.click(screen.getByRole("button", { name: "Rotate URL" }));
    expect(ediService.rotateInboundUrl).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Rotate" }));

    await waitFor(() => expect(screen.getByText(/NEWADDRESS\/$/)).toBeTruthy());
    expect(ediService.rotateInboundUrl).toHaveBeenCalledWith("edicp_1");
  });
});
