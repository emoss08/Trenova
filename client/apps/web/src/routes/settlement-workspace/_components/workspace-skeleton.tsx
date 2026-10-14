import { SettlementWorkspaceSkeleton } from "@/components/settlements/settlement-workspace-skeleton";

const ACTION_WIDTHS = ["w-20", "w-20", "w-36"] as const;

/** The driver workspace: six figures, and Refresh, Pay now and Generate settlements. */
export function WorkspaceSkeleton() {
  return <SettlementWorkspaceSkeleton figures={6} actions={ACTION_WIDTHS} />;
}
