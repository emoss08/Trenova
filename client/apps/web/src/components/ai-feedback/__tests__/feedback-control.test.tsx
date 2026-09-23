import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { AiFeedback, AiFeedbackTarget } from "@/lib/graphql/ai-feedback";
import { FeedbackControl } from "../feedback-control";

const mocks = vi.hoisted(() => ({
  fetchMyAiFeedback: vi.fn(),
  setMyAiFeedback: vi.fn(),
  clearMyAiFeedback: vi.fn(),
  toastError: vi.fn(),
}));

vi.mock("sonner", () => ({ toast: { error: mocks.toastError, success: vi.fn() } }));

vi.mock("@/lib/graphql/ai-feedback", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/graphql/ai-feedback")>();

  return {
    ...actual,
    fetchMyAiFeedback: mocks.fetchMyAiFeedback,
    setMyAiFeedback: mocks.setMyAiFeedback,
    clearMyAiFeedback: mocks.clearMyAiFeedback,
  };
});

function saved(target: AiFeedbackTarget, rating: 1 | -1): AiFeedback {
  return {
    id: `aifb_${target.targetId}`,
    targetType: target.targetType,
    targetId: target.targetId,
    targetPart: target.targetPart ?? "",
    rating,
    reasons: [],
    comment: "",
    version: 1,
    updatedAt: 1_760_000_000,
  };
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (error: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });

  return { promise, resolve, reject };
}

function renderControls(targets: AiFeedbackTarget[]) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  }

  return render(
    <>
      {targets.map((target) => (
        <FeedbackControl key={target.targetId} target={target} />
      ))}
    </>,
    { wrapper: Wrapper },
  );
}

const answer: AiFeedbackTarget = { targetType: "AssistantMessage", targetId: "amsg_answer" };

describe("FeedbackControl", () => {
  beforeEach(() => {
    mocks.fetchMyAiFeedback.mockReset();
    mocks.setMyAiFeedback.mockReset();
    mocks.clearMyAiFeedback.mockReset();
    mocks.toastError.mockReset();
    mocks.fetchMyAiFeedback.mockResolvedValue([]);
  });

  it("reads every rating on the screen in one request", async () => {
    const other: AiFeedbackTarget = {
      targetType: "BriefingSection",
      targetId: "brf_1",
      targetPart: "dispatch",
    };
    mocks.fetchMyAiFeedback.mockResolvedValue([saved(other, -1)]);

    renderControls([answer, other]);

    await waitFor(() => expect(mocks.fetchMyAiFeedback).toHaveBeenCalledTimes(1));
    const [targets] = mocks.fetchMyAiFeedback.mock.calls[0];
    expect(targets).toEqual([
      { targetType: "AssistantMessage", targetId: "amsg_answer", targetPart: "" },
      { targetType: "BriefingSection", targetId: "brf_1", targetPart: "dispatch" },
    ]);

    const downs = await screen.findAllByRole("button", { name: "Not helpful" });
    await waitFor(() => expect(downs[1]).toHaveAttribute("aria-pressed", "true"));
    expect(downs[0]).toHaveAttribute("aria-pressed", "false");
  });

  it("shows the thumb as chosen before the server answers", async () => {
    const pending = deferred<AiFeedback>();
    mocks.setMyAiFeedback.mockReturnValue(pending.promise);

    renderControls([answer]);
    await waitFor(() => expect(mocks.fetchMyAiFeedback).toHaveBeenCalled());

    const up = screen.getByRole("button", { name: "Helpful" });
    await userEvent.click(up);

    await waitFor(() =>
      expect(mocks.setMyAiFeedback).toHaveBeenCalledWith(
        { targetType: "AssistantMessage", targetId: "amsg_answer", targetPart: "" },
        { rating: 1, reasons: [], comment: "" },
      ),
    );
    expect(up).toHaveAttribute("aria-pressed", "true");

    pending.resolve(saved(answer, 1));
    await waitFor(() => expect(up).toHaveAttribute("aria-pressed", "true"));
    expect(mocks.toastError).not.toHaveBeenCalled();
  });

  it("puts the rating back and says so when the server refuses it", async () => {
    mocks.fetchMyAiFeedback.mockResolvedValue([saved(answer, 1)]);
    const pending = deferred<AiFeedback>();
    mocks.setMyAiFeedback.mockReturnValue(pending.promise);

    renderControls([answer]);

    const up = screen.getByRole("button", { name: "Helpful" });
    const down = screen.getByRole("button", { name: "Not helpful" });
    await waitFor(() => expect(up).toHaveAttribute("aria-pressed", "true"));

    await userEvent.click(down);
    await waitFor(() => expect(mocks.setMyAiFeedback).toHaveBeenCalled());
    expect(down).toHaveAttribute("aria-pressed", "true");
    expect(up).toHaveAttribute("aria-pressed", "false");

    pending.reject(new Error("refused"));

    await waitFor(() => expect(up).toHaveAttribute("aria-pressed", "true"));
    expect(down).toHaveAttribute("aria-pressed", "false");
    expect(mocks.toastError).toHaveBeenCalledWith("Your rating could not be saved");
    await waitFor(() => expect(screen.queryByText("What went wrong?")).not.toBeInTheDocument());
  });

  it("takes the rating back when the chosen thumb is pressed again", async () => {
    mocks.fetchMyAiFeedback.mockResolvedValue([saved(answer, -1)]);
    mocks.clearMyAiFeedback.mockResolvedValue(true);

    renderControls([answer]);

    const down = screen.getByRole("button", { name: "Not helpful" });
    await waitFor(() => expect(down).toHaveAttribute("aria-pressed", "true"));

    await userEvent.click(down);

    await waitFor(() => expect(down).toHaveAttribute("aria-pressed", "false"));
    await waitFor(() =>
      expect(mocks.clearMyAiFeedback).toHaveBeenCalledWith({
        targetType: "AssistantMessage",
        targetId: "amsg_answer",
        targetPart: "",
      }),
    );
    expect(mocks.setMyAiFeedback).not.toHaveBeenCalled();
  });

  it("restores a cleared rating when clearing fails", async () => {
    mocks.fetchMyAiFeedback.mockResolvedValue([saved(answer, -1)]);
    mocks.clearMyAiFeedback.mockRejectedValue(new Error("offline"));

    renderControls([answer]);

    const down = screen.getByRole("button", { name: "Not helpful" });
    await waitFor(() => expect(down).toHaveAttribute("aria-pressed", "true"));

    await userEvent.click(down);

    await waitFor(() => expect(mocks.toastError).toHaveBeenCalled());
    expect(down).toHaveAttribute("aria-pressed", "true");
  });
});
