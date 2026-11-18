import { supportAccess } from "@/lib/queries/support-access";
import { useQuery } from "@tanstack/react-query";

/** What to do with a save the screen makes on its own: send it, keep it in memory, or wait. */
export type BackgroundWrites = "send" | "keep" | "wait";

/**
 * Whether the session may save what the screen keeps on its own (a table's layout, the last
 * view a person had open). A read-only Trenova support session may not: the server refuses every
 * change it makes, and a refusal for something the staff member never asked to change is noise,
 * so such a save is kept in memory for the session instead. A change the person asks for is still
 * sent, so its refusal says why. Until the session is known a save waits; if it cannot be known,
 * saves are sent as they always were.
 */
export function useBackgroundWrites(): BackgroundWrites {
  const { data, isSuccess, isError } = useQuery({
    ...supportAccess.currentSession(),
    staleTime: Number.POSITIVE_INFINITY,
  });
  if (isError) {
    return "send";
  }
  if (!isSuccess) {
    return "wait";
  }
  return data?.scope === "ReadOnly" ? "keep" : "send";
}
