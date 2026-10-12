import { useQueryState, type ParserBuilder } from "nuqs";
import { useCallback } from "react";

/**
 * A dialog the address opens by a key alone. Closing removes the key rather
 * than writing false, so a closed dialog leaves nothing in the link.
 */
export function useAddressedFlag(
  key: string,
  parser: ParserBuilder<boolean>,
): readonly [boolean, (open: boolean) => void] {
  const [value, setValue] = useQueryState(key, parser);
  const setOpen = useCallback((open: boolean) => void setValue(open ? true : null), [setValue]);
  return [value === true, setOpen] as const;
}
