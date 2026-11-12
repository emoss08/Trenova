import { act } from "@testing-library/react";
import type { Editor } from "@tiptap/core";
import { docOf, textOf } from "./editor-text";

/**
 * What a test needs of the composer's box. The box is a Tiptap editor, which
 * does not take `fireEvent.change` or a typed `value`: a test writes into it
 * through the editor the box element carries, and reads its words back the
 * same way. Keys are still fired at the element (`fireEvent.keyDown`).
 */
function editorOf(box: HTMLElement): Editor {
  const editor = (box as HTMLElement & { editor?: Editor }).editor;
  if (!editor) {
    throw new Error("The element is not the composer's box");
  }
  return editor;
}

/** Replaces what is in the box, as selecting everything and typing would. */
export function writeInBox(box: HTMLElement, text: string): void {
  act(() => {
    const editor = editorOf(box);
    editor.commands.setContent(docOf(text), { emitUpdate: true });
    editor.commands.setTextSelection(editor.state.doc.content.size);
  });
}

/** Types at the caret, after whatever is in the box. */
export function typeInBox(box: HTMLElement, text: string): void {
  act(() => {
    editorOf(box).commands.insertContent(text);
  });
}

/** The words in the box. */
export function textInBox(box: HTMLElement): string {
  return textOf(editorOf(box).state.doc);
}

/** The placeholder the empty box shows, or "" when it shows none. */
export function placeholderOf(box: HTMLElement): string {
  return box.querySelector("[data-placeholder]")?.getAttribute("data-placeholder") ?? "";
}
