import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { LoadChat } from "./load-chat";

const { fetchMyLoadComments, createMyLoadComment, fetchMyPortalFeatures, scrollToEnd, toast } =
  vi.hoisted(() => ({
    fetchMyLoadComments: vi.fn(),
    createMyLoadComment: vi.fn(),
    fetchMyPortalFeatures: vi.fn(),
    scrollToEnd: vi.fn(),
    toast: { success: vi.fn(), error: vi.fn() },
  }));

vi.mock("@trenova/shared/lib/graphql/driver-portal", () => ({
  fetchMyLoadComments,
  createMyLoadComment,
  fetchMyPortalFeatures,
}));

vi.mock("@trenova/shared/components/ui/message-scroller", async (importOriginal) => {
  const actual =
    await importOriginal<typeof import("@trenova/shared/components/ui/message-scroller")>();
  return {
    ...actual,
    useMessageScroller: () => ({
      scrollToEnd,
      scrollToMessage: vi.fn(),
      scrollToStart: vi.fn(),
    }),
  };
});

vi.mock("sonner", () => ({ toast }));

const now = 1_800_000_000;

const comments = [
  {
    id: "cmt_3",
    comment: "Running ten minutes late",
    type: "DriverUpdate",
    priority: "Normal",
    authorName: "Me",
    createdAt: now,
  },
  {
    id: "cmt_2",
    comment: "Dock 4 when you arrive",
    type: "Dispatch",
    priority: "Normal",
    authorName: "Dispatch",
    createdAt: now - 600,
  },
  {
    id: "cmt_1",
    comment: "On my way to pickup",
    type: "DriverUpdate",
    priority: "Normal",
    authorName: "Me",
    createdAt: now - 1200,
  },
];

function renderChat() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <LoadChat shipmentId="shp_1" />
    </QueryClientProvider>,
  );
}

describe("LoadChat", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("lists the thread oldest first without pinning the driver's own messages", async () => {
    fetchMyLoadComments.mockResolvedValue(comments);
    fetchMyPortalFeatures.mockResolvedValue(undefined);

    const { container } = renderChat();

    await screen.findByText("Running ten minutes late");
    const items = Array.from(container.querySelectorAll("[data-message-id]"));
    expect(items.map((item) => item.getAttribute("data-message-id"))).toEqual([
      "cmt_1",
      "cmt_2",
      "cmt_3",
    ]);
    for (const item of items) {
      expect(item).toHaveAttribute("data-scroll-anchor", "false");
    }
  });

  it("scrolls to the newest message once the driver's message is sent", async () => {
    fetchMyLoadComments.mockResolvedValue(comments);
    fetchMyPortalFeatures.mockResolvedValue(undefined);
    createMyLoadComment.mockResolvedValue({ id: "cmt_4" });

    renderChat();

    const input = await screen.findByPlaceholderText("Message dispatch...");
    fireEvent.change(input, { target: { value: "Arrived at the dock" } });
    fireEvent.keyDown(input, { key: "Enter" });

    await waitFor(() => expect(scrollToEnd).toHaveBeenCalledWith({ behavior: "smooth" }));
    expect(createMyLoadComment).toHaveBeenCalledWith({
      shipmentId: "shp_1",
      comment: "Arrived at the dock",
    });
    await waitFor(() => expect(fetchMyLoadComments).toHaveBeenCalledTimes(2));
  });

  it("does not scroll when the message fails to send", async () => {
    fetchMyLoadComments.mockResolvedValue(comments);
    fetchMyPortalFeatures.mockResolvedValue(undefined);
    createMyLoadComment.mockRejectedValue(new Error("Load is closed"));

    renderChat();

    const input = await screen.findByPlaceholderText("Message dispatch...");
    fireEvent.change(input, { target: { value: "Arrived at the dock" } });
    fireEvent.keyDown(input, { key: "Enter" });

    await waitFor(() => expect(toast.error).toHaveBeenCalledWith("Load is closed"));
    expect(scrollToEnd).not.toHaveBeenCalled();
  });
});
