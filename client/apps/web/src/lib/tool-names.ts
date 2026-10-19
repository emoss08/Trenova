import { TOOL_NAMESPACES } from "@trenova/shared/types/generated/tool-namespaces";

const NAMESPACES: ReadonlySet<string> = new Set(TOOL_NAMESPACES);

/**
 * A tool's name with its verb first, the way the words are read: an agent tool is named
 * `<namespace>_<verb>[_<object>]`, so `shipment_place_hold` reads as `place_shipment_hold`,
 * and a bare `<namespace>_list` lists that record: `shipment_list` reads as `list_shipments`.
 * A name outside a namespace is returned as it is.
 */
export function toolNameInReadingOrder(name: string): string {
  const [namespace, verb, ...rest] = name.split("_");
  if (verb === undefined || !NAMESPACES.has(namespace)) {
    return name;
  }
  if (verb === "list" && rest.length === 0) {
    return `list_${plural(namespace)}`;
  }

  return [verb, namespace, ...rest].join("_");
}

function plural(noun: string): string {
  if (/[^aeiou]y$/u.test(noun)) {
    return `${noun.slice(0, -1)}ies`;
  }

  return noun.endsWith("s") ? `${noun}es` : `${noun}s`;
}
