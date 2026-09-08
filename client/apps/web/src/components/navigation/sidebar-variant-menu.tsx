import {
  DropdownMenuPortal,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import {
  isSidebarVariant,
  useNavigationStore,
  type SidebarVariant,
} from "@/stores/navigation-store";
import { PanelsTopLeftIcon } from "lucide-react";

interface SidebarVariantOption {
  value: SidebarVariant;
  label: string;
  description: string;
}

export const SIDEBAR_VARIANT_OPTIONS: readonly SidebarVariantOption[] = [
  {
    value: "workspace",
    label: "Workspace",
    description: "Modules in the header, the current area in the sidebar",
  },
  {
    value: "classic",
    label: "Classic",
    description: "Every section stacked in one column",
  },
];

/**
 * Lives inside the user menu so the previous layout stays one click away.
 */
export function SidebarLayoutSubmenu() {
  const variant = useNavigationStore((state) => state.sidebarVariant);
  const setVariant = useNavigationStore((state) => state.setSidebarVariant);

  return (
    <DropdownMenuSub>
      <DropdownMenuSubTrigger>
        <PanelsTopLeftIcon className="mr-2 size-4" />
        <span>Sidebar layout</span>
      </DropdownMenuSubTrigger>
      <DropdownMenuPortal>
        <DropdownMenuSubContent sideOffset={5} className="w-64">
          <DropdownMenuRadioGroup
            value={variant}
            onValueChange={(value) => {
              if (isSidebarVariant(value)) {
                setVariant(value);
              }
            }}
          >
            {SIDEBAR_VARIANT_OPTIONS.map((option) => (
              <DropdownMenuRadioItem
                key={option.value}
                value={option.value}
                className="cursor-pointer"
              >
                <span className="flex flex-col">
                  <span className="text-sm">{option.label}</span>
                  <span className="text-2xs text-muted-foreground">{option.description}</span>
                </span>
              </DropdownMenuRadioItem>
            ))}
          </DropdownMenuRadioGroup>
        </DropdownMenuSubContent>
      </DropdownMenuPortal>
    </DropdownMenuSub>
  );
}
