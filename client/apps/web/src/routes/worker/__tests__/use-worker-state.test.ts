import { readSearchState } from "@trenova/shared/hooks/use-search-state";
import { describe, expect, it } from "vitest";
import {
  CLOSED_WORKER_OPEN_STATE,
  WORKER_FORM_HOMES,
  WORKER_FORMS,
  WORKER_TAB_VIEWS,
  WORKER_VIEWS,
  isWorkerViewOf,
  resolveWorkerView,
  WORKER_FORM_RECORDS,
  workerFormBlockedByAddress,
  workerFormTarget,
  workerOpenStateFor,
  workerPageParsers,
  workerPath,
  workerTabOfView,
  type WorkerTab,
} from "../use-worker-state";

function read(query: string) {
  return readSearchState(workerPageParsers, new URLSearchParams(query));
}

describe("workerPath", () => {
  it("opens a worker on the overview with no query", () => {
    expect(workerPath("wrk_1")).toBe("/hr/workers/wrk_1");
    expect(workerPath("wrk_1", { tab: "overview" })).toBe("/hr/workers/wrk_1");
  });

  it("names the tab and the view", () => {
    expect(workerPath("wrk_1", { tab: "safety" })).toBe("/hr/workers/wrk_1?tab=safety");
    expect(workerPath("wrk_1", { tab: "time", view: "hours-of-service" })).toBe(
      "/hr/workers/wrk_1?tab=time&view=hours-of-service",
    );
  });

  it("carries the record a link opens and the form it starts", () => {
    expect(
      workerPath("wrk_1", { tab: "qualification", view: "credentials", credential: "wcr_9" }),
    ).toBe("/hr/workers/wrk_1?tab=qualification&view=credentials&credential=wcr_9");
    expect(
      workerPath("wrk_1", {
        tab: "employment",
        view: "history",
        form: "record-event",
        kind: "Terminated",
      }),
    ).toBe("/hr/workers/wrk_1?tab=employment&view=history&form=record-event&kind=Terminated");
  });

  it("drops a preset that has no form to preset", () => {
    expect(workerPath("wrk_1", { tab: "employment", kind: "Terminated" })).toBe(
      "/hr/workers/wrk_1?tab=employment",
    );
  });

  it("encodes the id as one path segment", () => {
    expect(workerPath("a/b?c")).toBe("/hr/workers/a%2Fb%3Fc");
  });
});

describe("the worker page's address", () => {
  it("reads an address with nothing in it as the overview with nothing open", () => {
    expect(read("")).toEqual({ tab: "overview", view: null, ...CLOSED_WORKER_OPEN_STATE });
  });

  it("reads what workerPath writes", () => {
    const url = new URL(
      workerPath("wrk_1", {
        tab: "development",
        view: "training",
        training: "wtr_2",
        form: "record-training-result",
        kind: "retake",
      }),
      "http://localhost",
    );

    expect(read(url.search)).toEqual({
      ...CLOSED_WORKER_OPEN_STATE,
      tab: "development",
      view: "training",
      training: "wtr_2",
      form: "record-training-result",
      kind: "retake",
    });
  });

  it("keeps the course a row chose for its form, so a reload or a shared link keeps it", () => {
    const path = workerPath("wrk_1", {
      tab: "development",
      view: "training",
      form: "assign-training",
      course: "trc_7",
    });

    expect(path).toBe(
      "/hr/workers/wrk_1?tab=development&view=training&course=trc_7&form=assign-training",
    );
    expect(read(new URL(path, "http://localhost").search)).toEqual({
      ...CLOSED_WORKER_OPEN_STATE,
      tab: "development",
      view: "training",
      form: "assign-training",
      course: "trc_7",
    });
  });

  it("reads a tab, view or form it does not know as nothing", () => {
    expect(read("?tab=timeline&view=hos&form=delete-everything")).toMatchObject({
      tab: "overview",
      view: null,
      form: null,
    });
  });
});

