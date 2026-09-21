import { cn } from "@trenova/shared/lib/utils";
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
          "ui-field ui-focus-ring h-7 w-full min-w-0 px-2 py-0.5 text-base outline-none md:text-sm",
          "aria-invalid:border-danger aria-invalid:bg-danger-subtle aria-invalid:[--ring:var(--ring-danger)]",
          "aria-invalid:placeholder:text-danger-foreground",
          "disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-60",
          "file:inline-flex file:h-6 file:border-0 file:bg-transparent file:text-sm file:font-medium file:text-foreground placeholder:text-muted-foreground",
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
