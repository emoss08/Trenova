import { matchFields } from "@trenova/shared/lib/fuzzy-score";
import { describe, expect, it } from "vitest";
import { samePaletteItem, type PaletteItem } from "../palette-model";
import { command, page, record } from "./palette-test-fixtures";

const PAGE = page();
const RECORD = record();
const COMMAND = command();

function titleMatch(query: string, title: string) {
  return matchFields(query, [{ key: "title", text: title }]) ?? undefined;
}

describe("samePaletteItem", () => {
  it("treats a row rebuilt for the next key as the same when it draws the same thing", () => {
    const before: PaletteItem = {
      kind: "page",
      key: "page:/billing/invoices",
      page: PAGE,
      match: titleMatch("inv", PAGE.title),
    };
    const after: PaletteItem = {
      kind: "page",
      key: "page:/billing/invoices",
      page: PAGE,
      match: { ...titleMatch("inv", PAGE.title)!, score: 0.2 },
    };

    expect(samePaletteItem(before, after)).toBe(true);
  });

  it("redraws a row whose marked characters moved", () => {
    const before: PaletteItem = {
      kind: "command",
      key: "command:quick:create-shipment",
      command: COMMAND,
      match: titleMatch("ne", COMMAND.label),
    };
    const after: PaletteItem = { ...before, match: titleMatch("new", COMMAND.label) };

    expect(samePaletteItem(before, after)).toBe(false);
  });

  it("redraws a record row handed another copy of the record, which may hold newer fields", () => {
    const before: PaletteItem = { kind: "record", key: "record:shipment:shp_1", record: RECORD };
    const after: PaletteItem = {
      kind: "record",
      key: "record:shipment:shp_1",
      record: { ...RECORD, metadata: { status: "Delivered" } },
    };

    expect(samePaletteItem(before, after)).toBe(false);
    expect(samePaletteItem(before, { ...before })).toBe(true);
  });

  it("redraws a recent record when it was opened again", () => {
    const before: PaletteItem = {
      kind: "record",
      key: "recent-record:shipment:shp_1",
      record: RECORD,
      recent: true,
      visitedAt: 100,
    };

    expect(samePaletteItem(before, { ...before, visitedAt: 200 })).toBe(false);
  });

  it("compares a question by its words and the record it is about", () => {
    const subject = { type: "shipment", id: "shp_1", label: "PRO-1001" } as const;
    const ask: PaletteItem = {
      kind: "ask",
      key: "ask:shipment:shp_1:Where is it?",
      question: "Where is it?",
      subject,
    };

    expect(samePaletteItem(ask, { ...ask, subject: { ...subject } })).toBe(true);
    expect(samePaletteItem(ask, { ...ask, subject: { ...subject, id: "shp_2" } })).toBe(false);
  });

  it("never treats rows with different keys as the same", () => {
    const a: PaletteItem = { kind: "widen", key: "widen:all", query: "acme" };
    const b: PaletteItem = { kind: "widen", key: "widen:other", query: "acme" };

    expect(samePaletteItem(a, b)).toBe(false);
    expect(samePaletteItem(a, { ...a, query: "acme co" })).toBe(false);
  });
});
