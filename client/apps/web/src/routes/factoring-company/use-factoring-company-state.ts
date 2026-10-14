import { parseAsString, parseAsStringLiteral } from "nuqs";

export const FACTORING_COMPANY_SECTIONS = ["agreements", "notices"] as const;

/** The agreement or notice a record link opened, and which section of its factor it sits in. */
export const factoringCompanyRecordSearchParamsParser = {
  section: parseAsStringLiteral(FACTORING_COMPANY_SECTIONS),
  agreement: parseAsString,
  noa: parseAsString,
};
