import {
  TOOL_NAMESPACE_QUALIFIERS,
  TOOL_NAMESPACE_RECORDS,
  TOOL_NAMESPACES,
  type ToolNamespace,
} from "@trenova/shared/types/generated/tool-namespaces";

const NAMESPACES: ReadonlySet<string> = new Set(TOOL_NAMESPACES);

/**
 * A tool's name with its verb first, the way the words are read: an agent tool is named
 * `<namespace>_<verb>[_<object>]`, so `shipment_place_hold` reads as `place_shipment_hold`,
 * a bare `<namespace>_list` lists that record (`shipment_list` reads as `list_shipments`), and a
 * namespace's qualifier reads before it (`settlement_get_driver` reads as
 * `get_driver_settlement`), and a record the namespace names on its own reads without it
 * (`receivable_get_bank_receipt` reads as `get_bank_receipt`). A name outside a namespace is
 * returned as it is.
 */
export function toolNameInReadingOrder(name: string): string {
  const [namespace, verb, ...object] = name.split("_");
  if (verb === undefined || !NAMESPACES.has(namespace)) {
    return name;
  }
  if (opensWithRecord(namespace as ToolNamespace, object)) {
    return [verb, ...object].join("_");
  }
  const qualifiers = TOOL_NAMESPACE_QUALIFIERS[namespace as ToolNamespace] ?? [];
  const qualified = object.length > 0 && qualifiers.includes(object[0]);
  const noun = qualified ? `${object[0]}_${namespace}` : namespace;
  const rest = qualified ? object.slice(1) : object;
  if (verb === "list" && rest.length === 0) {
    return `list_${plural(noun)}`;
  }

  return [verb, noun, ...rest].join("_");
}

function opensWithRecord(namespace: ToolNamespace, object: readonly string[]): boolean {
  const joined = object.join("_");
  return (TOOL_NAMESPACE_RECORDS[namespace] ?? []).some((record) =>
    [record, plural(record)].some((form) => joined === form || joined.startsWith(`${form}_`)),
  );
}

const IRREGULAR_PLURALS: Readonly<Record<string, string>> = {
  person: "people",
  child: "children",
  criterion: "criteria",
  matrix: "matrices",
  vertex: "vertices",
  appendix: "appendices",
  analysis: "analyses",
  basis: "bases",
  datum: "data",
};

const IRREGULAR_SINGULARS: ReadonlySet<string> = new Set(Object.values(IRREGULAR_PLURALS));

/** The plural of a snake_case noun's last word, as the server's stringutils.PluralNoun forms it. */
function plural(noun: string): string {
  const cut = noun.lastIndexOf("_") + 1;
  const head = noun.slice(0, cut);
  const last = noun.slice(cut);
  if (last === "") {
    return noun;
  }
  const irregular = IRREGULAR_PLURALS[last];
  if (irregular !== undefined) {
    return head + irregular;
  }
  if (IRREGULAR_SINGULARS.has(last)) {
    return noun;
  }
  if (/[^aeiou]y$/u.test(last)) {
    return `${head}${last.slice(0, -1)}ies`;
  }

  return /(?:s|x|z|ch|sh)$/u.test(last) ? `${noun}es` : `${noun}s`;
}
