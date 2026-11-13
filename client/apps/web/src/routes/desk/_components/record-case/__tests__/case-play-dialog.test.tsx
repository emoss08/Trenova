import type { WatchtowerPlay } from "@/lib/graphql/watchtower";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render as renderBare, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import type { ReactElement, ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CasePlayDialog } from "../case-play-dialog";

const fetchWatchtowerPlay = vi.fn();
const rewordWatchtowerPlay = vi.fn();
const sendWatchtowerPlay = vi.fn();
const handleMutationError = vi.fn();

vi.mock("@/lib/graphql/watchtower", () => ({
  rewordWatchtowerPlay: (...args: unknown[]) => rewordWatchtowerPlay(...args),
  sendWatchtowerPlay: (...args: unknown[]) => sendWatchtowerPlay(...args),
}));
vi.mock("@/lib/queries/watchtower", () => ({
  watchtower: {
    play: (itemId: string) => ({
      queryKey: ["watchtower", "play", itemId],
      queryFn: () => fetchWatchtowerPlay(itemId),
    }),
  },
}));
vi.mock("@/hooks/use-api-mutation", () => ({
  handleMutationError: (...args: unknown[]) => handleMutationError(...args),
}));

/** A Dash message to the driver who delivered the load, as the server plans it. */
function dashPlay(overrides: Partial<WatchtowerPlay> = {}): WatchtowerPlay {
  return {
    itemId: "wt_1",
    kind: "AskForPod",
    channel: "Dash",
    recipient: "Laura Wright",
    sender: "",
    subject: "",
    message: "Hi Laura, please send the signed proof of delivery for PRO S100.",
    messageLimit: 500,
    summary: "",
    canReword: true,
    ...overrides,
  };
}

function render(ui: ReactElement) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );

  return renderBare(ui, { wrapper });
}

