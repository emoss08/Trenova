import { describe, expect, it } from "vitest";
import { composerHint, type ComposerHintState } from "../composer-hint";

const resting: ComposerHintState = {
  disabled: false,
  issue: null,
  dictation: "idle",
  menuOpen: false,
  active: false,
  focused: false,
  draftEmpty: true,
};

function hint(overrides: Partial<ComposerHintState>) {
  return composerHint({ ...resting, ...overrides });
}

/**
 * One hint at a time, and the one that matters: a problem before an
 * instruction, the microphone before the keyboard, and keyboard hints only
 * while the box is in use.
 */
describe("composerHint", () => {
  it("says nothing about a box nobody is using", () => {
    expect(hint({})).toBe("none");
    expect(hint({ draftEmpty: false })).toBe("none");
  });

  it("teaches the slash and the at sign on an empty focused box", () => {
    expect(hint({ focused: true })).toBe("discover");
  });

  it("says how to send once there is a draft", () => {
    expect(hint({ focused: true, draftEmpty: false })).toBe("compose");
  });

  it("says how to move through a list that is open", () => {
    expect(hint({ focused: true, menuOpen: true, draftEmpty: false })).toBe("choosing");
  });

  it("says a reply is being written whether or not the box has focus", () => {
    expect(hint({ active: true })).toBe("replying");
    expect(hint({ active: true, focused: true, draftEmpty: false })).toBe("replying");
  });

  it("puts what the microphone is doing ahead of the keyboard", () => {
    expect(hint({ dictation: "starting", focused: true })).toBe("starting");
    expect(hint({ dictation: "listening", focused: true, active: true })).toBe("listening");
  });

  it("puts a dictation problem ahead of everything once the microphone has let go", () => {
    expect(hint({ issue: "denied", focused: true, active: true, menuOpen: true })).toBe("issue");
  });

  it("does not show an old problem over a new attempt", () => {
    expect(hint({ issue: "no-speech", dictation: "listening" })).toBe("listening");
  });

  it("says nothing on a composer that cannot be used", () => {
    expect(hint({ disabled: true, issue: "denied", focused: true })).toBe("none");
  });
});
