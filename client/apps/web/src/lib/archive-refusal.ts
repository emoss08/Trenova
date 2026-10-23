import { ApiRequestError } from "@trenova/shared/lib/api";

/**
 * Why the server would not archive templates or libraries: the person has to confirm, having
 * seen what still prices with them, or is not allowed to (archiving one in
 * use needs approve permission), in the server's words either way.
 */
export type ArchiveRefusal =
  | { kind: "needsConfirmation"; message: string }
  | { kind: "forbidden"; message: string };

export const CONFIRM_IN_USE_FIELD = "confirmInUse";

export function archiveRefusal(error: unknown): ArchiveRefusal | null {
  if (!(error instanceof ApiRequestError)) return null;

  const confirmation = error.getFieldError(CONFIRM_IN_USE_FIELD);
  if (confirmation) {
    return { kind: "needsConfirmation", message: confirmation.message };
  }

  if (error.isAuthorizationError()) {
    return { kind: "forbidden", message: error.normalize().message };
  }

  return null;
}
