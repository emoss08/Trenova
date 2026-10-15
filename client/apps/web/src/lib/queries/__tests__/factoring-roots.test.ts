import { factoring } from "@/lib/queries/factoring";
import { factoringNoa } from "@/lib/queries/factoring-noa";
import { factoringRemittance } from "@/lib/queries/factoring-remittance";
import {
  FACTORING_INVOICE_STATE_KEY,
  FACTORING_REMITTANCE_KEY,
  FACTORING_SUBMISSION_KEY,
  invalidateFactoringQueries,
} from "@/lib/queries/factoring-roots";
import { QueryClient, type QueryKey } from "@tanstack/react-query";
import { describe, expect, it } from "vitest";

describe("invalidateFactoringQueries", () => {
  it("stales every factoring cache, whichever factory or table holds it, and nothing else", () => {
    const client = new QueryClient();
    const factoringKeys: QueryKey[] = [
      factoring.summary().queryKey,
      factoring.submission("facsub_1").queryKey,
      factoring.reserveBalance("fac_rts").queryKey,
      factoringNoa.list({ customerId: "cus_1", factoringCompanyId: "fac_rts" }).queryKey,
      factoringNoa.owner("facnoa_1").queryKey,
      factoringRemittance.list({ inboundMessageId: "inmsg_1" }).queryKey,
      factoringRemittance.detail("facrem_1").queryKey,
      factoringRemittance.lines("facrem_1", null).queryKey,
      [FACTORING_INVOICE_STATE_KEY, { pageIndex: 1 }],
      [FACTORING_SUBMISSION_KEY, { pageIndex: 1 }],
      [FACTORING_REMITTANCE_KEY, { pageIndex: 1 }],
    ];
    const unrelated: QueryKey = ["invoice-list", { pageIndex: 1 }];
    for (const key of [...factoringKeys, unrelated]) {
      client.setQueryData(key, { cached: true });
    }

    invalidateFactoringQueries(client);

    expect(factoringKeys.filter((key) => !client.getQueryState(key)?.isInvalidated)).toEqual([]);
    expect(client.getQueryState(unrelated)?.isInvalidated).toBe(false);
  });
});
