import type { CaptureBatchDetail } from "@/lib/graphql/capture";
import { describe, expect, it } from "vitest";
import {
  LOOSE,
  dropPosition,
  isDirty,
  layoutFromBatch,
  leaveOut,
  mergeWithNext,
  movePage,
  newDocumentFrom,
  rotate,
  splitAfter,
  toEditInput,
} from "../page-layout";

type Item = CaptureBatchDetail["items"][number];
type Page = CaptureBatchDetail["pages"][number];

function page(id: string, sequence: number, rotation = 0): Page {
  return {
    id,
    sequence,
    status: "Processed",
    rotation,
    widthPx: 0,
    heightPx: 0,
    dpi: 300,
    isBlank: false,
    isSeparator: false,
    patchCode: "",
    isCoverSheet: false,
    unrecognizedCoverSheet: false,
    failureMessage: "",
    contentPath: "",
    thumbnailPath: "",
  };
}

function item(id: string, position: number, status: Item["status"], pageIds: string[]): Item {
  return {
    id,
    position,
    status,
    pageIds,
    pageCount: pageIds.length,
    suggestedType: "",
    suggestedId: null,
    suggestedDocumentTypeId: null,
    suggestionSource: null,
    suggestionConfidence: null,
    suggestionReason: "",
    detectedKind: "",
    filedType: "",
    filedId: null,
    filedDocumentTypeId: null,
    documentId: null,
    filedAt: null,
    failureMessage: "",
    version: 1,
    suggestedRecord: null,
    filedRecord: null,
  };
}

/*
 * A stack of seven: a filed invoice (p1), a patch sheet the splitter set aside
 * (p2), a two-page POD (p3, p4), a discarded page (p5) and a failed filing
 * (p6, p7) that can be tried again. Items arrive out of position order, as
 * nothing promises they are sorted.
 */
function batch(): CaptureBatchDetail {
  return {
    pages: [
      page("p7", 7),
      page("p1", 1),
      page("p2", 2),
      page("p3", 3, 450),
      page("p4", 4),
      page("p5", 5),
      page("p6", 6),
    ],
    items: [
      item("i3", 3, "Failed", ["p6", "p7"]),
      item("i1", 1, "Filed", ["p1"]),
      item("i2", 2, "Proposed", ["p3", "p4"]),
      item("i9", 9, "Discarded", ["p5"]),
    ],
  } as CaptureBatchDetail;
}

describe("layoutFromBatch", () => {
  it("edits only open documents, in position order", () => {
    const layout = layoutFromBatch(batch());
    expect(layout.groups).toEqual([
      { key: "i2", pageIds: ["p3", "p4"] },
      { key: "i3", pageIds: ["p6", "p7"] },
    ]);
  });

  it("sets aside pages in no document, including a discarded one, but never a filed one", () => {
    expect(layoutFromBatch(batch()).loose).toEqual(["p2", "p5"]);
  });

  it("reads rotations as quarter turns within one revolution", () => {
    expect(layoutFromBatch(batch()).rotations.p3).toBe(90);
  });
});