describe("views", () => {
  it("names every view once, in tab order", () => {
    expect(WORKER_VIEWS).toEqual([
      "history",
      "checklists",
      "credentials",
      "dq-file",
      "drug-and-alcohol",
      "training",
      "reviews",
      "hours-of-service",
      "schedule",
      "time-off",
      "leave",
      "pay-profile",
      "benefits",
      "documents",
      "driver-portal",
      "agent-memory",
    ]);
  });

  it("keeps a view the tab has", () => {
    expect(resolveWorkerView("time", "leave")).toBe("leave");
  });

  it("falls to the tab's first view for a view of another tab or none", () => {
    expect(resolveWorkerView("time", "credentials")).toBe("hours-of-service");
    expect(resolveWorkerView("files", null)).toBe("documents");
    expect(resolveWorkerView("qualification", "nonsense")).toBe("credentials");
  });

  it("has no view on a tab without views", () => {
    expect(resolveWorkerView("safety", "credentials")).toBeNull();
    expect(resolveWorkerView("overview", null)).toBeNull();
  });

  it("finds the tab a view belongs to", () => {
    expect(workerTabOfView("benefits")).toBe("pay");
    expect(workerTabOfView("agent-memory")).toBe("files");
    expect(isWorkerViewOf("pay", "benefits")).toBe(true);
    expect(isWorkerViewOf("time", "benefits")).toBe(false);
  });
});

describe("forms", () => {
  it.each(WORKER_FORMS)("opens %s on a view of its own tab", (form) => {
    const home = WORKER_FORM_HOMES[form] as { tab: WorkerTab; view: string | null };
    const views = WORKER_TAB_VIEWS[home.tab] as readonly string[];

    expect(home.view === null ? views.length === 0 : views.includes(home.view)).toBe(true);
  });

  it("targets the tab, view and preset a form opens on", () => {
    expect(workerFormTarget("record-event", "Terminated")).toEqual({
      tab: "employment",
      view: "history",
      form: "record-event",
      kind: "Terminated",
    });
    expect(workerFormTarget("record-safety-event")).toEqual({
      tab: "safety",
      form: "record-safety-event",
    });
  });
});

describe("workerOpenStateFor", () => {
  const state = {
    ...CLOSED_WORKER_OPEN_STATE,
    tab: "qualification" as const,
    form: "renew-credential" as const,
    kind: "medical",
    credential: "wcr_1",
  };

  it("gives the selected tab its record and form", () => {
    expect(workerOpenStateFor(state, "qualification")).toEqual({
      ...CLOSED_WORKER_OPEN_STATE,
      form: "renew-credential",
      kind: "medical",
      credential: "wcr_1",
    });
  });

  it("gives every other tab nothing open, so a hidden tab never opens a pane", () => {
    expect(workerOpenStateFor(state, "safety")).toEqual(CLOSED_WORKER_OPEN_STATE);
  });

  it("drops a preset when no form is open", () => {
    expect(
      workerOpenStateFor({ ...state, form: null, kind: "medical" }, "qualification").kind,
    ).toBeNull();
  });
});

describe("workerFormBlockedByAddress", () => {
  const everything = () => true;
  const state = (query: string) => {
    const read_ = read(query);
    return { ...read_, tab: read_.tab };
  };

  it("blocks a form that edits a record the address does not name", () => {
    for (const [form, param] of Object.entries(WORKER_FORM_RECORDS)) {
      const home = WORKER_FORM_HOMES[form as keyof typeof WORKER_FORM_HOMES];
      const base = `tab=${home.tab}${home.view ? `&view=${home.view}` : ""}&form=${form}`;
      expect(workerFormBlockedByAddress(state(base), everything), form).toBe(true);
      expect(
        workerFormBlockedByAddress(state(`${base}&${param}=rec_1`), everything),
        form,
      ).toBe(false);
    }
  });

  it("lets a form that needs no record through, with or without a record named", () => {
    expect(
      workerFormBlockedByAddress(
        state("tab=development&view=training&form=record-training-result"),
        everything,
      ),
    ).toBe(false);
    expect(
      workerFormBlockedByAddress(
        state("tab=development&view=training&form=record-training-result&training=wtr_x"),
        everything,
      ),
    ).toBe(false);
  });

  it("blocks a form whose view the reader may not draw, or one away from its own tab", () => {
    const noReviews = ({ view }: { view: string | null }) => view !== "reviews";
    expect(
      workerFormBlockedByAddress(
        state("tab=development&view=reviews&form=start-review"),
        noReviews,
      ),
    ).toBe(true);
    expect(
      workerFormBlockedByAddress(state("tab=safety&form=start-review"), everything),
    ).toBe(true);
    expect(workerFormBlockedByAddress(state("tab=safety"), everything)).toBe(false);
  });
});
