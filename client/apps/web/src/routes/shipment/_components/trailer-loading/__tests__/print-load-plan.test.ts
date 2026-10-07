import { registerCatalogSource, setLocale } from "@trenova/shared/i18n/runtime";
import type { LoadingOptimizationResult } from "@/types/loading-optimization";
import { afterAll, beforeAll, describe, expect, it } from "vitest";
import { buildLoadPlanBlob, type ShipmentMeta } from "../print-load-plan";

const SPANISH: Record<string, string> = {
  "Load Plan": "Plan de carga",
  "Load Plan — {0}": "Plan de carga — {0}",
  "{0} axle": "Eje {0}",
  PASS: "CORRECTO",
  FAIL: "FALLA",
  "Handling & Notes": "Manejo y <notas>",
  "of {0}ft ({1}%)": "de {0} pies ({1} %)",
  "HAZMAT ({0})": "MATPEL ({0})",
};

const result: LoadingOptimizationResult = {
  trailerLengthFeet: 53,
  totalLinearFeet: 20,
  totalWeight: 12000,
  maxWeight: 45000,
  linearFeetUtil: 37.7,
  weightUtil: 26.6,
  utilizationScore: 60,
  utilizationGrade: "B",
  placements: [
    {
      commodityId: "c1",
      commodityName: "Paint <thinner>",
      positionFeet: 0,
      lengthFeet: 20,
      weight: 12000,
      pieces: 10,
      stackable: false,
      fragile: false,
      isHazmat: true,
      hazmatClass: "3",
      estimatedLength: false,
    },
  ],
  hazmatZones: [],
  warnings: [],
  axleWeights: [
    { axle: "drive", weight: 30000, limit: 34000, percentage: 88, compliant: true },
    { axle: "trailer", weight: 36000, limit: 34000, percentage: 105, compliant: false },
  ],
  recommendations: [],
};

const meta: ShipmentMeta = {
  shipmentId: "shp_1",
  proNumber: "PRO-1",
  bol: "BOL-1",
  customerName: "Acme",
  originName: "Dallas DC",
  originAddress: "Dallas, TX",
  destinationName: "Austin DC",
  destinationAddress: "Austin, TX",
  trailerCode: "TRL-9",
  driverName: "Sam Rivera",
};

async function printed(): Promise<string> {
  return buildLoadPlanBlob(result, meta).text();
}

describe("buildLoadPlanBlob", () => {
  beforeAll(async () => {
    await registerCatalogSource({ es: async () => SPANISH });
    await setLocale("es");
  });

  afterAll(async () => {
    await setLocale("en");
  });

  it("prints the plan in the language the person is using", async () => {
    const html = await printed();

    expect(html).toContain('<html lang="es">');
    expect(html).toContain("<title>Plan de carga — PRO-1</title>");
    expect(html).toContain("<h1>Plan de carga</h1>");
    expect(html).toContain("de 53 pies (38 %)");
    expect(html).toContain("Eje drive");
    expect(html).toContain("CORRECTO");
    expect(html).toContain("FALLA");
    expect(html).toContain("MATPEL (3)");
    expect(html).not.toContain(">Load Plan<");
  });

  it("escapes a translation like any other text placed in the document", async () => {
    const html = await printed();

    expect(html).toContain("<th>Manejo y &lt;notas&gt;</th>");
    expect(html).toContain("Paint &lt;thinner&gt;");
  });
});
