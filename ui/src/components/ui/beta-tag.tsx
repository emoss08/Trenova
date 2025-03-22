import { cn } from "@/lib/utils";
import { faSparkles } from "@fortawesome/pro-regular-svg-icons";
import { Icon } from "./icons";

type BetaTagProps = {
  label?: string;
  className?: string;
};

export function BetaTag({ label = "BETA", className }: BetaTagProps) {
  return (
    <span
      tabIndex={0}
      className={cn(
        "inline-flex text-center items-center rounded-full bg-primary/10 gap-1 px-2 py-0.5 text-2xs font-medium text-primary ring-1 ring-inset ring-primary/20 select-none",
        className,
      )}
    >
      <Icon icon={faSparkles} className="size-3 text-primary/70" />
      <span className="text-center mt-0.5">{label}</span>
    </span>
  );
}
