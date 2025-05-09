import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Icon } from "@/components/ui/icons";
import {
  faEllipsisVertical,
  faFileCsv,
  faFileHalfDashed,
} from "@fortawesome/pro-regular-svg-icons";

export function AuditActions({ onExport }: { onExport: () => void }) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="icon" className="p-2">
          <Icon icon={faEllipsisVertical} className="size-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent side="bottom" align="end">
        <DropdownMenuItem
          title="Export to CSV"
          description="Export the audit log as a CSV file."
          startContent={<Icon icon={faFileCsv} className="size-4" />}
        />
        <DropdownMenuItem
          title="Export to JSON"
          description="Export the audit log as a JSON file."
          startContent={<Icon icon={faFileHalfDashed} className="size-4" />}
          onClick={onExport}
        />
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
