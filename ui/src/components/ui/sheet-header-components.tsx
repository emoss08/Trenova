/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

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
