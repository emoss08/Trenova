import { FormCreatePanel } from "@/components/form-create-panel";
import { TabbedFormEditPanel } from "@/components/tabbed-form-edit-panel";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { apiService } from "@/services/api";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { createOrder, updateOrder } from "@/lib/graphql/order";
import { orderSchema, type Order } from "@trenova/shared/types/order";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { HistoryIcon } from "lucide-react";
import { lazy } from "react";
import { useForm } from "react-hook-form";
import { OrderForm } from "./order-form";

const AuditTab = lazy(() => import("@/components/audit-tab"));

function OwnerDisplay({ ownerId }: { ownerId?: string | null }) {
  const { data: owner, isLoading } = useQuery({
    queryKey: ["user", ownerId],
    queryFn: () => apiService.userService.get(ownerId!),
    enabled: !!ownerId,
    staleTime: 5 * 60 * 1000,
  });

  if (isLoading) return <Skeleton className="h-2.5 w-34" />;

  if (ownerId) {
    return (
      <div className="flex items-center gap-1">
        <span className="text-2xs text-muted-foreground">Owner:</span>
        <span className="text-2xs text-blue-500">{owner?.name}</span>
      </div>
    );
  }

  return <span className="text-2xs text-foreground">No owner assigned</span>;
}

export function OrderPanel({ open, onOpenChange, mode, row }: DataTablePanelProps<Order>) {
  const queryClient = useQueryClient();
  const form = useForm({
    resolver: zodResolver(orderSchema),
    defaultValues: {
      customerId: "",
      ownerId: "",
      status: "Draft",
      poNumber: "",
      bol: "",
      currencyCode: "USD",
      quotedAmount: null,
      baseAmount: null,
      totalAmount: null,
    },
    mode: "onChange",
  });

  if (mode === "edit") {
    return (
      <TabbedFormEditPanel
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        form={form}
        queryKey="order-list"
        title="Order"
        fieldKey="orderNumber"
        formComponent={<OrderForm mode="edit" />}
        mutationFn={async (values, currentRow) => {
          const updated = await updateOrder(currentRow.id!, values);
          void queryClient.invalidateQueries({ queryKey: ["order-detail"] });
          return updated;
        }}
        descriptionExtra={<OwnerDisplay ownerId={row?.ownerId} />}
        tabs={[
          {
            value: "history",
            label: "History",
            icon: HistoryIcon,
            hideFooter: true,
            content: AuditTab,
            contentProps: { resourceId: row?.id },
          },
        ]}
        useDock
      />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      queryKey="order-list"
      title="Order"
      formComponent={<OrderForm mode="create" />}
      mutationFn={(values) => createOrder(values)}
    />
  );
}
