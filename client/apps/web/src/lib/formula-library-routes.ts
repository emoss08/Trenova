import { recordPath } from "@/config/record-links";

const BASE = "/billing/configuration-files/formula-libraries";

/** Every route the library pages navigate to, spelled once. */
export const formulaLibraryRoutes = {
  list: BASE,
  new: `${BASE}/new`,
  edit: (libraryId: string) => recordPath("formula_library", libraryId),
} as const;
