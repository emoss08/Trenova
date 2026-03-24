import { cn } from "@/lib/utils";
import { Input as InputPrimitive } from "@base-ui/react/input";

export type InputProps = React.ComponentProps<"input"> & {
  sideText?: string;
  rightElement?: React.ReactNode;
  leftElement?: React.ReactNode;
  inputContainerClassName?: string;
};

function Input({
  className,
  sideText,
  rightElement,
  leftElement,
  inputContainerClassName,
  ...props
}: InputProps) {
  return (
    <div className={cn("relative flex", inputContainerClassName)}>
      {leftElement && (
        <div
          className="pointer-events-none absolute inset-y-0 left-0 z-10 flex items-center pl-2"
          aria-hidden="true"
        >
          {leftElement}
        </div>
      )}
      <InputPrimitive
        data-slot="input"
        className={cn(
          "h-8 w-full min-w-0 rounded-md border border-input bg-muted px-2.5 py-1 text-base outline-none md:text-sm",
          "focus-visible:border-brand focus-visible:ring-4 focus-visible:ring-brand/30 focus-visible:outline-hidden",
          "aria-invalid:border-destructive aria-invalid:bg-destructive/20 aria-invalid:ring-destructive aria-invalid:focus:outline-hidden",
          "aria-invalid:placeholder:text-destructive aria-invalid:focus-visible:border-destructive aria-invalid:focus-visible:ring-4 aria-invalid:focus-visible:ring-destructive/20",
          "disabled:pointer-events-none disabled:cursor-not-allowed disabled:bg-input/50 disabled:opacity-50 dark:disabled:bg-input/80",
          "file:inline-flex file:h-6 file:border-0 file:bg-transparent file:text-sm file:font-medium file:text-foreground placeholder:text-muted-foreground",
          "transition-[border-color,box-shadow] duration-200 ease-in-out",
          (rightElement || sideText) && "pr-12",
          leftElement && "pl-7",
          className,
        )}
        {...props}
      />
      {sideText && (
        <div
          className="pointer-events-none absolute inset-y-0 right-0 z-10 flex items-center pr-2 text-xs text-muted-foreground"
          aria-hidden="true"
        >
          {sideText}
        </div>
      )}
      {rightElement && (
        <div className="absolute inset-y-0 right-0 z-10 flex items-center pr-1">{rightElement}</div>
      )}
    </div>
  );
}

export { Input };
