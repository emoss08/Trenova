import { describe, expect, it } from "vitest";
import { workNoteGist, workNoteHasMore } from "../work-note";

/**
 * An agent's note is a report; the queue shows one plain sentence of it and
 * keeps the rest for whoever asks.
 */
describe("the gist of what an agent wrote", () => {
  it("is what was found, past the note's own lead-in and its first heading", () => {
    const note =
      "Summary for Detention at Phoenix Cold Storage on SEED-DET-002 (FreshHaul Foods, occurrence): **What I found** - The driver arrived yesterday (2026-11-09) at 09:56 PST, against a 09:36 appointment. Free time is 60 minutes. - The notice window closed. **What I did** - I did not send a notice.";

    expect(workNoteGist(note)).toBe(
      "The driver arrived yesterday (2026-11-09) at 09:56 PST, against a 09:36 appointment.",
    );
    expect(workNoteHasMore(note)).toBe(true);
  });

  it("keeps a note that is one bold sentence, without its marks", () => {
    const note = "**SEED-DET-011 is not blocked, so I made no changes.**";

    expect(workNoteGist(note)).toBe("SEED-DET-011 is not blocked, so I made no changes.");
    expect(workNoteHasMore(note)).toBe(false);
  });

  it("reads plain prose as it is and takes its first sentence", () => {
    expect(
      workNoteGist(
        "No billing queue item exists for PRO SEED-SHP-005, so nothing changed. More follows.",
      ),
    ).toBe("No billing queue item exists for PRO SEED-SHP-005, so nothing changed.");
  });

  it("takes out list marks, code marks and links", () => {
    expect(workNoteGist("- Status is `ReadyToInvoice` on [the shipment](/shipments/1).")).toBe(
      "Status is ReadyToInvoice on the shipment.",
    );
  });

  it("cuts a sentence that runs on at a word, and says it was cut", () => {
    const gist = workNoteGist(`${"word ".repeat(80)}end.`);

    expect(gist.length).toBeLessThanOrEqual(181);
    expect(gist.endsWith("…")).toBe(true);
    expect(gist).not.toMatch(/\s…$/);
  });

  it("is nothing for an empty note", () => {
    expect(workNoteGist("  ")).toBe("");
    expect(workNoteHasMore("  ")).toBe(false);
  });
});
