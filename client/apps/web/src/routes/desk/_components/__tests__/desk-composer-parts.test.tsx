import type { ComposerAttachment } from "@/components/assistant/composer";
import type { AssistantProviderOption } from "@/types/assistant";
import { act, fireEvent, render, renderHook, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useDeskAttachments, type AttachmentSource } from "../composer/desk-attachments";
import { DeskModelPicker } from "../composer/desk-model-picker";

function source(attachments: ComposerAttachment[] = []): AttachmentSource {
  return {
    attachments,
    attachFiles: vi.fn(),
    removeAttachment: vi.fn(),
    retryAttachment: vi.fn(),
  };
}

const file = (name: string, size = 1000) => new File([new Uint8Array(size)], name);

describe("useDeskAttachments", () => {
  it("uploads what the Desk can read and refuses the rest on the spot", async () => {
    const upload = source();
    const { result } = renderHook(() => useDeskAttachments(upload));

    act(() => result.current.add([file("rate-con.pdf"), file("packet.zip")]));

    await waitFor(() => expect(upload.attachFiles).toHaveBeenCalled());
    expect(upload.attachFiles).toHaveBeenCalledWith([
      expect.objectContaining({ name: "rate-con.pdf" }),
    ]);
    expect(result.current.items).toHaveLength(1);
    expect(result.current.items[0]).toMatchObject({
      name: "packet.zip",
      status: "error",
      refused: true,
    });
    expect(result.current.failed).toBe(1);
  });

  it("refuses a file over the size limit", () => {
    const { result } = renderHook(() => useDeskAttachments(source()));

    act(() => result.current.add([file("scan.pdf", 26 * 1024 * 1024)]));

    expect(result.current.items[0].error).toMatch(/Too large/);
  });

  it("takes five files at most and says how many did not fit", async () => {
    const upload = source();
    const { result } = renderHook(() => useDeskAttachments(upload));

    act(() =>
      result.current.add(Array.from({ length: 7 }, (_, index) => file(`page-${index}.pdf`))),
    );

    await waitFor(() => expect(upload.attachFiles).toHaveBeenCalled());
    expect(upload.attachFiles).toHaveBeenCalledWith(expect.arrayContaining([expect.any(File)]));
    expect((upload.attachFiles as ReturnType<typeof vi.fn>).mock.calls[0][0]).toHaveLength(5);
    expect(result.current.note).toBe("Up to 5 files per message · 2 not added");
  });

  it("refuses a PDF locked with a password before it uploads", async () => {
    const upload = source();
    const { result } = renderHook(() => useDeskAttachments(upload));
    const locked = new File(
      ["%PDF-1.7\ntrailer << /Root 1 0 R /Encrypt 4 0 R >>\n%%EOF"],
      "locked.pdf",
      {
        type: "application/pdf",
      },
    );

    act(() => result.current.add([locked]));

    await waitFor(() => expect(result.current.items).toHaveLength(1));
    expect(result.current.items[0]).toMatchObject({
      name: "locked.pdf",
      error: "Password-protected · Desk can't open it",
      refused: true,
    });
    expect(upload.attachFiles).not.toHaveBeenCalled();
  });

  it("removes a refused file locally and an uploaded one through its source", () => {
    const upload = source([
      { id: "up-1", name: "a.pdf", size: 1, status: "ready", progress: 1, documentId: "doc_1" },
    ]);
    const { result } = renderHook(() => useDeskAttachments(upload));
    act(() => result.current.add([file("b.exe")]));
    const refused = result.current.items.find((item) => item.refused);

    act(() => result.current.remove(refused?.id ?? ""));
    act(() => result.current.remove("up-1"));

    expect(result.current.items.some((item) => item.refused)).toBe(false);
    expect(upload.removeAttachment).toHaveBeenCalledWith("up-1");
  });
});

const providers: AssistantProviderOption[] = [
  {
    id: "p1",
    name: "Anthropic · primary",
    kind: "AnthropicMessages",
    model: "Claude Sonnet",
    trusted: true,
    vendor: "anthropic",
  },
  {
    id: "p2",
    name: "Groq",
    kind: "OpenAIChat",
    model: "Llama 3.3 70B",
    trusted: false,
    vendor: "groq",
    unavailable: true,
  },
];

describe("DeskModelPicker", () => {
  it("offers Auto first and each endpoint under its vendor", () => {
    render(<DeskModelPicker options={providers} value="" onChange={vi.fn()} hasReplies={false} />);
    fireEvent.click(screen.getByRole("button", { name: "Choose which model answers" }));

    const options = screen.getAllByRole("option");
    expect(options[0]).toHaveTextContent("Auto");
    expect(options[0]).toHaveAttribute("aria-selected", "true");
    expect(screen.getByText("Anthropic")).toBeInTheDocument();
    expect(screen.getByText("Primary")).toBeInTheDocument();
  });

  it("will not pick an endpoint that failed its last check", () => {
    const onChange = vi.fn();
    render(<DeskModelPicker options={providers} value="" onChange={onChange} hasReplies />);
    fireEvent.click(screen.getByRole("button", { name: "Choose which model answers" }));

    fireEvent.click(screen.getByRole("option", { name: /Llama 3.3 70B/ }));
    expect(onChange).not.toHaveBeenCalled();
    expect(screen.getByText("Unavailable")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("option", { name: /Claude Sonnet/ }));
    expect(onChange).toHaveBeenCalledWith("p1");
  });

  it("says switching re-reads the conversation once it has replies", () => {
    render(<DeskModelPicker options={providers} value="p1" onChange={vi.fn()} hasReplies />);
    fireEvent.click(screen.getByRole("button", { name: "Choose which model answers" }));

    expect(
      screen.getByText("Switching re-reads this conversation before the next reply."),
    ).toBeInTheDocument();
  });
});
