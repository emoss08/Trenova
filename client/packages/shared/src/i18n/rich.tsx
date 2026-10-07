// rich.tsx translates a sentence that carries markup as one message.
//
// "You've used <b>{0}</b> of this month's AI allowance" used to be written as two catalog
// entries around a <b>, and neither half can be translated: Chinese puts the amount in a
// different place, and a fragment like "of this month's AI allowance" gives a translator
// nothing to work with. A rich message keeps the sentence whole and names the markup with
// tags the translator moves along with the words; the caller says what each tag renders.
//
//   rt("Press <kbd/> to approve", { kbd: () => <Kbd>⌘↵</Kbd> })
//   rt("You've used <b>{0}</b> of this month's AI allowance", { b: (c) => <b>{c}</b> }, pct)
//
// Only the tags the caller declares are markup; any other "<" is text. A tag holds text and
// placeholders but no other tag, and a plural cannot span a tag — the pieces between tags
// are formatted on their own.
import { formatMessage } from "@trenova/shared/i18n/format-message";
import { DEFAULT_LOCALE, type Locale } from "@trenova/shared/i18n/generated/locales";
import { getCatalogVersion, getLocale, lookupIn, subscribe } from "@trenova/shared/i18n/runtime";
import { Fragment, type ReactNode, useCallback, useSyncExternalStore } from "react";

export type RichTag = (children: ReactNode) => ReactNode;

export type RichTags = Readonly<Record<string, RichTag>>;

export type RichTranslateFn = (message: string, tags: RichTags, ...args: unknown[]) => ReactNode;

type RichPiece = { tag: null; text: string } | { tag: string; text: string | null };

const TAG_NAME = /^[A-Za-z][\w-]*$/;

function escapeRegExp(text: string): string {
  return text.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

/**
 * parseRich splits a template at the declared tags, or returns null when the template uses
 * one of them wrongly — unclosed, nested or stray — so the caller can fall back rather than
 * render a half-tagged sentence.
 */
export function parseRich(template: string, tagNames: readonly string[]): RichPiece[] | null {
  const names = tagNames.filter((name) => TAG_NAME.test(name)).map(escapeRegExp);
  if (names.length === 0) return [{ tag: null, text: template }];

  const alternatives = names.join("|");
  const pattern = new RegExp(`<(${alternatives})\\s*/>|<(${alternatives})>([\\s\\S]*?)</\\2>`, "g");
  const stray = new RegExp(`</?(?:${alternatives})\\s*/?>`);

  const pieces: RichPiece[] = [];
  let last = 0;
  for (const match of template.matchAll(pattern)) {
    const before = template.slice(last, match.index);
    if (stray.test(before)) return null;
    if (before !== "") pieces.push({ tag: null, text: before });

    const inner = match[3];
    if (inner !== undefined && stray.test(inner)) return null;
    pieces.push(
      match[1] !== undefined ? { tag: match[1], text: null } : { tag: match[2], text: inner },
    );
    last = match.index + match[0].length;
  }

  const rest = template.slice(last);
  if (stray.test(rest)) return null;
  if (rest !== "") pieces.push({ tag: null, text: rest });
  return pieces;
}

function stripTags(template: string, tagNames: readonly string[]): string {
  const names = tagNames.filter((name) => TAG_NAME.test(name)).map(escapeRegExp);
  if (names.length === 0) return template;
  return template.replace(new RegExp(`</?(?:${names.join("|")})\\s*/?>`, "g"), "");
}

function renderPieces(
  locale: Locale,
  pieces: RichPiece[],
  tags: RichTags,
  args: unknown[],
): ReactNode {
  return pieces.map((piece, index) => {
    if (piece.tag === null) {
      return <Fragment key={index}>{formatMessage(locale, piece.text, args)}</Fragment>;
    }
    const children = piece.text === null ? null : formatMessage(locale, piece.text, args);
    return <Fragment key={index}>{tags[piece.tag](children)}</Fragment>;
  });
}

/**
 * renderRich renders a message in a locale. A translation whose tags do not parse falls back
 * to the English sentence with its markup, and that to the words alone, so a mistranslated
 * tag costs the formatting and never the sentence.
 */
export function renderRich(
  locale: Locale,
  message: string,
  tags: RichTags,
  args: unknown[],
): ReactNode {
  const names = Object.keys(tags);
  const translated = lookupIn(locale, message);

  const own = parseRich(translated, names);
  if (own !== null) return renderPieces(locale, own, tags, args);

  const source = parseRich(message, names);
  if (source !== null) return renderPieces(DEFAULT_LOCALE, source, tags, args);

  return formatMessage(locale, stripTags(translated, names), args);
}

/**
 * useRichT returns `rt`, the rich counterpart of `t`: it re-renders the component on a
 * language switch or a catalog bundle landing, exactly as useT does.
 */
export function useRichT(): RichTranslateFn {
  useSyncExternalStore(subscribe, getCatalogVersion, getCatalogVersion);
  const locale = useSyncExternalStore(subscribe, getLocale, getLocale);
  return useCallback(
    (message: string, tags: RichTags, ...args: unknown[]) =>
      renderRich(locale, message, tags, args),
    [locale],
  );
}
