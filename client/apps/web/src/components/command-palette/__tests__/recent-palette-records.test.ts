import { clearPermissions, seedPermissions } from "@/test/permissions";
import { useRecentRecordsStore } from "@/stores/recent-records-store";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Operation } from "@trenova/shared/types/permission";
import { renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { useRecentPaletteRecords } from "../use-palette-data";

function opened(entityType: "shipment" | "invoice", id: string) {
  useRecentRecordsStore
    .getState()
    .recordOpen("org_1", { entityType, id, title: `${entityType} ${id}`, href: `/${id}` });
}

function offered(): string[] {
  const { result } = renderHook(() => useRecentPaletteRecords());
  return result.current.map(({ record }) => `${record.entityType}:${record.id}`);
}

beforeEach(() => {
  useRecentRecordsStore.setState({ recordsByOrganization: {} });
  useAuthStore.setState({
    user: { currentOrganizationId: "org_1" } as ReturnType<typeof useAuthStore.getState>["user"],
  });
});

afterEach(() => {
  clearPermissions();
});

describe("useRecentPaletteRecords", () => {
  it("offers a recent record only while the person may still read its kind", () => {
    opened("shipment", "shp_1");
    opened("invoice", "inv_1");
    seedPermissions({ shipment: Operation.Read });

    expect(offered()).toEqual(["shipment:shp_1"]);
  });

  it("offers every recent record, newest first, to someone who reads both kinds", () => {
    opened("shipment", "shp_1");
    opened("invoice", "inv_1");
    seedPermissions({ shipment: Operation.Read, invoice: Operation.Read });

    expect(offered()).toEqual(["invoice:inv_1", "shipment:shp_1"]);
  });

  it("offers none before the person's permissions have loaded", () => {
    opened("shipment", "shp_1");

    expect(offered()).toEqual([]);
  });
});
