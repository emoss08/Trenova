import "./desk-editor.css";
import type { AssistantEntityRef } from "@/types/assistant";
import { Extension } from "@tiptap/core";
import { Fragment, Slice } from "@tiptap/pm/model";
import { Plugin, PluginKey, TextSelection } from "@tiptap/pm/state";
import { Decoration, DecorationSet, type EditorView } from "@tiptap/pm/view";
import { EditorContent, useEditor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import { useEffect, useImperativeHandle, useRef, useState, type Ref } from "react";
import {
  cleanPastedText,
  docOf,
  mentionEndingAt,
  mentionRanges,
  offsetAt,
  pastesAsFile,
  posAt,
  textOf,
  textOfHtml,
} from "./editor-text";

/** What the composer's menus need of the box: its words, its caret and the focus. */
export type DeskEditorHandle = {
  focus: () => void;
  /** The draft as it stands in the box, ahead of any render. */
  text: () => string;
  /** The caret, as an offset into the draft. */
  caret: () => number;
  /** Whether the selection is a caret and not a range. */
  collapsed: () => boolean;
  setCaret: (offset: number) => void;
};

/** A slash command being filled in: how much of the draft is its name, and the slots left. */
export type DeskEditorCommand = { length: number; hint: string };

export type DeskEditorProps = {
  ref?: Ref<DeskEditorHandle>;
  value: string;
  onChange: (text: string, caret: number) => void;
  /** The caret moved without the words changing. */
  onCaret?: (text: string, caret: number) => void;
  /** A key before the box acts on it. True when it was handled and the box should not. */
  onKey?: (event: KeyboardEvent) => boolean;
  onFocus?: () => void;
  disabled?: boolean;
  placeholder?: string;
  label: string;
  describedBy?: string;
  keyShortcuts?: string;
  /** Records named with @, drawn as chips where their @label stands. */
  mentions?: readonly AssistantEntityRef[];
  command?: DeskEditorCommand | null;
  /** Takes files pasted in, and a paste too long to be a message. Without it files are not taken. */
  onFiles?: (files: File[]) => void;
  /** Whether a paste too long to be a message is attached as a file; else it is typed in. */
  longPasteAsFile?: boolean;
  /** The name a paste too long to be a message is attached under. */
  pastedFileName?: string;
};

/** What the box draws over its words; see `decorate`. */
type Marks = {
  mentions: readonly AssistantEntityRef[];
  command: DeskEditorCommand | null;
  placeholder: string;
};

type PasteTarget = Pick<DeskEditorProps, "onFiles"> & {
  pastedFileName: string;
  longPasteAsFile: boolean;
};

declare module "@tiptap/core" {
  interface Storage {
    deskComposer: { marks: Marks };
  }
}

const marksKey = new PluginKey<DecorationSet>("deskComposerMarks");

function ghost(hint: string): HTMLElement {
  const span = document.createElement("span");
  span.className = "dk-sl-ghost";
  span.textContent = hint;
  span.setAttribute("aria-hidden", "true");
  return span;
}

/**
 * What is drawn over the plain text: each named record's @label as a chip,
 * a command's name set apart with its empty slots as a hint after the caret,
 * and the placeholder in an empty box. None of it is in the document, so the
 * draft stays the words the person typed.
 */
function decorate(doc: Parameters<typeof textOf>[0], marks: Marks): DecorationSet {
  const text = textOf(doc);
  const decorations: Decoration[] = [];
  if (text === "") {
    if (marks.placeholder !== "") {
      decorations.push(
        Decoration.node(0, doc.firstChild?.nodeSize ?? 2, {
          class: "dk-ed-empty",
          "data-placeholder": marks.placeholder,
        }),
      );
    }
    return DecorationSet.create(doc, decorations);
  }
  if (marks.command !== null && marks.command.length <= text.length) {
    decorations.push(
      Decoration.inline(posAt(doc, 0), posAt(doc, marks.command.length), { class: "dk-sl-cmd" }),
    );
    if (marks.command.hint !== "") {
      const hint = (text.endsWith(" ") ? "" : " ") + marks.command.hint;
      decorations.push(
        Decoration.widget(posAt(doc, text.length), () => ghost(hint), { side: 1, key: hint }),
      );
    }
    return DecorationSet.create(doc, decorations);
  }
  for (const range of mentionRanges(text, marks.mentions)) {
    decorations.push(
      Decoration.inline(posAt(doc, range.from), posAt(doc, range.to), { class: "dk-ed-mn" }),
    );
  }
  return DecorationSet.create(doc, decorations);
}

function insertLines(view: EditorView, text: string) {
  const { schema } = view.state;
  const paragraphs = text
    .split("\n")
    .map((line) => schema.nodes.paragraph.create(null, line === "" ? null : schema.text(line)));
  view.dispatch(
    view.state.tr
      .replaceSelection(new Slice(Fragment.from(paragraphs), 1, 1))
      .scrollIntoView()
      .setMeta("paste", true),
  );
}

function paste(view: EditorView, event: ClipboardEvent, target: PasteTarget): boolean {
  const data = event.clipboardData;
  if (!data) {
    return false;
  }
  const { onFiles, pastedFileName, longPasteAsFile } = target;
  const plain = data.getData("text/plain");
  const html = plain === "" ? data.getData("text/html") : "";
  const text = cleanPastedText(plain !== "" ? plain : html ? textOfHtml(html) : "");
  // A spreadsheet's cells arrive as text and as a picture of them: the words
  // are what was meant, so files are taken only when there are no words.
  if (text === "") {
    const files = [...data.files];
    if (files.length > 0 && onFiles) {
      onFiles(files);
    }
    return true;
  }
  if (onFiles && longPasteAsFile && pastesAsFile(text)) {
    onFiles([new File([text], pastedFileName, { type: "text/plain" })]);
    return true;
  }
  insertLines(view, text);
  return true;
}

/**
 * The composer's own part of the editor: Shift+Enter breaks a line, one
 * Backspace takes a named record whole, and the marks are drawn. The marks
 * live in the extension's storage, which the box writes as its props change.
 */
const ComposerExtension = Extension.create<{ marks: Marks }, { marks: Marks }>({
  name: "deskComposer",

  addOptions() {
    return { marks: { mentions: [], command: null, placeholder: "" } };
  },

  addStorage() {
    return { marks: this.options.marks };
  },

  addKeyboardShortcuts() {
    return {
      "Shift-Enter": () => this.editor.commands.splitBlock(),
      Backspace: () => {
        const { state, view } = this.editor;
        if (!state.selection.empty) {
          return false;
        }
        const range = mentionEndingAt(
          textOf(state.doc),
          this.storage.marks.mentions,
          offsetAt(state.doc, state.selection.head),
        );
        if (range === null) {
          return false;
        }
        view.dispatch(state.tr.delete(posAt(state.doc, range.from), posAt(state.doc, range.to)));
        return true;
      },
    };
  },

  addProseMirrorPlugins() {
    const { storage } = this;
    return [
      new Plugin<DecorationSet>({
        key: marksKey,
        state: {
          init: (_config, state) => decorate(state.doc, storage.marks),
          apply: (tr, set) =>
            tr.docChanged || tr.getMeta(marksKey) ? decorate(tr.doc, storage.marks) : set,
        },
        props: {
          decorations: (state) => marksKey.getState(state),
        },
      }),
    ];
  },
});

const PLAIN_LINES = StarterKit.configure({
  blockquote: false,
  bold: false,
  bulletList: false,
  code: false,
  codeBlock: false,
  dropcursor: false,
  gapcursor: false,
  hardBreak: false,
  heading: false,
  horizontalRule: false,
  italic: false,
  link: false,
  listItem: false,
  listKeymap: false,
  orderedList: false,
  strike: false,
  trailingNode: false,
  underline: false,
});

/**
 * The box a person types into: Tiptap over a document of plain lines. It
 * holds its own words, so a keystroke is on screen before React hears of it;
 * `value` is only followed when it changes from outside, as when a starter
 * question is dropped in or the box is emptied on send.
 */
export function DeskEditor({
  ref,
  value,
  onChange,
  onCaret,
  onKey,
  onFocus,
  disabled = false,
  placeholder = "",
  label,
  describedBy,
  keyShortcuts,
  mentions = NO_MENTIONS,
  command = null,
  onFiles,
  pastedFileName = "Pasted text.txt",
  longPasteAsFile = true,
}: DeskEditorProps) {
  // What the box has told its owner and not yet heard back: a `value` that is
  // one of these is an echo of typing, never a reason to replace the words.
  const told = useRef<string[]>([]);
  // Made once: the editor is told of later marks through the extension's storage.
  const [setup] = useState(() => ({
    extensions: [
      PLAIN_LINES,
      ComposerExtension.configure({ marks: { mentions, command, placeholder } }),
    ],
    content: docOf(value),
  }));

  // Tiptap keeps the callbacks and the editor props below as each render gives
  // them, so they read this render's props without a ref.
  const editor = useEditor({
    ...setup,
    editable: !disabled,
    shouldRerenderOnTransaction: false,
    editorProps: {
      attributes: {
        class: "dk-ed",
        role: "textbox",
        "aria-multiline": "true",
        "aria-label": label,
        ...(describedBy ? { "aria-describedby": describedBy } : {}),
        ...(keyShortcuts ? { "aria-keyshortcuts": keyShortcuts } : {}),
        ...(disabled ? { "aria-disabled": "true" } : {}),
        spellcheck: "true",
      },
      handleKeyDown: (_view, event) => onKey?.(event) ?? false,
      handlePaste: (view, event) =>
        paste(view, event, { onFiles, pastedFileName, longPasteAsFile }),
    },
    // A draft that was already there is continued, so the caret starts at its end.
    onCreate: ({ editor: current }) => {
      current.commands.setTextSelection(current.state.doc.content.size);
    },
    onUpdate: ({ editor: current }) => {
      const text = textOf(current.state.doc);
      told.current.push(text);
      onChange(text, offsetAt(current.state.doc, current.state.selection.head));
    },
    onSelectionUpdate: ({ editor: current, transaction }) => {
      if (!transaction.docChanged) {
        onCaret?.(
          textOf(current.state.doc),
          offsetAt(current.state.doc, current.state.selection.head),
        );
      }
    },
    onFocus: () => onFocus?.(),
  });

  useEffect(() => {
    if (editor.isDestroyed) {
      return;
    }
    const echo = told.current.indexOf(value);
    if (echo !== -1) {
      told.current.splice(0, echo + 1);
      return;
    }
    told.current.length = 0;
    if (textOf(editor.state.doc) === value) {
      return;
    }
    const focused = editor.isFocused;
    editor.commands.setContent(docOf(value), { emitUpdate: false });
    if (focused) {
      editor.commands.focus("end");
    } else {
      editor.commands.setTextSelection(editor.state.doc.content.size);
    }
  }, [editor, value]);

  useEffect(() => {
    if (!editor.isDestroyed && editor.isEditable === disabled) {
      editor.setEditable(!disabled);
    }
  }, [editor, disabled]);

  useEffect(() => {
    if (editor.isDestroyed) {
      return;
    }
    editor.storage.deskComposer.marks = { mentions, command, placeholder };
    editor.view.dispatch(editor.state.tr.setMeta(marksKey, true).setMeta("addToHistory", false));
    // The command is compared by what it draws: a new object each render says nothing new.
    // oxlint-disable-next-line react-hooks/exhaustive-deps
  }, [editor, mentions, command?.length, command?.hint, placeholder]);

  useImperativeHandle(
    ref,
    () => ({
      focus: () => {
        if (!editor.isDestroyed) {
          editor.commands.focus();
        }
      },
      text: () => textOf(editor.state.doc),
      caret: () => offsetAt(editor.state.doc, editor.state.selection.head),
      collapsed: () => editor.state.selection.empty,
      setCaret: (offset) => {
        if (editor.isDestroyed) {
          return;
        }
        const { state, view } = editor;
        view.dispatch(
          state.tr.setSelection(TextSelection.create(state.doc, posAt(state.doc, offset))),
        );
        view.focus();
      },
    }),
    [editor],
  );

  return <EditorContent editor={editor} className="dk-ed-w" />;
}

const NO_MENTIONS: readonly AssistantEntityRef[] = [];
