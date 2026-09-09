import { describe, expect, it } from "vitest";
import {
  IFTA_TAX_RATE_TEMPLATE_CSV,
  iftaTaxRateTemplateFileName,
  parseIftaTaxRateCsv,
} from "../ifta-tax-rate-import";

const jurisdictions = [
  { id: "ij_tx", countryCode: "US", code: "TX" },
  { id: "ij_in", countryCode: "US", code: "IN" },
  { id: "ij_ca_bc", countryCode: "CA", code: "BC" },
  { id: "ij_mx_bc", countryCode: "MX", code: "BC" },
];

const period = { year: 2026, quarter: 2 };

function parse(text: string) {
  return parseIftaTaxRateCsv(text, { jurisdictions, ...period });
}

describe("parseIftaTaxRateCsv", () => {
  it("reads the four columns by name, in any order and casing", () => {
    const result = parse(
      "Fuel Type,Rate Per Gallon,Surcharge,Jurisdiction\nDiesel,0.2000,,TX\nGasoline,0.19,0.0,IN\n",
    );
    expect(result.fileErrors).toEqual([]);
    expect(result.rows).toHaveLength(2);
    expect(result.rows[0]).toMatchObject({
      line: 2,
      jurisdictionCode: "TX",
      fuelType: "Diesel",
      ratePerGallon: "0.2000",
      surchargeRatePerGallon: null,
      error: null,
    });
    expect(result.valid).toEqual([
      {
        jurisdictionId: "ij_tx",
        year: 2026,
        quarter: 2,
        fuelType: "Diesel",
        ratePerGallon: "0.2000",
        surchargeRatePerGallon: null,
      },
      {
        jurisdictionId: "ij_in",
        year: 2026,
        quarter: 2,
        fuelType: "Gasoline",
        ratePerGallon: "0.19",
        surchargeRatePerGallon: "0.0",
      },
    ]);
  });

  it("accepts the common header aliases and quoted cells", () => {
    const result = parse(
      '"State","Fuel","Tax Rate","Surcharge Rate"\n"TX","diesel","0.2","0.01"\n',
    );
    expect(result.fileErrors).toEqual([]);
    expect(result.rows[0]).toMatchObject({
      jurisdictionCode: "TX",
      fuelType: "Diesel",
      ratePerGallon: "0.2",
      surchargeRatePerGallon: "0.01",
      error: null,
    });
  });

  it("refuses the file when a required column is missing", () => {
    const result = parse("Jurisdiction,Fuel Type\nTX,Diesel\n");
    expect(result.fileErrors).toEqual(['The file has no "rate per gallon" column.']);
    expect(result.rows).toEqual([]);
    expect(result.valid).toEqual([]);
  });

  it("refuses an empty file", () => {
    expect(parse("").fileErrors).toEqual(["The file is empty."]);
    expect(parse("Jurisdiction,Fuel Type,Rate\n").fileErrors).toEqual([
      "The file has a header row but no rates.",
    ]);
  });

  it("keeps bad rows in the preview with a reason, and out of the valid set", () => {
    const result = parse(
      [
        "Jurisdiction,Fuel Type,Rate,Surcharge",
        "ZZ,Diesel,0.2,",
        "TX,Kerosene,0.2,",
        "TX,Diesel,abc,",
        "TX,Diesel,-0.1,",
        "TX,Diesel,0.12345,",
        "TX,Diesel,0.2,x",
        "TX,Diesel,,",
        ",Diesel,0.2,",
      ].join("\n"),
    );
    expect(result.rows.map((row) => row.error)).toEqual([
      'Unknown jurisdiction "ZZ".',
      'Unknown fuel type "Kerosene".',
      "Rate must be a number with up to four decimals.",
      "Rate cannot be negative.",
      "Rate must be a number with up to four decimals.",
      "Surcharge must be a number with up to four decimals.",
      "Rate is required.",
      "Jurisdiction is required.",
    ]);
    expect(result.valid).toEqual([]);
  });

  it("flags a jurisdiction and fuel type repeated in the file, keeping the first", () => {
    const result = parse(
      "Jurisdiction,Fuel Type,Rate\nTX,Diesel,0.2\nTX,Diesel,0.25\nTX,Gasoline,0.2\n",
    );
    expect(result.rows.map((row) => row.error)).toEqual([
      null,
      "Duplicate of line 2 (TX, Diesel).",
      null,
    ]);
    expect(result.valid).toHaveLength(2);
  });

  it("resolves a code shared by two countries only with a country prefix", () => {
    const result = parse(
      "Jurisdiction,Fuel Type,Rate\nBC,Diesel,0.2\nCA-BC,Diesel,0.2\nUS-TX,Diesel,0.2\n",
    );
    expect(result.rows.map((row) => row.error)).toEqual([
      'Jurisdiction "BC" exists in more than one country; write it as CA-BC or MX-BC.',
      null,
      null,
    ]);
    expect(result.valid.map((input) => input.jurisdictionId)).toEqual(["ij_ca_bc", "ij_tx"]);
  });

  it("skips blank lines and handles CRLF endings", () => {
    const result = parse("Jurisdiction,Fuel Type,Rate\r\n\r\nTX,Diesel,0.2\r\n\r\n");
    expect(result.rows).toHaveLength(1);
    expect(result.rows[0].line).toBe(3);
  });
});

describe("template", () => {
  it("starts from the header the parser expects", () => {
    expect(IFTA_TAX_RATE_TEMPLATE_CSV.split("\n")[0]).toBe(
      "Jurisdiction,Fuel Type,Rate Per Gallon,Surcharge Per Gallon",
    );
    expect(
      parseIftaTaxRateCsv(IFTA_TAX_RATE_TEMPLATE_CSV, { jurisdictions, ...period }).fileErrors,
    ).toEqual([]);
    expect(iftaTaxRateTemplateFileName(2026, 2)).toBe("ifta-tax-rates-2026Q2.csv");
  });
});
