import { SettlementWorkspaceSkeleton } from "@/components/settlements/settlement-workspace-skeleton";

const ACTION_WIDTHS = ["w-20", "w-36"] as const;

/** The carrier workspace: five figures, and Refresh and Generate settlements. */
export function WorkspaceSkeleton() {
  return <SettlementWorkspaceSkeleton figures={5} actions={ACTION_WIDTHS} />;
}
