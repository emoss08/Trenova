import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";
import {
  RECENT_RECORDS_LIMIT,
  useRecentRecords,
  useRecentRecordsStore,
  type RecentRecordInput,
} from "../recent-records-store";

const ORG_A = "org_a";
const ORG_B = "org_b";

function shipment(id: string, title = `PRO-${id}`): RecentRecordInput {
  return {
    entityType: "shipment",
    id,
    title,
    href: `/shipment-management/shipments?expanded=${id}`,
  };
}

function idsFor(organizationId: string | undefined): string[] {
  const { result } = renderHook(() => useRecentRecords(organizationId));
  return result.current.map((record) => `${record.entityType}:${record.id}`);
}

describe("recent records store", () => {
  beforeEach(() => {
    useRecentRecordsStore.setState({ recordsByOrganization: {} });
  });

  it("records an opened record with the newest first", () => {
    const { recordOpen } = useRecentRecordsStore.getState();
    recordOpen(ORG_A, shipment("1"));
    recordOpen(ORG_A, shipment("2"));

    expect(idsFor(ORG_A)).toEqual(["shipment:2", "shipment:1"]);
  });

  it("keeps each organization's records apart", () => {
    const { recordOpen } = useRecentRecordsStore.getState();
    recordOpen(ORG_A, shipment("1"));
    recordOpen(ORG_B, shipment("2"));

    expect(idsFor(ORG_A)).toEqual(["shipment:1"]);
    expect(idsFor(ORG_B)).toEqual(["shipment:2"]);
  });

  it("moves a reopened record to the front and keeps its newest title", () => {
    const { recordOpen } = useRecentRecordsStore.getState();
    recordOpen(ORG_A, shipment("1", "Old"));
    recordOpen(ORG_A, shipment("2"));
    recordOpen(ORG_A, shipment("1", "New"));

    expect(idsFor(ORG_A)).toEqual(["shipment:1", "shipment:2"]);
    expect(useRecentRecordsStore.getState().recordsByOrganization[ORG_A]?.[0]?.title).toBe("New");
  });

  it("tells records of different types with the same id apart", () => {
    const { recordOpen } = useRecentRecordsStore.getState();
    recordOpen(ORG_A, shipment("1"));
    recordOpen(ORG_A, { entityType: "customer", id: "1", title: "Acme", href: "/c" });

    expect(idsFor(ORG_A)).toEqual(["customer:1", "shipment:1"]);
  });

  it("keeps only the most recent records", () => {
    const { recordOpen } = useRecentRecordsStore.getState();
    for (let i = 0; i < RECENT_RECORDS_LIMIT + 3; i++) {
      recordOpen(ORG_A, shipment(String(i)));
    }

    expect(idsFor(ORG_A)).toHaveLength(RECENT_RECORDS_LIMIT);
    expect(idsFor(ORG_A)[0]).toBe(`shipment:${RECENT_RECORDS_LIMIT + 2}`);
  });

  it("ignores a record with no organization, id or title", () => {
    const { recordOpen } = useRecentRecordsStore.getState();
    recordOpen(undefined, shipment("1"));
    recordOpen(ORG_A, shipment(""));
    recordOpen(ORG_A, shipment("2", "   "));

    expect(idsFor(ORG_A)).toEqual([]);
    expect(idsFor(undefined)).toEqual([]);
  });

  it("brings a known record forward when it is opened elsewhere, and never adds an unknown one", () => {
    const { recordOpen, touch } = useRecentRecordsStore.getState();
    recordOpen(ORG_A, shipment("1"));
    recordOpen(ORG_A, shipment("2"));
    touch(ORG_A, "shipment", "1");
    touch(ORG_A, "shipment", "3");

    expect(idsFor(ORG_A)).toEqual(["shipment:1", "shipment:2"]);
  });

  it("forgets a record", () => {
    const { recordOpen, forget } = useRecentRecordsStore.getState();
    recordOpen(ORG_A, shipment("1"));
    recordOpen(ORG_A, shipment("2"));
    forget(ORG_A, "shipment", "1");

    expect(idsFor(ORG_A)).toEqual(["shipment:2"]);
  });
});
