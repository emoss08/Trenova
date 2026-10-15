import { describe, expect, it } from "vitest";
import { withoutRecordIds } from "../record-ids";

describe("withoutRecordIds", () => {
  it("drops an id in an aside, with its label", () => {
    expect(
      withoutRecordIds(
        "Approval completed: **SEED-PAY-009** created invoice **INV2610000011** (ID **inv_01M42BNHACKS99T13QKY88TVXV**).",
      ),
    ).toBe("Approval completed: **SEED-PAY-009** created invoice **INV2610000011**.");
    expect(
      withoutRecordIds(
        "It is **INV2610000011**, `inv_01M42BNHACKS99T13QKY88TVXV`. Its status is Draft.",
      ),
    ).toBe("It is **INV2610000011**. Its status is Draft.");
  });

  it("drops an id named in a sentence", () => {
    expect(withoutRecordIds("Approve proposal **ap_01M42BMTM3FGMGES7JN5715W3K** to proceed.")).toBe(
      "Approve proposal to proceed.",
    );
    expect(
      withoutRecordIds(
        "- **SEED-PAY-009** — shipment `shp_01M3Q2Y4SRFE0YW60JY6F5NRW7`; Peak Distributing; **InReview**",
      ),
    ).toBe("- **SEED-PAY-009** — shipment; Peak Distributing; **InReview**");
    expect(
      withoutRecordIds(
        "The uncovered move was on **SEED-SHP-006** (shipment ID **shp_01M3Q2Y3JYTGFNHAJX8H1DB6JH**).",
      ),
    ).toBe("The uncovered move was on **SEED-SHP-006**.");
  });

  it("leaves artifact links, web addresses and ordinary words alone", () => {
    const text =
      "See [Billing queue items](artifact:art_01M42AYFPF05FCCQ4R5R1Y7EQJ) and https://x.test/r/inv_01M42BNHACKS99T13QKY88TVXV for SEED_PAY_009.";
    expect(withoutRecordIds(text)).toBe(text);
    expect(withoutRecordIds("No ids here, just the BOL-2026-0109.")).toBe(
      "No ids here, just the BOL-2026-0109.",
    );
  });

  it("drops a sentence that only names an id", () => {
    expect(
      withoutRecordIds(
        "The partner was saved, as recorded in [EDI partner Acme Retail Group](artifact:art_01M504P2QDKXPBM3QCTD7C0HWR). Its ID is edip_01M504P2M5BKZJ2EK36C1T4XKA. Nothing is live yet.",
      ),
    ).toBe(
      "The partner was saved, as recorded in [EDI partner Acme Retail Group](artifact:art_01M504P2QDKXPBM3QCTD7C0HWR). Nothing is live yet.",
    );
    expect(
      withoutRecordIds(
        "Acme Retail Group's ID: `edip_01M504P2M5BKZJ2EK36C1T4XKA`.\nNothing is live yet.",
      ),
    ).toBe("Nothing is live yet.");
    expect(
      withoutRecordIds("The partner is saved, and its ID is **edip_01M504P2M5BKZJ2EK36C1T4XKA**."),
    ).toBe("The partner is saved.");
    expect(withoutRecordIds("Saved. The partner ID was edip_01M504P2M5BKZJ2EK36C1T4XKA")).toBe(
      "Saved.",
    );
  });

  it("keeps a sentence that says more than the id", () => {
    expect(withoutRecordIds("Its ID is edip_01M504P2M5BKZJ2EK36C1T4XKA the one Acme sent.")).toBe(
      "Its ID is the one Acme sent.",
    );
  });
});
