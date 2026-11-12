import { watchtower } from "@/lib/queries/watchtower";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";

export type LensCounts = { everything: number; watchtower: number; decisions: number };

/** Open items in each lens of Needs you, from the Watchtower's counts. */
export function useLensCounts(enabled: boolean): LensCounts {
  const { data } = useQuery({ ...watchtower.counts(), enabled });

  return useMemo(() => {
    const counts = { everything: 0, watchtower: 0, decisions: 0 };
    for (const row of data?.byKind ?? []) {
      counts.everything += row.count;
      if (row.deskLens === "Watchtower") {
        counts.watchtower += row.count;
      } else if (row.deskLens === "Decisions") {
        counts.decisions += row.count;
      }
    }
    return counts;
  }, [data]);
}
