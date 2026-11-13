import type { DeskAttention, DeskAttentionItem } from "@/lib/graphql/desk-attention";
import { patchAttentionItem, removeAttentionItem } from "@/lib/desk-attention-cache";
import { deskAttention } from "@/lib/queries/desk-attention";
import { QueryClient } from "@tanstack/react-query";
import { describe, expect, it } from "vitest";

function item(id: string, seen = true): DeskAttentionItem {
  return { id, seen, deskLens: "Watchtower", aboutId: "shp_1" } as DeskAttentionItem;
}

function attention(): DeskAttention {
  return {
    groups: [
      {
        key: "shp_1",
        record: null,
        severity: "Critical",
        dueAt: 100,
        dueLabel: null,
        unseen: true,
        items: [item("wt_1", false), item("wt_2")],
        ownership: null,
      },
      {
        key: "wt_3",
        record: null,
        severity: "Info",
        dueAt: null,
        dueLabel: null,
        unseen: false,
        items: [item("wt_3")],
        ownership: null,
      },
    ],
    counts: { everything: 3, watchtower: 3, triage: 0, decisions: 0 },
    hasMore: false,
    seenAt: 0,
    seesEveryone: false,
  };
}

function seeded() {
  const client = new QueryClient();
  const everything = deskAttention.list().queryKey;
  const oneRecord = deskAttention.list({ recordId: "shp_1" }).queryKey;
  client.setQueryData(everything, attention());
  client.setQueryData(oneRecord, attention());
  return { client, everything, oneRecord };
}

describe("the Needs you cache", () => {
  it("takes an item out of every cached list, keeping its row while others remain", () => {
    const { client, everything, oneRecord } = seeded();

    removeAttentionItem(client, "wt_1");

    for (const key of [everything, oneRecord]) {
      const data = client.getQueryData<DeskAttention>(key);
      expect(data?.groups.map((group) => group.items.map((entry) => entry.id))).toEqual([
        ["wt_2"],
        ["wt_3"],
      ]);
    }
  });

  it("drops a row when its last item leaves", () => {
    const { client, everything } = seeded();

    removeAttentionItem(client, "wt_3");

    expect(
      client.getQueryData<DeskAttention>(everything)?.groups.map((group) => group.key),
    ).toEqual(["shp_1"]);
  });

  it("marks a row seen once the item that was new is opened", () => {
    const { client, everything } = seeded();

    patchAttentionItem(client, "wt_1", (entry) => ({ ...entry, seen: true }));

    const row = client.getQueryData<DeskAttention>(everything)?.groups[0];
    expect(row?.unseen).toBe(false);
    expect(row?.items[0].seen).toBe(true);
  });
});
