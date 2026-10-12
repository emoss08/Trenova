import { describe, expect, it } from "vitest";
import { effectivePresence, groupPresence } from "../use-triage-presence";

describe("groupPresence", () => {
  const members = [
    { userId: "usr_me", connectionId: "c0", name: "Me", messageId: "imsg_1", activity: "viewing" as const },
    { userId: "usr_dana", connectionId: "c1", name: "Dana Whitfield", messageId: "imsg_1", activity: "viewing" as const },
    { userId: "usr_dana", connectionId: "c2", name: "Dana Whitfield", messageId: "imsg_1", activity: "writing" as const },
    { userId: "usr_priya", connectionId: "c3", name: "Priya Shah", messageId: "imsg_2" },
    { userId: "usr_marcus", connectionId: "c4", name: "Marcus Lee", messageId: null },
  ];

  it("groups the others by the message they have open, once per person, writing over viewing", () => {
    expect(groupPresence(members, "usr_me")).toEqual({
      imsg_1: [{ userId: "usr_dana", name: "Dana Whitfield", activity: "writing" }],
      imsg_2: [{ userId: "usr_priya", name: "Priya Shah", activity: "viewing" }],
    });
  });

  it("keeps a message's list when nothing about it changed, so its row does not re-render", () => {
    const first = groupPresence(members, "usr_me");
    const second = groupPresence([...members.slice(0, 3), { ...members[3], messageId: "imsg_3" }], "usr_me", first);
    expect(second.imsg_1).toBe(first.imsg_1);
    expect(second.imsg_2).toBeUndefined();
  });
});

describe("effectivePresence", () => {
  it("says writing while any composer is open, else the latest reader, else nothing open", () => {
    expect(effectivePresence([])).toEqual({ messageId: null, activity: "viewing" });
    expect(
      effectivePresence([
        { token: 1, messageId: "a", activity: "viewing" },
        { token: 2, messageId: "a", activity: "writing" },
        { token: 3, messageId: "b", activity: "viewing" },
      ]),
    ).toEqual({ messageId: "a", activity: "writing" });
    expect(
      effectivePresence([
        { token: 1, messageId: "a", activity: "viewing" },
        { token: 3, messageId: "b", activity: "viewing" },
      ]),
    ).toEqual({ messageId: "b", activity: "viewing" });
  });
});
