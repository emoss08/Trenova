import { useCreateReportDashboard } from "@/hooks/use-reports";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { useNavigate } from "react-router";
import { toast } from "sonner";

/**
 * Creating a dashboard is offered from the toolbar and from the empty state,
 * so the mutation, the toast, and the hand-off into the editor live here
 * rather than being written twice.
 */
export function useCreateDashboardAction() {
  const navigate = useNavigate();
  const createDashboard = useCreateReportDashboard();

  const create = () => {
    createDashboard.mutate(
      { name: "Untitled dashboard", layout: { tiles: [] } },
      {
        onSuccess: (created) => {
          void navigate(`/reports/dashboards/${created.id}`);
        },
        onError: (error) =>
          toast.error(graphQLErrorMessage(error, "Failed to create the dashboard")),
      },
    );
  };

  return { create, isPending: createDashboard.isPending };
}
