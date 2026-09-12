import { getColumns } from "@/routes/shipment/_components/shipment-columns";
import { describe, expect, it } from "vitest";

describe("shipment billing column", () => {
  const columns = getColumns([]);
  const column = columns.find((entry) => entry.id === "billing");

  it("sits right after the tender column", () => {
    const ids = columns.map((entry) => entry.id);

    expect(ids.indexOf("billing")).toBe(ids.indexOf("tenderStatus") + 1);
  });

  it("filters on the shipment's billing transfer status, Posted included", () => {
    expect(column?.header).toBe("Billing");
    expect(column?.meta?.apiField).toBe("billingTransferStatus");
    expect(column?.meta?.filterType).toBe("select");
    expect(column?.meta?.filterOptions?.map((option) => option.value)).toEqual(
      expect.arrayContaining([
        "ReadyForReview",
        "InReview",
        "Approved",
        "Posted",
        "OnHold",
        "SentBackToOps",
        "Exception",
        "Canceled",
      ]),
    );
  });
});
