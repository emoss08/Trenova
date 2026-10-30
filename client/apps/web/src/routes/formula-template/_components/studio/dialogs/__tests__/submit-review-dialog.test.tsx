import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { FormulaTemplate } from "@trenova/shared/types/formula-template";
import { useEffect } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { SubmitReviewDialog } from "../submit-review-dialog";

const mocks = vi.hoisted(() => ({
  draftReviewNote: vi.fn(),
  submit: vi.fn(),
  listReviewers: vi.fn(),
  approvalImpact: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiService: { formulaTemplateService: mocks },
}));

vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: Object.assign(vi.fn(), { error: vi.fn() }),
}));

vi.mock("../../../readiness-panel", () => ({
  ReadinessPanel: ({ onReadinessChange }: { onReadinessChange?: (ready: boolean) => void }) => {
    useEffect(() => onReadinessChange?.(true), [onReadinessChange]);
    return null;
  },
}));

const template = { id: "ft_01", name: "LTL by weight break" } as unknown as FormulaTemplate;

const reviewers = [
  { id: "usr_dana", name: "Dana Whitfield", username: "dana.whitfield" },
  { id: "usr_marcus", name: "Marcus Lee", username: "marcus.lee" },
];

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (error: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

function renderDialog(open = true) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const onOpenChange = vi.fn();
  const view = render(
    <QueryClientProvider client={client}>
      <SubmitReviewDialog open={open} onOpenChange={onOpenChange} template={template} />
    </QueryClientProvider>,
  );
  const rerender = (next: boolean) =>
    view.rerender(
      <QueryClientProvider client={client}>
        <SubmitReviewDialog open={next} onOpenChange={onOpenChange} template={template} />
      </QueryClientProvider>,
    );
  return { onOpenChange, rerender };
}

const note = () => screen.getByRole("textbox", { name: "Note for the reviewer" });
const submitButton = () => screen.getByRole("button", { name: "Submit" });

describe("SubmitReviewDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.listReviewers.mockResolvedValue(reviewers);
    mocks.approvalImpact.mockReturnValue(new Promise(() => {}));
    mocks.submit.mockResolvedValue({ ...template, status: "InReview" });
  });

  it("lets the person write a note and submit while Nova is still drafting", async () => {
    const user = userEvent.setup();
    mocks.draftReviewNote.mockReturnValue(new Promise(() => {}));
    renderDialog();

    await user.type(note(), "Adds deficit rating.");
    await waitFor(() => expect(submitButton()).toBeEnabled());
    await user.click(submitButton());

    await waitFor(() =>
      expect(mocks.submit).toHaveBeenCalledWith("ft_01", {
        comment: "",
        reviewerId: "usr_dana",
        reviewerNote: "Adds deficit rating.",
      }),
    );
  });

  it("fills an empty note with Nova's draft", async () => {
    mocks.draftReviewNote.mockResolvedValue({
      note: "Nova's summary of the change.",
      generated: true,
    });
    renderDialog();

    await waitFor(() => expect(note()).toHaveValue("Nova's summary of the change."));
  });

  it("offers a draft that arrives after the person started writing, without replacing their words", async () => {
    const user = userEvent.setup();
    const draft = deferred<{ note: string; generated: boolean }>();
    mocks.draftReviewNote.mockReturnValue(draft.promise);
    renderDialog();

    await user.type(note(), "My own note.");
    await act(async () =>
      draft.resolve({ note: "Nova's summary of the change.", generated: true }),
    );

    expect(note()).toHaveValue("My own note.");
    expect(screen.getByText("Nova drafted a note while you were writing.")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Use Nova's draft" }));

    expect(note()).toHaveValue("Nova's summary of the change.");
    expect(screen.queryByText("Nova drafted a note while you were writing.")).toBeNull();
  });

  it("keeps the person's note when they turn Nova's late draft down", async () => {
    const user = userEvent.setup();
    const draft = deferred<{ note: string; generated: boolean }>();
    mocks.draftReviewNote.mockReturnValue(draft.promise);
    renderDialog();

    await user.type(note(), "My own note.");
    await act(async () =>
      draft.resolve({ note: "Nova's summary of the change.", generated: true }),
    );
    await user.click(screen.getByRole("button", { name: "Keep my note" }));

    expect(note()).toHaveValue("My own note.");
    expect(screen.queryByRole("button", { name: "Use Nova's draft" })).toBeNull();
  });

  it("replaces the note when the person asks Nova to rewrite it", async () => {
    const user = userEvent.setup();
    mocks.draftReviewNote
      .mockResolvedValueOnce({ note: "First draft.", generated: true })
      .mockResolvedValueOnce({ note: "Second draft.", generated: true });
    renderDialog();
    await waitFor(() => expect(note()).toHaveValue("First draft."));

    await user.clear(note());
    await user.type(note(), "Edited by hand.");
    await user.click(screen.getByRole("button", { name: "Rewrite" }));

    await waitFor(() => expect(note()).toHaveValue("Second draft."));
  });

  it("cancels Nova's draft when the dialog closes", async () => {
    mocks.draftReviewNote.mockReturnValue(new Promise(() => {}));
    const { rerender } = renderDialog();
    await waitFor(() => expect(mocks.draftReviewNote).toHaveBeenCalledTimes(1));
    const { signal } = mocks.draftReviewNote.mock.calls[0]![1] as { signal: AbortSignal };

    rerender(false);

    expect(signal.aborted).toBe(true);
  });

  it("says Nova could not draft the note, and leaves the note to the person, when the draft fails", async () => {
    mocks.draftReviewNote.mockRejectedValue(
      new Error("Formula assistant could not answer just now."),
    );
    renderDialog();

    expect(
      await screen.findByText("Formula assistant could not answer just now."),
    ).toBeInTheDocument();
    expect(note()).toHaveValue("");
    expect(note()).toBeEnabled();
  });
});
