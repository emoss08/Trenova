import { addErrorSurface } from "@/lib/form-errors";
import { useEffect } from "react";

/**
 * Declares, while the calling component is mounted, that it shows every error of the form
 * under `prefix` (such as `moves.0`), so a server error there is set where it shows it rather
 * than sent to the form root. See `addErrorSurface`.
 */
export function useErrorSurface(control: object, prefix: string): void {
  useEffect(() => addErrorSurface(control, prefix), [control, prefix]);
}
