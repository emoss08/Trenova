import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Icon } from "@/components/ui/icons";
import { type Shipment } from "@/types/shipment";
import { faEllipsisVertical } from "@fortawesome/pro-regular-svg-icons";
import { useState } from "react";
import { ShipmentCancellationDialog } from "../../cancellation/shipment-cancellatioin-dialog";

// const menuSections = [
//   {
//     label: "General Actions",
//     items: [
//       {
//         title: "Assign",
//         description: "Assign this shipment to a worker(s).",
//         onClick: () => {
//           /* handle click */
//         },
//       },
//       {
//         title: "Edit",
//         description: "Modify shipment details.",
//         onClick: () => {
//           /* handle click */
//         },
//       },
//       {
//         title: "Duplicate",
//         description: "Create a copy of this shipment.",
//         onClick: () => {
//           /* handle click */
//         },
//       },
//       {
//         title: "Cancel",
//         description: "Cancel this shipment and update its status.",
//         onClick: () => {
//           /* handle click */
//         },
//       },
//     ],
//   },
//   {
//     label: "Management Actions",
//     items: [
//       {
//         title: "Split Shipment",
//         description: "Divide this shipment into multiple parts.",
//         onClick: () => {
//           /* handle click */
//         },
//       },
//       {
//         title: "Merge Shipment",
//         description: "Combine multiple shipments into one.",
//         onClick: () => {
//           /* handle click */
//         },
//       },
//       {
//         title: "Send to Worker",
//         description: "Assign this shipment for processing.",
//         onClick: () => {
//           /* handle click */
//         },
//       },
//     ],
//   },
//   {
//     label: "Documentation & Communication",
//     items: [
//       {
//         title: "Add Document(s)",
//         description: "Attach relevant documents to this shipment.",
//         onClick: () => {
//           /* handle click */
//         },
//       },
//       {
//         title: "Add Comment(s)",
//         description: "Leave internal notes or comments on this shipment.",
//         onClick: () => {
//           /* handle click */
//         },
//       },
//     ],
//   },
//   {
//     label: "View Actions",
//     items: [
//       {
//         title: "View Documents",
//         description: "Review attached shipment documents.",
//         onClick: () => {
//           /* handle click */
//         },
//       },
//       {
//         title: "View Comments",
//         description: "Check comments and notes related to this shipment.",
//         onClick: () => {
//           /* handle click */
//         },
//       },
//       {
//         title: "View Audit Log",
//         description: "Track all modifications and updates to this shipment.",
//         onClick: () => {
//           /* handle click */
//         },
//       },
//     ],
//   },
// ];

// type MenuSection = {
//   label: string;
//   items: {
//     title: string;
//     description: string;
//     onClick?: () => void;
//     disabled?: boolean;
//   }[];
// };

// type StickySectionDropdownProps = {
//   icon?: IconDefinition;
//   sections: MenuSection[];
//   align?: "start" | "end" | "center";
//   className?: string;
// };

export function ShipmentActions({ shipment }: { shipment: Shipment }) {
  const [cancellationDialogOpen, setCancellationDialogOpen] =
    useState<boolean>(false);

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" size="sm" className="p-2">
            <Icon icon={faEllipsisVertical} className="size-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start">
          {/* General Actions */}
          <DropdownMenuLabel>General Actions</DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            title="Assign"
            description="Assign this shipment to a worker(s)."
          />
          <DropdownMenuItem
            title="Edit"
            description="Modify shipment details."
          />
          <DropdownMenuItem
            title="Duplicate"
            description="Create a copy of this shipment."
          />
          <DropdownMenuItem
            title="Cancel"
            description="Cancel this shipment and update its status."
            onClick={() => setCancellationDialogOpen(true)}
          />

          {/* Management Actions */}
          <DropdownMenuLabel>Management Actions</DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            title="Split Shipment"
            description="Divide this shipment into multiple parts."
          />
          <DropdownMenuItem
            title="Merge Shipment"
            description="Combine multiple shipments into one."
          />
          <DropdownMenuItem
            title="Send to Worker"
            description="Assign this shipment for processing."
          />

          {/* Documentation & Communication */}
          <DropdownMenuLabel>Documentation & Communication</DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            title="Add Document(s)"
            description="Attach relevant documents to this shipment."
          />
          <DropdownMenuItem
            title="Add Comment(s)"
            description="Leave internal notes or comments on this shipment."
          />

          {/* View Actions */}
          <DropdownMenuLabel>View Actions</DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            title="View Documents"
            description="Review attached shipment documents."
          />
          <DropdownMenuItem
            title="View Comments"
            description="Check comments and notes related to this shipment."
          />
          <DropdownMenuItem
            title="View Audit Log"
            description="Track all modifications and updates to this shipment."
          />
        </DropdownMenuContent>
      </DropdownMenu>
      <ShipmentCancellationDialog
        open={cancellationDialogOpen}
        onOpenChange={setCancellationDialogOpen}
        shipmentId={shipment.id ?? ""}
      />
    </>
  );
}
