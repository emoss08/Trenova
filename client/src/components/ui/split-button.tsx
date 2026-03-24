import { Button } from "@/components/ui/button";
import { ButtonGroup } from "@/components/ui/button-group";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { CheckIcon, ChevronDownIcon } from "lucide-react";

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
};

export function SplitButton<T extends string = string>({
  options,
  selectedOption,
  onOptionSelect,
  isLoading = false,
  loadingText,
  disabled = false,
  className,
  formId,
}: SplitButtonProps<T>) {
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
        className="border-r border-r-brand-foreground/10"
      >
        {selected?.label}
      </Button>
      <DropdownMenu>
        <DropdownMenuTrigger
          disabled={disabled || isLoading}
          render={
            <Button type="button" disabled={disabled || isLoading}>
              <ChevronDownIcon className="size-4" />
            </Button>
          }
        />
        <DropdownMenuContent align="end" sideOffset={4}>
          {otherOptions.map((option) => (
            <DropdownMenuItem
              key={option.id}
              title={option.label}
              description={option.description}
              onClick={() => onOptionSelect(option.id)}
              endContent={
                option.id === selectedOption ? (
                  <CheckIcon className="size-4" />
                ) : undefined
              }
            />
          ))}
        </DropdownMenuContent>
      </DropdownMenu>
    </ButtonGroup>
  );
}
