import type { DeskAttentionKind } from "@/lib/graphql/desk-attention";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import {
  NO_FILTERS,
  addressOf,
  attentionFilter,
  filterCount,
  filterTokens,
  filtersFromAddress,
  isNarrowed,
  kindLabel,
  offeredKinds,
  toggled,
  withoutToken,
  type NeedsFilters,
} from "../needs-filters";

const t = ((text: string) => text) as unknown as TranslateFn;

const KINDS: DeskAttentionKind[] = [
  { kind: "PodMissing", label: "POD missing", lens: "Watchtower", count: 40 },
  { kind: "AgentProposal", label: "Agent proposal", lens: "Decisions", count: 12 },
  { kind: "InboundMessage", label: "Inbound message", lens: "Triage", count: 5 },
  { kind: "DetentionOccurrence", label: "Detention", lens: "Watchtower", count: 2 },
  { kind: "Factoring", label: "Factoring", lens: "Watchtower", count: 0 },
];

const filters = (over: Partial<NeedsFilters> = {}): NeedsFilters => ({ ...NO_FILTERS, ...over });

describe("the filters Needs you keeps in its address", () => {
  it("reads the search, the severities in loudness order, the kinds once each and the switches", () => {
    expect(
      filtersFromAddress({
        q: "  SEED-DET  ",
        severities: ["Info", "Critical", "Loud"],
        kinds: ["PodMissing", "PodMissing", "EtaAtRisk"],
        unseen: true,
        working: false,
      }),
    ).toEqual({
      search: "SEED-DET",
      severities: ["Critical", "Info"],
      kinds: ["PodMissing", "EtaAtRisk"],
      unseen: true,
      working: false,
    });
  });

  it("leaves a filter that is off out of the address", () => {
    expect(addressOf(NO_FILTERS)).toEqual({
      q: null,
      severities: null,
      kinds: null,
      unseen: null,
      working: null,
    });
    expect(
      addressOf(
        filters({
          search: " late ",
          severities: ["Warning"],
          kinds: ["PodMissing"],
          working: true,
        }),
      ),
    ).toEqual({
      q: "late",
      severities: ["Warning"],
      kinds: ["PodMissing"],
      unseen: null,
      working: true,
    });
  });

  it("asks the server for the lens, whose cases and every filter", () => {
    expect(
      attentionFilter(
        "Watchtower",
        true,
        filters({ search: "acme", unseen: true, kinds: ["PodMissing"] }),
      ),
    ).toEqual({
      lens: "Watchtower",
      mine: true,
      kinds: ["PodMissing"],
      severities: [],
      search: "acme",
      unseen: true,
      working: false,
    });
  });

  it("counts every filter but the search, which has its own box", () => {
    const some = filters({
      search: "acme",
      severities: ["Critical", "Warning"],
      kinds: ["PodMissing"],
      unseen: true,
    });

    expect(filterCount(some)).toBe(4);
    expect(filterCount(filters({ search: "acme" }))).toBe(0);
    expect(isNarrowed(filters({ search: "acme" }))).toBe(true);
    expect(isNarrowed(filters({ working: true }))).toBe(true);
    expect(isNarrowed(NO_FILTERS)).toBe(false);
  });

  it("adds a value a list lacks and takes out one it has", () => {
    expect(toggled(["Critical"], "Info")).toEqual(["Critical", "Info"]);
    expect(toggled(["Critical", "Info"], "Critical")).toEqual(["Info"]);
  });
});

describe("the kinds the filter offers", () => {
  it("offers every kind with something open under Everything, most first", () => {
    expect(offeredKinds(KINDS, "Everything", []).map((entry) => entry.kind)).toEqual([
      "PodMissing",
      "AgentProposal",
      "InboundMessage",
      "DetentionOccurrence",
    ]);
  });

  it("offers only the lens's own kinds under a lens", () => {
    expect(offeredKinds(KINDS, "Watchtower", []).map((entry) => entry.kind)).toEqual([
      "PodMissing",
      "DetentionOccurrence",
    ]);
  });

  it("keeps a chosen kind on offer with nothing open or from another lens, so it can be taken off", () => {
    expect(
      offeredKinds(KINDS, "Watchtower", ["Factoring", "AgentProposal"]).map((entry) => entry.kind),
    ).toEqual(["PodMissing", "AgentProposal", "DetentionOccurrence", "Factoring"]);
  });

  it("names a kind as the server does, and spells out its code until the server answers", () => {
    expect(kindLabel(KINDS, "PodMissing")).toBe("POD missing");
    expect(kindLabel([], "MissedPickup")).toBe("Missed pickup");
  });
});

describe("the tokens under the search", () => {
  const on = filters({
    search: "acme",
    severities: ["Critical", "Info"],
    kinds: ["DetentionOccurrence"],
    unseen: true,
    working: true,
  });

  it("draws one per filter in the menu's order and none for the search", () => {
    expect(filterTokens(t, on, KINDS).map((token) => [token.key, token.label])).toEqual([
      ["severity:Critical", "Critical"],
      ["severity:Info", "Info"],
      ["unseen", "Not opened yet"],
      ["working", "An agent is working"],
      ["kind:DetentionOccurrence", "Detention"],
    ]);
    expect(filterTokens(t, filters({ search: "acme" }), KINDS)).toEqual([]);
  });

  it("takes off only the filter its token stands for", () => {
    const [critical, , unseen, working, detention] = filterTokens(t, on, KINDS);

    expect(withoutToken(on, critical)).toEqual({ ...on, severities: ["Info"] });
    expect(withoutToken(on, unseen)).toEqual({ ...on, unseen: false });
    expect(withoutToken(on, working)).toEqual({ ...on, working: false });
    expect(withoutToken(on, detention)).toEqual({ ...on, kinds: [] });
  });
});
