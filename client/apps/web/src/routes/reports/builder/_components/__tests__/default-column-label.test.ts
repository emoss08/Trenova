import type { ReportCatalog } from "@/lib/graphql/reports";
import type { ReportColumnSpec, ReportIR } from "@/types/report";
import { registerCatalogSource, setLocale } from "@trenova/shared/i18n/runtime";
import { afterEach, beforeAll, describe, expect, it } from "vitest";
import { buildCatalogIndex, defaultColumnLabel } from "../builder-state";

const index = buildCatalogIndex({
  entities: [
    {
      key: "shipment",
      label: "Envío",
      pluralLabel: "Envíos",
      fields: [
        { key: "id", label: "ID", type: "string" },
        { key: "totalCharge", label: "Total Charge", type: "decimal" },
        { key: "deliveredAt", label: "Entregado", type: "epoch" },
        { key: "grossAmount", label: "Importe total", type: "decimal" },
        { key: "weight", label: "Peso", type: "decimal" },
      ],
      edges: [],
    },
  ],
} as unknown as ReportCatalog);

const ir = { entity: "shipment" } as ReportIR;

function measure(field: string, agg: ReportColumnSpec["agg"]): ReportColumnSpec {
  return { id: field, kind: "measure", agg, ref: { field } } as ReportColumnSpec;
}

describe("defaultColumnLabel", () => {
  beforeAll(async () => {
    await registerCatalogSource({
      es: async () => ({
        "Total {0}": "{0} total",
        "Latest {0}": "Último {0}",
        "{0} Count": "Cantidad de {0}",
        "{0} (Month)": "{0} (mes)",
        Calculation: "Cálculo",
      }),
    });
  });

  afterEach(async () => {
    await setLocale("en");
  });

  it("reads in English with the English catalog", () => {
    expect(defaultColumnLabel(index, ir, measure("deliveredAt", "max"))).toBe("Latest Entregado");
    expect(defaultColumnLabel(index, ir, measure("totalCharge", "sum"))).toBe("Total Charge");
  });

  it("composes the header from the language on screen", async () => {
    await setLocale("es");

    expect(defaultColumnLabel(index, ir, measure("deliveredAt", "max"))).toBe("Último Entregado");
    expect(defaultColumnLabel(index, ir, measure("id", "count"))).toBe("Cantidad de Envíos");
    expect(
      defaultColumnLabel(index, ir, {
        id: "delivered",
        kind: "dimension",
        bucket: "month",
        ref: { field: "deliveredAt" },
      } as ReportColumnSpec),
    ).toBe("Entregado (mes)");
    expect(defaultColumnLabel(index, ir, { id: "c", kind: "computed" } as ReportColumnSpec)).toBe(
      "Cálculo",
    );
  });

  it("skips the aggregation's words when the label already carries them", async () => {
    await setLocale("es");

    expect(defaultColumnLabel(index, ir, measure("weight", "sum"))).toBe("Peso total");
    expect(defaultColumnLabel(index, ir, measure("grossAmount", "sum"))).toBe("Importe total");
    expect(defaultColumnLabel(index, ir, measure("totalCharge", "sum"))).toBe("Total Charge");
  });
});
