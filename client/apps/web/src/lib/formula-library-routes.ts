import { appPath } from "@/lib/app-path";
import { recordPath } from "@/config/record-links";

/** Every route the library pages navigate to, spelled once. */
export const formulaLibraryRoutes = {
  list: appPath("/billing/configuration-files/formula-libraries"),
  new: appPath("/billing/configuration-files/formula-libraries/new"),
  edit: (libraryId: string) => recordPath("formula_library", libraryId),
} as const;
