import { Button } from "@trenova/shared/components/ui/button";
import { ButtonGroup } from "@trenova/shared/components/ui/button-group";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { useT } from "@trenova/shared/i18n/use-t";
import { CheckIcon, ChevronDownIcon } from "@trenova/shared/components/icons";
import { Kbd, kbdOnSolidClassName } from "@trenova/shared/components/ui/kbd";
import { cn } from "@trenova/shared/lib/utils";

export type SplitButtonOption<T extends string = string> = {
  id: T;
  label: string;
  description?: string;
};

type SplitButtonProps<T extends string = string> = {
  options: SplitButtonOption<T>[];
  selectedOption: T;
  onOptionSelect: (optionId: T) => void;
  isLoading?: boolean;
  loadingText?: string;
  disabled?: boolean;
  className?: string;
  formId?: string;
  /** The keys that press the main button, shown inside it, such as "Ctrl+S". */
  shortcut?: string;
  /** The fill: the inverted accent, or the primary colour for a form's save. */
  tone?: "invert" | "primary";
};

const PRIMARY_TONE = "bg-primary text-primary-foreground hover:bg-primary/90 hover:text-primary-foreground";

export function SplitButton<T extends string = string>({
  options,
  selectedOption,
  onOptionSelect,
  isLoading = false,
  loadingText,
  disabled = false,
  className,
  formId,
  shortcut,
  tone = "invert",
}: SplitButtonProps<T>) {
  const t = useT();

  const selected = options.find((opt) => opt.id === selectedOption);
  const otherOptions = options.filter((opt) => opt.id !== selectedOption);

  return (
    <ButtonGroup className={className}>
      <Button
        type="submit"
        form={formId}
        isLoading={isLoading}
        loadingText={loadingText}
        disabled={disabled}
        variant="ghostInvert"
        className={cn(
          "border-r border-r-brand-foreground/10",
          tone === "primary" && cn(PRIMARY_TONE, "border-r-primary-foreground/15"),
        )}
      >
        {selected?.label}
        {shortcut && (
          <Kbd aria-hidden className={cn(kbdOnSolidClassName, "ml-1")}>
            {shortcut}
          </Kbd>
        )}
      </Button>
      <DropdownMenu>
        <DropdownMenuTrigger
          disabled={disabled || isLoading}
          render={
            <Button
              variant="ghostInvert"
              type="button"
              disabled={disabled || isLoading}
              className={tone === "primary" ? PRIMARY_TONE : undefined}
            >
              <ChevronDownIcon className="size-4" />
            </Button>
          }
        />
        <DropdownMenuContent align="end" sideOffset={8}>
          {otherOptions.map((option) => (
            <DropdownMenuItem
              key={option.id}
              title={t(option.label)}
              description={t(option.description)}
              onClick={() => onOptionSelect(option.id)}
              endContent={
                option.id === selectedOption ? <CheckIcon className="size-4" /> : undefined
              }
            />
          ))}
        </DropdownMenuContent>
      </DropdownMenu>
    </ButtonGroup>
  );
}
