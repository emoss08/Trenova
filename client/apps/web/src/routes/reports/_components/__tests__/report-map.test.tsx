import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import type { ReportChartSpec } from "@/types/report";
import { describe, expect, it, vi } from "vitest";
import { ReportMap } from "../report-map";

vi.mock("@vis.gl/react-google-maps", () => ({
  APIProvider: ({ children }: { children: ReactNode }) => <>{children}</>,
  Map: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

vi.mock("@/components/map/advanced-marker", () => ({
  AdvancedMarker: ({ title }: { title?: string }) => <div data-testid="pin" title={title} />,
}));

vi.mock("@/hooks/use-map-style", () => ({
  useMapStyle: () => ({ mapId: "map", colorScheme: "LIGHT" }),
}));

vi.mock("@tanstack/react-query", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@tanstack/react-query")>()),
  useQuery: () => ({ data: { config: { apiKey: "key" } } }),
}));

const TEXT = {
  style: "text",
  decimals: 0,
  grouping: true,
  currency: "USD",
  negative: "minus",
  notation: "standard",
  prefix: "",
  suffix: "",
  dateStyle: "dateTime",
  boolStyle: "yesNo",
  durationUnit: "seconds",
  durationStyle: "clock",
  nullText: "",
  rules: [],
  band: null,
};

const CHART = {
  id: "map",
  type: "map",
  latColumnId: "lat",
  lngColumnId: "lng",
  labelColumnId: "status",
} as unknown as ReportChartSpec;

describe("ReportMap", () => {
  // GraphQL sends an enum column's labels on its display; a pin titled by that column names
  // the label, as the grid does, never the enum's code.
  it("titles each pin with the label column's cell as the grid writes it", async () => {
    render(
      <ReportMap
        chart={CHART}
        columns={[
          { id: "lat", label: "Latitude", type: "decimal" },
          { id: "lng", label: "Longitude", type: "decimal" },
          {
            id: "status",
            label: "Status",
            type: "enum",
            display: { ...TEXT, labels: { InProgress: "In Progress" } },
          },
        ]}
        rows={[
          ["32.7767", "-96.797", "InProgress"],
          ["41.8781", "-87.6298", "Unlisted"],
        ]}
      />,
    );

    const pins = await screen.findAllByTestId("pin");
    expect(pins.map((pin) => pin.getAttribute("title"))).toEqual(["In Progress", "Unlisted"]);
  });
});
