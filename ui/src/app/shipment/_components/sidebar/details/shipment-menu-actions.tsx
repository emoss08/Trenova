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
import { ScrollArea } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";
import {
  faEllipsisVertical,
  IconDefinition,
} from "@fortawesome/pro-regular-svg-icons";

const menuSections = [
  {
    label: "General Actions",
    items: [
      {
        title: "Assign",
        description: "Assign this shipment to a worker(s).",
        onClick: () => {
          /* handle click */
        },
      },
      {
        title: "Edit",
        description: "Modify shipment details.",
        onClick: () => {
          /* handle click */
        },
      },
      {
        title: "Duplicate",
        description: "Create a copy of this shipment.",
        onClick: () => {
          /* handle click */
        },
      },
      {
        title: "Cancel",
        description: "Cancel this shipment and update its status.",
        onClick: () => {
          /* handle click */
        },
      },
    ],
  },
  {
    label: "Management Actions",
    items: [
      {
        title: "Split Shipment",
        description: "Divide this shipment into multiple parts.",
        onClick: () => {
          /* handle click */
        },
      },
      {
        title: "Merge Shipment",
        description: "Combine multiple shipments into one.",
        onClick: () => {
          /* handle click */
        },
      },
      {
        title: "Send to Worker",
        description: "Assign this shipment for processing.",
        onClick: () => {
          /* handle click */
        },
      },
    ],
  },
  {
    label: "Documentation & Communication",
    items: [
      {
        title: "Add Document(s)",
        description: "Attach relevant documents to this shipment.",
        onClick: () => {
          /* handle click */
        },
      },
      {
        title: "Add Comment(s)",
        description: "Leave internal notes or comments on this shipment.",
        onClick: () => {
          /* handle click */
        },
      },
    ],
  },
  {
    label: "View Actions",
    items: [
      {
        title: "View Documents",
        description: "Review attached shipment documents.",
        onClick: () => {
          /* handle click */
        },
      },
      {
        title: "View Comments",
        description: "Check comments and notes related to this shipment.",
        onClick: () => {
          /* handle click */
        },
      },
      {
        title: "View Audit Log",
        description: "Track all modifications and updates to this shipment.",
        onClick: () => {
          /* handle click */
        },
      },
    ],
  },
];

type MenuSection = {
  label: string;
  items: {
    title: string;
    description: string;
    onClick?: () => void;
    disabled?: boolean;
  }[];
};

type StickySectionDropdownProps = {
  icon?: IconDefinition;
  sections: MenuSection[];
  align?: "start" | "end" | "center";
  className?: string;
};

export function ShipmentDropdownMenu({
  icon,
  sections,
  align = "start",
  className,
}: StickySectionDropdownProps) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="sm" className={cn("p-2", className)}>
          {icon && <Icon icon={icon} className="size-4" />}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align={align} className="p-0">
        <ScrollArea className="flex max-h-[400px] 2xl:max-h-[100vh] flex-col overflow-y-auto px-1">
          <div className="relative">
            {sections.map((section, sectionIndex) => (
              <div key={section.label} className="relative">
                <div className="sticky top-0 z-10 bg-background">
                  <DropdownMenuLabel>{section.label}</DropdownMenuLabel>
                  <DropdownMenuSeparator />
                </div>
                {section.items.map((item, index) => (
                  <DropdownMenuItem
                    key={`${section.label}-${item.title}-${index}`}
                    onClick={item.onClick}
                    disabled={item.disabled}
                    className="flex flex-col items-start"
                    title={item.title}
                    description={item.description}
                  />
                ))}
                {sectionIndex < sections.length - 1 && <div className="h-2" />}
              </div>
            ))}
          </div>
        </ScrollArea>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function ShipmentDetailActions() {
  return (
    <ShipmentDropdownMenu
      icon={faEllipsisVertical}
      sections={menuSections}
      align="start"
    />
  );
}
