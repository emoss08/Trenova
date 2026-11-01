import type { SelectOption } from "@/lib/graphql/select-options";
import { selectOptionMetaString } from "@/lib/select-option-meta";
import type { DotHazmatReference } from "@/types/dot-hazmat-reference";

/** A DOT hazmat table entry as its `DOT_HAZMAT_REFERENCE` select option carries it. */
export function dotHazmatReferenceFromOption(option: SelectOption): DotHazmatReference {
  return {
    id: option.id,
    unNumber: selectOptionMetaString(option, "unNumber"),
    properShippingName: selectOptionMetaString(option, "properShippingName") || option.label,
    hazardClass: selectOptionMetaString(option, "hazardClass"),
    subsidiaryHazard: selectOptionMetaString(option, "subsidiaryHazard"),
    packingGroup: selectOptionMetaString(option, "packingGroup"),
    specialProvisions: selectOptionMetaString(option, "specialProvisions"),
    ergGuide: selectOptionMetaString(option, "ergGuide"),
    symbols: selectOptionMetaString(option, "symbols"),
  };
}
