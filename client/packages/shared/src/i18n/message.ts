// message.ts renders a message that arrived as data rather than as a literal in this code.
//
// The rules module (docs/engineering/rules-engine.md) answers with user-facing text as
// `{ message, args }`: an English ICU template that is also the catalog key, and its positional
// arguments. `task i18n` extracts those templates from the crate's `message!` literals, so the
// catalogs hold them even though no `t("…")` in TypeScript names them. translateMessage is the
// one way to show such a message: it looks the template up and fills the arguments, exactly as
// `t(template, ...args)` would, and the i18n check (`i18n/tools/fragments.mjs`) reports any
// other spelling of it.
//
// An argument is text, a count (which a plural selects on) or a phrase that is itself a
// message, such as a band label inside a coverage warning. A phrase is translated first and
// substituted as text, so every part of the sentence reads in the same language.
import { translate, type TranslateFn } from "@trenova/shared/i18n/runtime";

/** One positional argument: text, a count, or a nested message translated before it is used. */
export type CatalogMessageArg = string | number | CatalogMessage;

/** A catalog template and its positional arguments, `{0}` first. */
export type CatalogMessage = {
  readonly message: string;
  readonly args?: readonly CatalogMessageArg[];
};

/**
 * Translates a message that came from a catalog-backed source, translating nested messages in
 * its arguments first. Inside a component pass the `t` from `useT()`, so the text follows the
 * locale React renders with; elsewhere it reads the active locale, like `translate`.
 */
export function translateMessage(message: CatalogMessage, t: TranslateFn = translate): string {
  const args = (message.args ?? []).map((arg) =>
    typeof arg === "object" && arg !== null ? translateMessage(arg, t) : arg,
  );
  // i18n-ignore: the one place a catalog message is spread, after its nested messages are translated
  return t(message.message, ...args);
}
