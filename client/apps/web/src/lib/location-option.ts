import type { SelectOption } from "@/lib/graphql/select-options";
import { selectOptionMetaString } from "@/lib/select-option-meta";

/** A location as its `LOCATION` select option carries it: its name and its address. */
export type LocationOptionPlace = {
  name: string;
  addressLine1: string;
  city: string;
  stateAbbreviation: string;
  postalCode: string;
};

export function locationOptionPlace(option: SelectOption): LocationOptionPlace {
  return {
    name: option.label,
    addressLine1: selectOptionMetaString(option, "addressLine1"),
    city: selectOptionMetaString(option, "city"),
    stateAbbreviation: selectOptionMetaString(option, "stateAbbreviation"),
    postalCode: selectOptionMetaString(option, "postalCode"),
  };
}
