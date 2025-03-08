import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { faChevronLeft } from "@fortawesome/pro-regular-svg-icons";
import { AuditActions } from "./audit-actions";

type AuditLogHeaderProps = {
  onBack: () => void;
  onExport: () => void;
};

export function AuditLogHeader({ onBack, onExport }: AuditLogHeaderProps) {
  return (
    <div className="flex items-center px-4 justify-between">
      <HeaderBackButton onBack={onBack} />
      <AuditActions onExport={onExport} />
    </div>
  );
}

function HeaderBackButton({ onBack }: { onBack: () => void }) {
  return (
    <Button variant="outline" size="sm" onClick={onBack}>
      <Icon icon={faChevronLeft} className="size-4" />
      <span className="text-sm">Back</span>
    </Button>
  );
}
