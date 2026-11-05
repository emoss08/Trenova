import { describe, expect, it } from "vitest";
import { approvesFromKey } from "../approve-shortcut";

function textarea(value: string): HTMLTextAreaElement {
  const box = document.createElement("textarea");
  box.value = value;
  return box;
}

function key(overrides: Partial<Parameters<typeof approvesFromKey>[0]> = {}) {
  return {
    key: "Enter",
    metaKey: false,
    ctrlKey: true,
    defaultPrevented: false,
    target: document.body as EventTarget | null,
    ...overrides,
  };
}

/**
 * Cmd/Ctrl+Enter in the Desk's stage approves the change waiting on the person.
 * With "Send with Cmd Enter" chosen, the same keystroke in the message box sends
 * the message; it once went on to approve the change the person was asking about.
 */
describe("approvesFromKey", () => {
  it("approves from an empty message box, with Ctrl on Windows or Cmd on a Mac", () => {
    expect(approvesFromKey(key({ target: textarea("") }))).toBe(true);
    expect(approvesFromKey(key({ ctrlKey: false, metaKey: true, target: textarea("  ") }))).toBe(
      true,
    );
  });

  it("approves when focus is on the card or anywhere else in the stage", () => {
    expect(approvesFromKey(key({ target: document.createElement("button") }))).toBe(true);
    expect(approvesFromKey(key({ target: document.body }))).toBe(true);
  });

  it("never approves a key the message box already used to send", () => {
    expect(approvesFromKey(key({ target: textarea(""), defaultPrevented: true }))).toBe(false);
  });

  it("never approves while the message box holds a question", () => {
    expect(approvesFromKey(key({ target: textarea("Why this driver?") }))).toBe(false);
  });

  it("never approves from another text field that holds text", () => {
    const input = document.createElement("input");
    input.value = "S-1001";
    expect(approvesFromKey(key({ target: input }))).toBe(false);
  });

  it("does not approve on Enter alone or on another key", () => {
    expect(approvesFromKey(key({ ctrlKey: false, metaKey: false }))).toBe(false);
    expect(approvesFromKey(key({ key: "a" }))).toBe(false);
  });
});