function open(onSent = vi.fn(), onOpenChange = vi.fn()) {
  render(
    <CasePlayDialog itemId="wt_1" play="AskForPod" onOpenChange={onOpenChange} onSent={onSent} />,
  );

  return { onSent, onOpenChange };
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("a condition's first move, sent by the person", () => {
  it("shows who it goes to and how, with the words ready to change", async () => {
    fetchWatchtowerPlay.mockResolvedValue(dashPlay());
    open();

    const box = await screen.findByRole("textbox", { name: "Message" });
    expect((box as HTMLTextAreaElement).value).toBe(
      "Hi Laura, please send the signed proof of delivery for PRO S100.",
    );
    expect(screen.getByText("Laura Wright")).toBeTruthy();
    expect(screen.getByText("Dash message")).toBeTruthy();
    expect(screen.getByText("64 of 500")).toBeTruthy();
    // A Dash message has no sender line or subject.
    expect(screen.queryByText("From")).toBeNull();
    expect(screen.queryByText("Subject")).toBeNull();
    expect(fetchWatchtowerPlay).toHaveBeenCalledWith("wt_1");
    expect(sendWatchtowerPlay).not.toHaveBeenCalled();
  });

  it("sends the words the person left in the box, then tells the case and closes", async () => {
    fetchWatchtowerPlay.mockResolvedValue(dashPlay());
    sendWatchtowerPlay.mockResolvedValue({});
    const { onSent, onOpenChange } = open();

    const box = await screen.findByRole("textbox", { name: "Message" });
    await userEvent.clear(box);
    await userEvent.type(box, "Laura, third ask: we need the POD today.");
    await userEvent.click(screen.getByRole("button", { name: "Send" }));

    await waitFor(() => expect(onSent).toHaveBeenCalledTimes(1));
    expect(sendWatchtowerPlay).toHaveBeenCalledTimes(1);
    expect(sendWatchtowerPlay).toHaveBeenCalledWith(
      "wt_1",
      "Laura, third ask: we need the POD today.",
      expect.objectContaining({ idempotencyKey: expect.any(String) }),
    );
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("shows an email's sender and subject, which are not the person's to change", async () => {
    fetchWatchtowerPlay.mockResolvedValue(
      dashPlay({
        channel: "Email",
        recipient: "Acme Freight",
        sender: "Billing <billing@carrier.example>",
        subject: "Signed proof of delivery for PRO S100",
        messageLimit: 4000,
      }),
    );
    open();

    expect(await screen.findByText("Acme Freight")).toBeTruthy();
    expect(screen.getByText("Email")).toBeTruthy();
    expect(screen.getByText("Billing <billing@carrier.example>")).toBeTruthy();
    expect(screen.getByText("Signed proof of delivery for PRO S100")).toBeTruthy();
    expect(screen.queryByRole("textbox", { name: "Subject" })).toBeNull();
  });

  it("will not send a message too long for how it goes, or an empty one", async () => {
    fetchWatchtowerPlay.mockResolvedValue(dashPlay({ messageLimit: 70 }));
    open();

    const box = await screen.findByRole("textbox", { name: "Message" });
    await userEvent.type(box, " Thank you.");
    expect(screen.getByText("75 of 70")).toBeTruthy();
    expect(screen.getByRole("alert").textContent).toBe(
      "That is too long for how it is sent. Shorten it to send.",
    );
    expect((screen.getByRole("button", { name: "Send" }) as HTMLButtonElement).disabled).toBe(true);

    await userEvent.clear(box);
    expect(screen.getByRole("alert").textContent).toBe("Write the message, or cancel.");
    expect((screen.getByRole("button", { name: "Send" }) as HTMLButtonElement).disabled).toBe(true);
    expect(sendWatchtowerPlay).not.toHaveBeenCalled();
  });

  it("counts the message as the server does, in bytes", async () => {
    fetchWatchtowerPlay.mockResolvedValue(dashPlay({ message: "Señor", messageLimit: 500 }));
    open();

    await screen.findByRole("textbox", { name: "Message" });
    expect(screen.getByText("6 of 500")).toBeTruthy();
  });

  it("has Nova reword what is in the box, and puts the result there to read before sending", async () => {
    fetchWatchtowerPlay.mockResolvedValue(dashPlay());
    rewordWatchtowerPlay.mockResolvedValue("Laura, this is our third request for the signed POD.");
    open();

    await screen.findByRole("textbox", { name: "Message" });
    await userEvent.type(
      screen.getByRole("textbox", { name: "How Nova should change the message" }),
      "firmer, third ask",
    );
    await userEvent.click(screen.getByRole("button", { name: "Reword" }));

    await waitFor(() =>
      expect((screen.getByRole("textbox", { name: "Message" }) as HTMLTextAreaElement).value).toBe(
        "Laura, this is our third request for the signed POD.",
      ),
    );
    expect(rewordWatchtowerPlay).toHaveBeenCalledWith("wt_1", {
      message: "Hi Laura, please send the signed proof of delivery for PRO S100.",
      guidance: "firmer, third ask",
    });
    expect(sendWatchtowerPlay).not.toHaveBeenCalled();
  });

  it("offers no rewording where Nova cannot write", async () => {
    fetchWatchtowerPlay.mockResolvedValue(dashPlay({ canReword: false }));
    open();

    await screen.findByRole("textbox", { name: "Message" });
    expect(screen.queryByRole("button", { name: "Reword" })).toBeNull();
  });

  it("shows a tender as what it does, with no message to write", async () => {
    fetchWatchtowerPlay.mockResolvedValue(
      dashPlay({
        kind: "TenderAgain",
        channel: "Tender",
        recipient: "Acme Freight, Northline Carriers",
        message: "",
        messageLimit: 0,
        summary:
          "Offers the move down routing guide Midwest dry van, one carrier at a time in this order: Acme Freight, Northline Carriers.",
        canReword: false,
      }),
    );
    sendWatchtowerPlay.mockResolvedValue({});
    const { onSent } = open();

    expect(
      await screen.findByText(
        "Offers the move down routing guide Midwest dry van, one carrier at a time in this order: Acme Freight, Northline Carriers.",
      ),
    ).toBeTruthy();
    expect(screen.queryByRole("textbox", { name: "Message" })).toBeNull();
    await userEvent.click(screen.getByRole("button", { name: "Offer it" }));

    await waitFor(() => expect(onSent).toHaveBeenCalledTimes(1));
    expect(sendWatchtowerPlay).toHaveBeenCalledWith("wt_1", "", expect.anything());
  });

  it("says why when it cannot be sent, and offers nothing to send", async () => {
    const reason =
      "Theresa Aguilar has no Dash login, email or phone number on file, so they cannot be asked for the proof of delivery. Upload it here, or add a way to reach them.";
    const type = "https://trenova.app/problems/business-rule-violation";
    fetchWatchtowerPlay.mockRejectedValue(
      new GraphQLRequestError({
        kind: "graphql",
        message: reason,
        status: 200,
        graphQLErrors: [{ message: reason, extensions: { type }, type }],
      }),
    );
    const { onOpenChange } = open();

    expect((await screen.findByRole("alert")).textContent).toBe(reason);
    expect(screen.queryByRole("button", { name: "Send" })).toBeNull();
    await userEvent.click(screen.getByRole("button", { name: "OK" }));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("keeps the dialog open and reports the failure when the send is refused", async () => {
    fetchWatchtowerPlay.mockResolvedValue(dashPlay());
    const refusal = new Error("refused");
    sendWatchtowerPlay.mockRejectedValue(refusal);
    const { onSent, onOpenChange } = open();

    await screen.findByRole("textbox", { name: "Message" });
    await userEvent.click(screen.getByRole("button", { name: "Send" }));

    await waitFor(() => expect(handleMutationError).toHaveBeenCalledTimes(1));
    expect(handleMutationError).toHaveBeenCalledWith(expect.objectContaining({ error: refusal }));
    expect(onSent).not.toHaveBeenCalled();
    expect(onOpenChange).not.toHaveBeenCalled();
    expect((screen.getByRole("button", { name: "Send" }) as HTMLButtonElement).disabled).toBe(
      false,
    );
  });
});
