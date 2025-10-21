/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Extension } from "@tiptap/core";
import { Plugin, PluginKey } from "@tiptap/pm/state";

export interface SlashCommandOptions {
  char: string;
  allowSpaces: boolean;
  startOfLine: boolean;
}

export const SlashCommandExtension = Extension.create<SlashCommandOptions>({
  name: "slashCommand",

  addOptions() {
    return {
      char: "/",
      allowSpaces: false,
      startOfLine: true,
    };
  },

  addProseMirrorPlugins() {
    return [
      new Plugin({
        key: new PluginKey("slashCommand"),
        props: {
          handleTextInput(view, _from, _to, text) {
            const { state } = view;
            const { $from } = state.selection;

            const textBefore = $from.parent.textBetween(
              0,
              $from.parentOffset,
              "\n",
              " ",
            );

            if (
              text === "/" &&
              (textBefore === "" || textBefore.endsWith(" "))
            ) {
              return false;
            }

            return false;
          },
        },
      }),
    ];
  },
});
