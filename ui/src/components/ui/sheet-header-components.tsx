import { faChevronLeft } from "@fortawesome/pro-solid-svg-icons";
import { Button } from "./button";
import { Icon } from "./icons";

export function HeaderBackButton({ onBack }: { onBack: () => void }) {
  return (
    <Button variant="outline" size="sm" onClick={onBack}>
      <Icon icon={faChevronLeft} className="size-4" />
      <span className="text-sm">Back</span>
    </Button>
  );
}
