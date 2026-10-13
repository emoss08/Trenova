import type { SelectOption } from "@/lib/graphql/select-options";

/**
 * A document type's name as a person reads it. The select options carry the
 * type's code as the label ("RATECON") and its name as the description
 * ("Rate confirmation"); the name is what Triage shows and matches on.
 */
export function documentTypeName(option: Pick<SelectOption, "label" | "description">): string {
  const name = option.description?.trim() ?? "";

  return name === "" ? option.label : name;
}

/** The document type whose name or code is `name`, ignoring case; null when none is. */
export function documentTypeNamed(
  options: readonly SelectOption[],
  name: string,
): SelectOption | null {
  const wanted = name.trim().toLowerCase();
  if (wanted === "") {
    return null;
  }

  return (
    options.find((option) => documentTypeName(option).trim().toLowerCase() === wanted) ??
    options.find((option) => option.label.trim().toLowerCase() === wanted) ??
    null
  );
}
