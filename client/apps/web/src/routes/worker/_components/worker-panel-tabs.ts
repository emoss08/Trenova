export type EmploymentView = "details" | "history";

export type ResolvedWorkerPanelTab = {
  tab: string;
  employmentView: EmploymentView;
};

/**
 * Which tab the panel shows for a requested tab id. The employment history
 * used to be a tab of its own; links and concerns still say "timeline", and
 * that value lands on the Employment tab with the history view showing.
 */
export function resolveWorkerPanelTab(
  requested: string,
  employmentViewChoice: EmploymentView,
): ResolvedWorkerPanelTab {
  if (requested === "timeline") {
    return { tab: "employment", employmentView: "history" };
  }
  return { tab: requested, employmentView: employmentViewChoice };
}
