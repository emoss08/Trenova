import { Button, ButtonProps } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { IconDefinition } from "@fortawesome/pro-regular-svg-icons";
import { Icon } from "./icons";

interface EmptyStateProps {
  title: string;
  description: string;
  icons?: IconDefinition[];
  action?: {
    label: string;
    onClick: () => void;
    variant?: ButtonProps["variant"];
  };
  className?: string;
}

export function EmptyState({
  title,
  description,
  icons = [],
  action,
  className,
}: EmptyStateProps) {
  return (
    <div
      className={cn(
        "bg-background border-border hover:border-border/80 text-center",
        "border-2 border-dashed rounded-xl p-14 w-full max-w-[620px]",
        "group transition duration-500 hover:duration-200",
        className,
      )}
    >
      <div className="isolate flex justify-center">
        {icons.length === 3 ? (
          <>
            <div className="relative left-2.5 top-1.5 grid size-12 -rotate-6 place-items-center rounded-xl bg-background shadow-lg ring-1 ring-border transition duration-500 group-hover:-translate-x-5 group-hover:-translate-y-0.5 group-hover:-rotate-12 group-hover:duration-200">
              <Icon icon={icons[0]} className="size-6 text-muted-foreground" />
            </div>
            <div className="relative z-10 grid size-12 place-items-center rounded-xl bg-background shadow-lg ring-1 ring-border transition duration-500 group-hover:-translate-y-0.5 group-hover:duration-200">
              <Icon icon={icons[1]} className="size-6 text-muted-foreground" />
            </div>
            <div className="relative right-2.5 top-1.5 grid size-12 rotate-6 place-items-center rounded-xl bg-background shadow-lg ring-1 ring-border transition duration-500 group-hover:-translate-y-0.5 group-hover:translate-x-5 group-hover:rotate-12 group-hover:duration-200">
              <Icon icon={icons[2]} className="size-6 text-muted-foreground" />
            </div>
          </>
        ) : (
          <div className="grid size-12 place-items-center rounded-xl bg-background shadow-lg ring-1 ring-border transition duration-500 group-hover:-translate-y-0.5 group-hover:duration-200">
            <Icon icon={icons[0]} className="size-6 text-muted-foreground" />
          </div>
        )}
      </div>
      <h2 className="mt-6 font-medium text-foreground">{title}</h2>
      <p className="mt-1 whitespace-pre-line text-sm text-muted-foreground">
        {description}
      </p>
      {action && (
        <Button
          onClick={action.onClick}
          variant={action.variant}
          className={cn("mt-4", "active:shadow-none")}
        >
          {action.label}
        </Button>
      )}
    </div>
  );
}
