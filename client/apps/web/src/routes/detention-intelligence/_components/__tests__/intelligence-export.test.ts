import type { DetentionIntelligence } from "@/lib/graphql/detention-intelligence";
import { describe, expect, it } from "vitest";
import { csvCell, intelligenceCsv } from "../intelligence-export";

describe("csvCell", () => {
  it("opens a cell a spreadsheet would run as a formula with a quote", () => {
    expect(csvCell('=HYPERLINK("http://x")')).toBe('"\'=HYPERLINK(""http://x"")"');
    expect(csvCell("+1 Dock")).toBe("'+1 Dock");
    expect(csvCell("-Ridgeway")).toBe("'-Ridgeway");
    expect(csvCell("@SUM(A1)")).toBe("'@SUM(A1)");
  });

  it("quotes a cell holding a comma, a quote or a line break, and leaves the rest as they are", () => {
    expect(csvCell("Ridgeway, DC")).toBe('"Ridgeway, DC"');
    expect(csvCell('12" dock')).toBe('"12"" dock"');
    expect(csvCell("Ridgeway DC")).toBe("Ridgeway DC");
    expect(csvCell(42)).toBe("42");
  });
});

const t = ((message: string) => message) as Parameters<typeof intelligenceCsv>[1];

function page(netMargin: string | null): DetentionIntelligence {
  return {
    totals: { netMargin },
    facilities: [
      {
        locationName: "=Ridgeway DC",
        locationCity: "Joliet",
        locationState: "IL",
        stopCount: 80,
        detainedCount: 50,
        pastFreeMinutes: 300,
        dockMinutes: 210,
        openMinutes: 30,
        earlyMinutes: 30,
        lateMinutes: 30,
        medianDwellMinutes: 188.4,
        p90DwellMinutes: 412,
        medianFreeMinutes: 120,
        billedAmount: "7000.5",
        driverPayAmount: netMargin === null ? null : "3000",
        netMargin: netMargin === null ? null : "4000.5",
        waivedAmount: "400",
        noticeDueCount: 8,
        noticeOnTimeCount: 6,
      },
    ],
  } as unknown as DetentionIntelligence;
}

describe("intelligenceCsv", () => {
  it("writes each facility as figures a spreadsheet can sum", () => {
    const [header, row] = intelligenceCsv(page("4000.5"), t).split("\n");

    expect(header).toBe(
      'Facility,Stops,Stops detained,Hours past free time,"On time, the dock was slow",' +
        "No appointment,Arrived early,Arrived late,Median stop,Slowest one in ten," +
        "Free time granted,Billed,Driver pay,Net,Forgiven,Notice on time",
    );
    expect(row).toBe(
      "'=Ridgeway DC,80,0.625,5,3.5,0.5,0.5,0.5,188,412,120,7000.5,3000,4000.5,400,0.75",
    );
  });

  it("leaves out driver pay and net for a reader without margins", () => {
    const [header, row] = intelligenceCsv(page(null), t).split("\n");

    expect(header).not.toContain("Driver pay");
    expect(header).not.toContain("Net");
    expect(row).toBe("'=Ridgeway DC,80,0.625,5,3.5,0.5,0.5,0.5,188,412,120,7000.5,400,0.75");
  });
});
