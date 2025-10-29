import { ParserMap, useQueryState } from "nuqs";

/**
 * A hook that returns the query state and setter function for a specific key
 * from a ParserMap.
 * @see https://x.com/nuqs47ng/status/1983145777804271877
 * @param parsers - The ParserMap to use.
 * @param key - The key to use from the ParserMap.
 * @returns The query state and setter function for the specified key.
 * @example
 * const [value, setValue] = usePickedQueryState(searchParamsParser, "query");
 */
export function usePickedQueryState<Parsers extends ParserMap>(
  parsers: Parsers,
  key: keyof Parsers,
) {
  return useQueryState(String(key), parsers[key]);
}
