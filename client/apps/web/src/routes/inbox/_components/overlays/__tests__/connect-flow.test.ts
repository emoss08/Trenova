import type { InboundMailbox } from "@/lib/graphql/inbox";
import { describe, expect, it } from "vitest";
import {
  canContinue,
  canGoBack,
  connectReducer,
  initialConnect,
  receivingAddress,
  receivingDomain,
  testArrived,
  type ConnectState,
} from "../connect/connect-flow";

function mailbox(overrides: Partial<InboundMailbox> = {}): InboundMailbox {
  return {
    id: "imb_dispatch",
    name: "Dispatch",
    address: "dispatch@dev.trenova.app",
    provider: "Postmark",
    purpose: "",
    reviewPolicy: "ReviewBelowConfidence",
    minConfidence: 0.8,
    status: "Active",
    hasSigningSecret: true,
    hasApiKey: false,
    lastReceivedAt: null,
    health: "NeverReceived",
    testSentAt: null,
    testReceivedAt: null,
    blockedSenders: [],
    version: 1,
    createdAt: 1000,
    updatedAt: 1000,
    ...overrides,
  };
}

function filled(domain: string | null = "dev.trenova.app"): ConnectState {
  let state = connectReducer(initialConnect(null, domain), { type: "continue" });
  state = connectReducer(state, { type: "edit", field: "watch", value: "dispatch@acme.com" });
  return connectReducer(state, { type: "edit", field: "name", value: "Dispatch" });
}

describe("where Trenova receives a new source's mail", () => {
  it("uses the domain the other sources already receive on", () => {
    expect(
      receivingDomain([
        mailbox({ address: "intake@dev.trenova.app" }),
        mailbox({ address: "claims@dev.trenova.app" }),
        mailbox({ address: "billing@other.example" }),
      ]),
    ).toBe("dev.trenova.app");
    expect(receivingDomain([])).toBeNull();
  });

  it("names the address after the source", () => {
    expect(receivingAddress("Dispatch", "dev.trenova.app")).toBe("dispatch@dev.trenova.app");
    expect(receivingAddress("AP / Billing", "dev.trenova.app")).toBe("ap-billing@dev.trenova.app");
    expect(receivingAddress("   ", "dev.trenova.app")).toBe("");
    expect(receivingAddress("Dispatch", null)).toBe("");
  });
});

describe("connecting a source", () => {
  it("starts on the choice of source and moves to the address once email is chosen", () => {
    const start = initialConnect(null, "dev.trenova.app");
    expect(start.step).toBe("source");
    expect(canContinue(start)).toBe(true);
    expect(connectReducer(start, { type: "continue" }).step).toBe("address");
  });

  it("fills the receiving address from the name until it is typed over", () => {
    const state = filled();
    expect(state.receiveAt).toBe("dispatch@dev.trenova.app");
    const typed = connectReducer(state, {
      type: "edit",
      field: "receiveAt",
      value: "ops@inbound.acme.com",
    });
    const renamed = connectReducer(typed, { type: "edit", field: "name", value: "Ops" });
    expect(renamed.receiveAt).toBe("ops@inbound.acme.com");
  });

  it("needs a name, the address to watch and a different address to receive on", () => {
    expect(canContinue(filled())).toBe(true);
    expect(canContinue(connectReducer(filled(), { type: "edit", field: "name", value: " " }))).toBe(
      false,
    );
    expect(
      canContinue(connectReducer(filled(), { type: "edit", field: "watch", value: "dispatch" })),
    ).toBe(false);
    expect(
      canContinue(
        connectReducer(filled(), { type: "edit", field: "receiveAt", value: "dispatch@acme.com" }),
      ),
    ).toBe(false);
    expect(canContinue(filled(null))).toBe(false);
  });

  it("creates the source before showing where to forward, and says why when it cannot", () => {
    const saving = connectReducer(filled(), { type: "saving" });
    expect(saving.saving).toBe(true);
    expect(canContinue(saving)).toBe(false);

    const failed = connectReducer(saving, { type: "failed", message: "Address is taken" });
    expect(failed).toMatchObject({ step: "address", saving: false, error: "Address is taken" });

    const created = connectReducer(saving, {
      type: "created",
      mailbox: mailbox(),
      webhookUrl: "https://api.example/inbound/x",
    });
    expect(created).toMatchObject({ step: "forward", saving: false, error: null });
    expect(created.mailbox?.id).toBe("imb_dispatch");
    expect(canGoBack(created)).toBe(false);
  });

  it("waits for mail after the forward is set up and is done when it arrives", () => {
    let state = connectReducer(connectReducer(filled(), { type: "saving" }), {
      type: "created",
      mailbox: mailbox(),
      webhookUrl: null,
    });
    state = connectReducer(state, { type: "continue" });
    expect(state.step).toBe("test");
    expect(canGoBack(state)).toBe(true);

    state = connectReducer(state, { type: "waitFor", via: "person", since: 5000 });
    expect(state.waiting).toEqual({ via: "person", since: 5000 });
    expect(connectReducer(state, { type: "back" }).waiting).toBeNull();

    const done = connectReducer(state, { type: "arrived" });
    expect(done.step).toBe("done");
    expect(done.waiting).toBeNull();
  });

  it("reconnects a named source from where to forward, with nothing to create", () => {
    const state = initialConnect(mailbox(), "dev.trenova.app");
    expect(state.step).toBe("forward");
    expect(state.mailbox?.address).toBe("dispatch@dev.trenova.app");
    expect(canGoBack(state)).toBe(false);
  });
});

describe("knowing the mail arrived", () => {
  const since = 10_000;

  it("counts any mail received since the person said they sent it", () => {
    expect(testArrived(mailbox({ lastReceivedAt: since + 5 }), { via: "person", since })).toBe(
      true,
    );
    expect(testArrived(mailbox({ lastReceivedAt: since - 3600 }), { via: "person", since })).toBe(
      false,
    );
  });

  it("allows for a provider's clock a little behind ours", () => {
    expect(testArrived(mailbox({ lastReceivedAt: since - 30 }), { via: "person", since })).toBe(
      true,
    );
  });

  it("needs Trenova's own test back for a test it sent", () => {
    expect(
      testArrived(mailbox({ testSentAt: since, testReceivedAt: null, lastReceivedAt: since + 1 }), {
        via: "test",
        since,
      }),
    ).toBe(false);
    expect(
      testArrived(mailbox({ testSentAt: since, testReceivedAt: since + 4 }), {
        via: "test",
        since,
      }),
    ).toBe(true);
    expect(
      testArrived(mailbox({ testSentAt: since + 9, testReceivedAt: since + 4 }), {
        via: "test",
        since,
      }),
    ).toBe(false);
  });
});
