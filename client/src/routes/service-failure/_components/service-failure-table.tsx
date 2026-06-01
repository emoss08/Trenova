import { DataTable } from "@/components/data-table/data-table";
import { usePermission } from "@/hooks/use-permission";
import { apiService } from "@/services/api";
import type { RowAction } from "@/types/data-table";
import { Operation, Resource } from "@/types/permission";
import type { ServiceFailure } from "@/types/service-failure";
import { useQueryClient } from "@tanstack/react-query";
import type { Row } from "@tanstack/react-table";
import { ArchiveIcon, CheckCircle2Icon, ClipboardIcon, ShieldCheckIcon } from "lucide-react";
import { toast } from "sonner";
import { getColumns } from "./service-failure-columns";
import { ServiceFailurePanel } from "./service-failure-panel";

type LifecycleAction = "review" | "resolve" | "void";

export default function ServiceFailureTable() {
  const queryClient = useQueryClient();
  const canApprove = usePermission(Resource.ServiceFailure, Operation.Approve);
  const canArchive = usePermission(Resource.ServiceFailure, Operation.Archive);
  const canExport = usePermission(Resource.ServiceFailure, Operation.Export);
  const columns = getColumns();

  const invalidate = (shipmentId?: string) => {
    void queryClient.invalidateQueries({ queryKey: ["service-failure-list"] });
    if (shipmentId) {
      void queryClient.invalidateQueries({
        queryKey: ["serviceFailure", "list-by-shipment", shipmentId],
      });
    }
  };

  const handleLifecycle = async (row: Row<ServiceFailure>, action: LifecycleAction) => {
    const entity = row.original;
    const payload = {
      shipmentId: entity.shipmentId,
      version: entity.version ?? 0,
    };

    if (action === "review") {
      await apiService.serviceFailureService.review(entity.id ?? "", payload);
      toast.success("Service failure reviewed");
    } else if (action === "resolve") {
      await apiService.serviceFailureService.resolve(entity.id ?? "", payload);
      toast.success("Service failure resolved");
    } else {
      await apiService.serviceFailureService.void(entity.id ?? "", payload);
      toast.success("Service failure voided");
    }
    invalidate(entity.shipmentId);
  };

  const handleBuildEDI = async (row: Row<ServiceFailure>) => {
    const result = await apiService.serviceFailureService.buildEDI214Payload(row.original.id ?? "");
    await navigator.clipboard?.writeText(JSON.stringify(result.payload, null, 2));
    const diagnostics = result.diagnostics.length;
    toast.success("EDI 214 payload generated", {
      description: diagnostics
        ? `${diagnostics} diagnostic item(s); payload copied.`
        : "Payload copied to clipboard.",
    });
  };

  const contextMenuActions: RowAction<ServiceFailure>[] = [
    {
      id: "review",
      label: "Review",
      icon: ShieldCheckIcon,
      onClick: (row) => void handleLifecycle(row, "review"),
      hidden: (row) => !canApprove.allowed || row.original.status !== "Open",
    },
    {
      id: "resolve",
      label: "Resolve",
      icon: CheckCircle2Icon,
      onClick: (row) => void handleLifecycle(row, "resolve"),
      hidden: (row) => row.original.status === "Resolved" || row.original.status === "Voided",
    },
    {
      id: "void",
      label: "Void",
      icon: ArchiveIcon,
      variant: "destructive",
      onClick: (row) => void handleLifecycle(row, "void"),
      hidden: (row) => !canArchive.allowed || row.original.status === "Voided",
    },
    {
      id: "edi-214-payload",
      label: "Build EDI 214 Payload",
      icon: ClipboardIcon,
      onClick: (row) => void handleBuildEDI(row),
      hidden: () => !canExport.allowed,
    },
  ];

  return (
    <DataTable<ServiceFailure>
      name="Service Failure"
      link="/service-failures/"
      queryKey="service-failure-list"
      exportModelName="service-failure"
      resource={Resource.ServiceFailure}
      columns={columns}
      contextMenuActions={contextMenuActions}
      TablePanel={ServiceFailurePanel}
      enableCreateAction={false}
      preferDetailRowForEdit
    />
  );
}
