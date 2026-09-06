import {
  LazyCommentEditor,
  preloadCommentEditor,
} from "@trenova/shared/components/comment-editor/lazy-comment-editor";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

// The real editor is TipTap/ProseMirror — roughly 500 kB of chunk. It is the
// only thing on the panel that renders live formatting controls, so the
// presence of a labelled "Bold" button is what separates "TipTap is mounted"
// from "we are still showing the placeholder frame".
const boldButton = { name: "Bold" } as const;

const fetchMentionCandidates = vi.fn().mockResolvedValue({ items: [], hasMore: false });

function renderEditor(props: { autoFocus?: boolean } = {}) {
  return render(
    <LazyCommentEditor
      placeholder="Add a comment… @ to mention"
      fetchMentionCandidates={fetchMentionCandidates}
      toolbar={<button type="button">Send</button>}
      {...props}
    />,
  );
}

// Transforming TipTap and its ProseMirror dependencies takes several seconds
// under vitest. Paying that once up front keeps it out of the per-assertion
// wait windows below — what the tests care about is when the component chooses
// to render the editor, not how long the module takes to arrive.
beforeAll(async () => {
  await preloadCommentEditor();
}, 60_000);

afterEach(cleanup);

describe("LazyCommentEditor", () => {
  it("renders the placeholder frame without mounting the editor", () => {
    renderEditor();

    expect(screen.getByRole("textbox", { name: "Comment editor" })).toHaveTextContent(
      "Add a comment… @ to mention",
    );
    expect(screen.queryByRole("button", boldButton)).not.toBeInTheDocument();
  });

  it("keeps the caller's toolbar visible while the editor is still a placeholder", () => {
    renderEditor();

    expect(screen.getByRole("button", { name: "Send" })).toBeInTheDocument();
  });

  it("mounts the editor once the placeholder is pointed at", async () => {
    renderEditor();

    expect(screen.queryByRole("button", boldButton)).not.toBeInTheDocument();
    fireEvent.pointerDown(screen.getByRole("textbox", { name: "Comment editor" }));

    expect(await screen.findByRole("button", boldButton)).toBeInTheDocument();
  });

  it("mounts the editor immediately when autoFocus is requested", async () => {
    renderEditor({ autoFocus: true });

    expect(await screen.findByRole("button", boldButton)).toBeInTheDocument();
  });
});
