import type { DictationIssue, DictationPhase } from "./use-dictation";

/**
 * The one hint the line under the composer shows. Only what is useful at
 * this moment, never the whole list: the list is behind the shortcuts
 * button, and a line that wraps into three ragged rows at 400px teaches
 * nothing.
 */
export type ComposerHintKind =
  | "none"
  | "issue"
  | "starting"
  | "listening"
  | "choosing"
  | "replying"
  | "discover"
  | "compose";

export type ComposerHintState = {
  disabled: boolean;
  issue: DictationIssue | null;
  dictation: DictationPhase;
  /** A slash-command or record list is open over the box. */
  menuOpen: boolean;
  /** A reply is being written. */
  active: boolean;
  focused: boolean;
  draftEmpty: boolean;
};

/**
 * Which hint wins. A problem outranks an instruction, what the microphone is
 * doing outranks what the keyboard would do, and the keyboard hints show
 * only while the box has focus: an empty draft teaches the slash and the at
 * sign, a draft in progress says how to send it.
 */
export function composerHint(state: ComposerHintState): ComposerHintKind {
  if (state.disabled) {
    return "none";
  }
  if (state.issue !== null && state.dictation === "idle") {
    return "issue";
  }
  if (state.dictation === "starting") {
    return "starting";
  }
  if (state.dictation === "listening") {
    return "listening";
  }
  if (state.menuOpen) {
    return "choosing";
  }
  if (state.active) {
    return "replying";
  }
  if (!state.focused) {
    return "none";
  }

  return state.draftEmpty ? "discover" : "compose";
}
