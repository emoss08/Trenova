import { describe, expect, it } from "vitest";
import {
  MODEL_PAGE_SIZE,
  initialModelView,
  modelPage,
  searchModels,
  splitModels,
  type ModelLike,
} from "../model-list";

const DAY = 86_400;
const NEWEST = 1_780_000_000;

function model(id: string, createdAt?: number | null, displayName = id): ModelLike {
  return { id, displayName, createdAt };
}

function ids(models: readonly ModelLike[]): string[] {
  return models.map((entry) => entry.id);
}

describe("splitModels", () => {
  it("puts models within a year of the newest under latest, newest first, and the rest under older", () => {
    const result = splitModels([
      model("old", NEWEST - 400 * DAY),
      model("newest", NEWEST),
      model("edge", NEWEST - 365 * DAY),
      model("recent", NEWEST - 30 * DAY),
      model("past-edge", NEWEST - 365 * DAY - 1),
    ]);

    expect(result.split).toBe(true);
    expect(ids(result.latest)).toEqual(["newest", "recent", "edge"]);
    expect(ids(result.older)).toEqual(["past-edge", "old"]);
  });

  it("measures latest from the list's newest model, not from today", () => {
    const longAgo = 1_500_000_000;
    const result = splitModels([model("a", longAgo), model("b", longAgo - 500 * DAY)]);

    expect(ids(result.latest)).toEqual(["a"]);
    expect(ids(result.older)).toEqual(["b"]);
  });

  it("files undated models under older, by id, after the dated ones", () => {
    const result = splitModels([
      model("zeta", null),
      model("dated-old", NEWEST - 800 * DAY),
      model("alpha"),
      model("dated", NEWEST),
    ]);

    expect(ids(result.latest)).toEqual(["dated"]);
    expect(ids(result.older)).toEqual(["dated-old", "alpha", "zeta"]);
  });

  it("does not split a list in which no model is dated", () => {
    const result = splitModels([model("b"), model("a", null), model("c")]);

    expect(result.split).toBe(false);
    expect(ids(result.latest)).toEqual(["a", "b", "c"]);
    expect(result.older).toEqual([]);
  });

  it("does not split an empty list", () => {
    expect(splitModels([])).toEqual({ split: false, latest: [], older: [] });
  });
});

describe("searchModels", () => {
  const models = [
    model("claude-opus-4-1", NEWEST, "Claude Opus 4.1"),
    model("gpt-5", NEWEST, "GPT-5"),
    model("text-embedding-3", NEWEST, "Embeddings"),
  ];

  it("matches the id or the display name, ignoring case and surrounding space", () => {
    expect(ids(searchModels(models, "  OPUS "))).toEqual(["claude-opus-4-1"]);
    expect(ids(searchModels(models, "embeddings"))).toEqual(["text-embedding-3"]);
    expect(ids(searchModels(models, "gpt-5"))).toEqual(["gpt-5"]);
  });

  it("keeps every model, in order, for an empty search", () => {
    expect(ids(searchModels(models, "  "))).toEqual(ids(models));
  });

  it("narrows a list that spans pages down to one page", () => {
    const many = Array.from({ length: 25 }, (_, index) =>
      model(index === 17 ? "llama-special" : `model-${String(index).padStart(2, "0")}`),
    );
    const found = searchModels(many, "special");

    expect(ids(found)).toEqual(["llama-special"]);
    expect(modelPage(found, 2)).toMatchObject({ page: 0, pageCount: 1, total: 1 });
  });
});

describe("modelPage", () => {
  const many = Array.from({ length: 23 }, (_, index) => model(`m${index}`));

  it("serves ten at a time and says which ones", () => {
    const second = modelPage(many, 1);

    expect(MODEL_PAGE_SIZE).toBe(10);
    expect(ids(second.items)).toEqual(ids(many.slice(10, 20)));
    expect(second).toMatchObject({ page: 1, pageCount: 3, from: 11, to: 20, total: 23 });
    expect(modelPage(many, 2)).toMatchObject({ from: 21, to: 23 });
  });

  it("clamps a page past the end to the last page, and one before the start to the first", () => {
    expect(modelPage(many, 9).page).toBe(2);
    expect(modelPage(many.slice(0, 4), 2)).toMatchObject({ page: 0, pageCount: 1, from: 1, to: 4 });
    expect(modelPage(many, -3).page).toBe(0);
  });

  it("is one empty page for an empty list", () => {
    expect(modelPage([], 4)).toEqual({
      items: [],
      page: 0,
      pageCount: 1,
      from: 0,
      to: 0,
      total: 0,
    });
  });
});

describe("initialModelView", () => {
  const latest = [model("new-a", NEWEST), model("new-b", NEWEST - DAY)];
  const older = Array.from({ length: 14 }, (_, index) =>
    model(`old-${String(index).padStart(2, "0")}`, NEWEST - (400 + index) * DAY),
  );
  const split = () => splitModels([...older, ...latest]);

  it("opens older on the page holding a selected older model", () => {
    expect(initialModelView(split(), "old-12")).toEqual({ group: "older", page: 1 });
  });

  it("opens latest on its page for a selected latest model", () => {
    expect(initialModelView(split(), "new-b")).toEqual({ group: "latest", page: 0 });
  });

  it("opens the first page of latest when nothing listed is selected", () => {
    expect(initialModelView(split(), "")).toEqual({ group: "latest", page: 0 });
    expect(initialModelView(split(), "unlisted")).toEqual({ group: "latest", page: 0 });
  });

  it("opens latest when the list is not split", () => {
    const flat = splitModels(Array.from({ length: 12 }, (_, index) => model(`x${index + 10}`)));
    expect(initialModelView(flat, "x21")).toEqual({ group: "latest", page: 1 });
  });

  it("opens older for a selected undated model", () => {
    const onlyOld = splitModels([model("dated", NEWEST), model("undated")]);
    expect(initialModelView(onlyOld, "undated")).toEqual({ group: "older", page: 0 });
  });
});
