import { useAppDialogsStore } from "@/stores/app-dialogs-store";
import { useAssistantStore } from "@/stores/assistant-store";
import { ThemeProvider } from "@trenova/shared/components/theme-provider";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { usePaletteIntentRunner } from "../use-palette-intents";

const { copy, toastSuccess, toastError, signOut, navigate } = vi.hoisted(() => ({
  copy: vi.fn<(text: string) => Promise<boolean>>(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
  signOut: vi.fn(async () => undefined),
  navigate: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { success: toastSuccess, error: toastError } }));
vi.mock("@/hooks/use-copy-to-clipboard", () => ({
  useCopyToClipboard: () => ({ copy, text: null, isCopied: false }),
}));
vi.mock("@/hooks/use-sign-out", () => ({ useSignOut: () => signOut }));
vi.mock("react-router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-router")>()),
  useNavigate: () => navigate,
}));

function setup() {
  const close = vi.fn();
  const askAbout = vi.fn();
  const client = new QueryClient();
  const { result } = renderHook(() => usePaletteIntentRunner({ close, askAbout }), {
    wrapper: ({ children }) => (
      <QueryClientProvider client={client}>
        <ThemeProvider defaultTheme="light" storageKey="palette-intents-test">
          <MemoryRouter>{children}</MemoryRouter>
        </ThemeProvider>
      </QueryClientProvider>
    ),
  });
  return { run: result.current, close, askAbout };
}

describe("usePaletteIntentRunner", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useAppDialogsStore.setState({ active: null });
    useAssistantStore.setState({ open: false, activeThreadId: "thr_1" });
    vi.spyOn(window, "open").mockImplementation(() => null);
  });

  it("copies the text itself and says only that it was copied, leaving the palette open", async () => {
    copy.mockResolvedValue(true);
    const { run, close } = setup();

    await act(async () => run({ type: "copy", text: "PRO-1001" }));

    expect(copy).toHaveBeenCalledWith("PRO-1001", { withToast: false });
    expect(toastSuccess).toHaveBeenCalledWith("Copied to clipboard");
    expect(close).not.toHaveBeenCalled();
  });

  it("copies a link as an absolute address", async () => {
    copy.mockResolvedValue(true);
    const { run } = setup();

    await act(async () => run({ type: "copy-link", href: "/billing/invoices?item=inv_1" }));

    expect(copy).toHaveBeenCalledWith(`${window.location.origin}/billing/invoices?item=inv_1`, {
      withToast: false,
    });
  });

  it("says so when the clipboard refuses", async () => {
    copy.mockResolvedValue(false);
    const { run } = setup();

    await act(async () => run({ type: "copy", text: "x" }));

    expect(toastError).toHaveBeenCalledWith("Couldn't copy to the clipboard");
    expect(toastSuccess).not.toHaveBeenCalled();
  });

  it("closes before navigating", () => {
    const { run, close } = setup();

    run({ type: "navigate", href: "/hr/workers" });

    expect(close).toHaveBeenCalled();
    expect(navigate).toHaveBeenCalledWith("/hr/workers");
  });

  it("opens a new tab without handing the page a handle back to this one", () => {
    const { run } = setup();

    run({ type: "new-tab", href: "/hr/workers" });

    expect(window.open).toHaveBeenCalledWith(
      `${window.location.origin}/hr/workers`,
      "_blank",
      "noopener,noreferrer",
    );
  });

  it("opens a document's content through its own address", () => {
    const { run } = setup();

    run({ type: "open-document", documentId: "doc 1", disposition: "download" });

    expect(vi.mocked(window.open).mock.calls[0]?.[0]).toMatch(/\/documents\/doc%201\/download\/$/);
  });

  it("opens a dialog through the shared dialog store", () => {
    const { run } = setup();

    run({ type: "open-dialog", dialog: "shortcuts" });

    expect(useAppDialogsStore.getState().active).toBe("shortcuts");
  });

  it("starts a new conversation by clearing the active thread before opening the assistant", () => {
    const { run } = setup();

    run({ type: "assistant", mode: "new-chat" });

    expect(useAssistantStore.getState()).toMatchObject({ open: true, activeThreadId: null });
  });

  it("keeps the active conversation when only opening the assistant", () => {
    const { run } = setup();

    run({ type: "assistant", mode: "open" });

    expect(useAssistantStore.getState()).toMatchObject({ open: true, activeThreadId: "thr_1" });
  });

  it("signs out through the one sign-out path", () => {
    const { run, close } = setup();

    run({ type: "sign-out" });

    expect(close).toHaveBeenCalled();
    expect(signOut).toHaveBeenCalled();
  });

  it("hands a record to the palette to ask about, without closing", () => {
    const { run, close, askAbout } = setup();
    const subject = { type: "shipment", id: "shp_1", label: "PRO-1" };

    run({ type: "ask-about", subject });

    expect(askAbout).toHaveBeenCalledWith(subject);
    expect(close).not.toHaveBeenCalled();
  });
});
