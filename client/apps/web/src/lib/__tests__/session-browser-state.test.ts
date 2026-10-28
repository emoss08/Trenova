import { useAssistantStore } from "@/stores/assistant-store";
import { useDeskSettingsStore } from "@/stores/desk-settings-store";
import { useDeskStore } from "@/stores/desk-store";
import { useRecentPagesStore } from "@/stores/recent-pages-store";
import { useRecentRecordsStore } from "@/stores/recent-records-store";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { User } from "@trenova/shared/types/user";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { clearSessionCache } from "../session-cache";

const mocks = vi.hoisted(() => ({ clearDocumentUploadSessions: vi.fn() }));

vi.mock("@/lib/document-upload-store", () => ({
  clearDocumentUploadSessions: mocks.clearDocumentUploadSessions,
}));

const ORG = "org_1";

// What a dispatcher does in a morning, every piece of it naming a record or a person.
function work() {
  useAuthStore.getState().setUser({
    id: "usr_1",
    currentOrganizationId: ORG,
    name: "Jane Doe",
    username: "jdoe",
    emailAddress: "jane.doe@acme-freight.test",
    phoneNumber: "+1 555 0100",
    timezone: "America/Denver",
  } as User);
  useRecentRecordsStore.getState().recordOpen(ORG, {
    entityType: "customer",
    id: "cus_1",
    title: "Acme Freight Holdings",
    subtitle: "Net 30, credit hold",
    href: "/billing/customers?expanded=cus_1",
  });
  useRecentPagesStore
    .getState()
    .recordVisit(ORG, { path: "/shipments/shp_1", title: "Shipment S-100234" });
  useDeskSettingsStore.getState().rememberSearch("overdue invoices Acme Freight");
  useAssistantStore.getState().setDraft("thr_1", "Call Jane Doe about S-100234 detention");
  useAssistantStore.getState().setActiveThreadId("thr_1");
  useAssistantStore.getState().deferDecisions(["proposal:prp_1"]);
  useAssistantStore.getState().setLastAgentId("agt_1");
  useDeskStore.getState().setActiveArtifact("thr_1", "art_1");
  useDeskStore.getState().toggleChapter("thr_1", "msg_1");
  useDeskStore.getState().askAgent("thr_1", "Create a shipment from Acme Freight's BOL");
}

const SENSITIVE = [
  "Jane Doe",
  "jdoe",
  "jane.doe@acme-freight.test",
  "555 0100",
  "Acme Freight",
  "credit hold",
  "S-100234",
  "overdue invoices",
];

function everythingStored(): string {
  const entries: string[] = [];
  for (const storage of [localStorage, sessionStorage]) {
    for (let index = 0; index < storage.length; index++) {
      const key = storage.key(index) ?? "";
      entries.push(`${key}=${storage.getItem(key) ?? ""}`);
    }
  }
  return entries.join("\n");
}

beforeEach(() => {
  localStorage.clear();
  sessionStorage.clear();
  mocks.clearDocumentUploadSessions.mockReset();
  mocks.clearDocumentUploadSessions.mockResolvedValue(undefined);
  useAuthStore.setState({ user: null, isAuthenticated: false });
  useRecentRecordsStore.getState().clear();
  useRecentPagesStore.getState().clear();
  useDeskSettingsStore.getState().forgetSearches();
  useAssistantStore.getState().endSession();
  useDeskStore.getState().endSession();
});

describe("what the browser keeps of a session", () => {
  it("writes nothing that names a record or a person to the browser's storage", () => {
    work();

    const stored = everythingStored();
    for (const text of SENSITIVE) {
      expect(stored).not.toContain(text);
    }
  });

  it("keeps the person's work in memory while the session lasts", () => {
    work();

    expect(useAuthStore.getState().user?.name).toBe("Jane Doe");
    expect(useRecentRecordsStore.getState().recordsByOrganization[ORG]?.[0]?.title).toBe(
      "Acme Freight Holdings",
    );
    expect(useRecentPagesStore.getState().pagesByOrganization[ORG]).toEqual([
      { path: "/shipments/shp_1", title: "Shipment S-100234" },
    ]);
    expect(useDeskSettingsStore.getState().recentSearches).toEqual([
      "overdue invoices Acme Freight",
    ]);
    expect(useAssistantStore.getState().drafts).toEqual({
      thr_1: "Call Jane Doe about S-100234 detention",
    });
  });

  it("forgets every record, page, search, draft and conversation when the session ends", async () => {
    work();

    await clearSessionCache();

    expect(useRecentRecordsStore.getState().recordsByOrganization).toEqual({});
    expect(useRecentPagesStore.getState().pagesByOrganization).toEqual({});
    expect(useDeskSettingsStore.getState().recentSearches).toEqual([]);
    expect(useAssistantStore.getState()).toMatchObject({
      drafts: {},
      activeThreadId: null,
      deferredDecisions: [],
      lastAgentId: null,
      decisionFocus: {},
      openingQuestion: null,
    });
    expect(useDeskStore.getState()).toMatchObject({
      activeArtifactByThread: {},
      chaptersByThread: {},
      asks: {},
      documentAsks: {},
      liveHandoff: {},
    });
    expect(mocks.clearDocumentUploadSessions).toHaveBeenCalledTimes(1);
    expect(everythingStored()).not.toContain("thr_1");
  });

  it("keeps how the person laid out the Desk and the assistant across a session end", async () => {
    useDeskStore.getState().setRail("closed");
    useAssistantStore.getState().setDock("bottom-left");
    useDeskSettingsStore.getState().setRailWidth(300);
    work();

    await clearSessionCache();

    expect(useDeskStore.getState().rail).toBe("closed");
    expect(useAssistantStore.getState().dock).toBe("bottom-left");
    expect(useDeskSettingsStore.getState().railWidth).toBe(300);
  });

  it("still clears memory when the upload records cannot be reached", async () => {
    const error = vi.spyOn(console, "error").mockImplementation(() => undefined);
    mocks.clearDocumentUploadSessions.mockRejectedValue(new Error("QuotaExceededError"));
    work();

    await expect(clearSessionCache()).resolves.toBeUndefined();

    expect(useRecentRecordsStore.getState().recordsByOrganization).toEqual({});
    expect(error).toHaveBeenCalledWith(
      "Could not delete the uploads kept for resuming:",
      expect.any(Error),
    );
    error.mockRestore();
  });
});