describe("editing", () => {
  it("splits after a page and keeps the route key on the first half", () => {
    const layout = splitAfter(layoutFromBatch(batch()), "i2", "p3");
    expect(layout.groups.map((group) => group.pageIds)).toEqual([["p3"], ["p4"], ["p6", "p7"]]);
    expect(layout.groups[0]!.key).toBe("i2");
    expect(layout.groups[1]!.key).not.toBe("i2");
  });

  it("does nothing when splitting after a document's last page", () => {
    const layout = layoutFromBatch(batch());
    expect(splitAfter(layout, "i2", "p4")).toEqual(layout);
  });

  it("merges a document with the next and not past the end", () => {
    const layout = layoutFromBatch(batch());
    expect(mergeWithNext(layout, "i2").groups).toEqual([
      { key: "i2", pageIds: ["p3", "p4", "p6", "p7"] },
    ]);
    expect(mergeWithNext(layout, "i3")).toEqual(layout);
  });

  it("moves a page between documents and drops a document left empty", () => {
    let layout = movePage(layoutFromBatch(batch()), "p6", "i2", 1);
    expect(layout.groups).toEqual([
      { key: "i2", pageIds: ["p3", "p6", "p4"] },
      { key: "i3", pageIds: ["p7"] },
    ]);

    layout = movePage(layout, "p7", "i2", 99);
    expect(layout.groups).toEqual([{ key: "i2", pageIds: ["p3", "p6", "p4", "p7"] }]);
  });

  it("puts a page set aside back in scan order", () => {
    const layout = leaveOut(layoutFromBatch(batch()), "p4");
    expect(layout.loose).toEqual(["p2", "p4", "p5"]);
    expect(layout.groups[0]!.pageIds).toEqual(["p3"]);
  });

  it("refuses to move a filed page or into a document that is not there", () => {
    const layout = layoutFromBatch(batch());
    expect(movePage(layout, "p1", "i2", 0)).toEqual(layout);
    expect(movePage(layout, "p3", "nope", 0)).toEqual(layout);
  });

  it("brings a set-aside page back as a document of its own", () => {
    const layout = newDocumentFrom(layoutFromBatch(batch()), "p2");
    expect(layout.loose).toEqual(["p5"]);
    expect(layout.groups.at(-1)).toEqual({ key: "new:p2", pageIds: ["p2"] });
    expect(newDocumentFrom(layout, "p3")).toEqual(layout);
  });

  it("moves a set-aside page into a document", () => {
    const layout = movePage(layoutFromBatch(batch()), "p2", "i2", 0);
    expect(layout.groups[0]!.pageIds).toEqual(["p2", "p3", "p4"]);
    expect(layout.loose).toEqual(["p5"]);
    expect(movePage(layout, "p2", LOOSE, 0).loose).toEqual(["p2", "p5"]);
  });

  it("rotates in quarter turns both ways", () => {
    let layout = rotate(layoutFromBatch(batch()), "p4", 1);
    expect(layout.rotations.p4).toBe(90);
    layout = rotate(layout, "p4", -2);
    expect(layout.rotations.p4).toBe(270);
    expect(rotate(layout, "unknown", 1)).toBe(layout);
  });
});

describe("saving", () => {
  it("is clean until something changes, and clean again when it is undone", () => {
    const saved = layoutFromBatch(batch());
    expect(isDirty(saved, saved)).toBe(false);

    const turned = rotate(saved, "p4", 1);
    expect(isDirty(turned, saved)).toBe(true);
    expect(isDirty(rotate(turned, "p4", -1), saved)).toBe(false);

    expect(isDirty(movePage(saved, "p4", "i3", 0), saved)).toBe(true);
  });

  it("sends open documents in order and only the rotations that changed", () => {
    const saved = layoutFromBatch(batch());
    const edited = rotate(leaveOut(saved, "p7"), "p6", 1);

    expect(toEditInput(edited, saved, 12)).toEqual({
      version: 12,
      items: [{ pageIds: ["p3", "p4"] }, { pageIds: ["p6"] }],
      rotations: [{ pageId: "p6", rotation: 90 }],
    });
  });
});

describe("dropPosition", () => {
  const layout = layoutFromBatch(batch());

  it("takes the place of the page it was dropped on", () => {
    expect(dropPosition(layout, "p4")).toEqual({ target: "i2", index: 1 });
    expect(movePage(layout, "p7", "i2", 1).groups[0]!.pageIds).toEqual(["p3", "p7", "p4"]);
  });

  it("goes to the end of a document dropped on its empty space", () => {
    expect(dropPosition(layout, "i3")).toEqual({ target: "i3", index: 2 });
  });

  it("sets the page aside when dropped among the pages set aside", () => {
    expect(dropPosition(layout, LOOSE)).toEqual({ target: LOOSE, index: 0 });
    expect(dropPosition(layout, "p5")).toEqual({ target: LOOSE, index: 0 });
  });

  it("goes nowhere when dropped on a filed page", () => {
    expect(dropPosition(layout, "p1")).toBeNull();
  });
});
