import { cn } from "@/lib/utils";
import { Skeleton } from "./ui/skeleton";
import { Spinner } from "./ui/spinner";
import { TextShimmer } from "./ui/text-shimmer";

export default function LoadingSkeleton() {
  return (
    <div className="flex min-h-screen flex-row items-center justify-center text-center">
      <div className="flex w-[700px] flex-col rounded-md border border-border bg-card sm:flex-row sm:items-center sm:justify-center">
        <div className="space-y-4 p-8">
          <div className="flex items-center justify-center">
            <Spinner className="size-10" />
          </div>
          <p className="mb-2 text-xl font-semibold">
            Hang tight! <u className="font-bold underline decoration-blue-600">Trenova</u> is
            gearing up for you.
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            We&apos;re working at lightning speed to get things ready. If this takes longer than a
            coffee break (10 seconds), please check your internet connection. <br />
            <u className="text-foreground decoration-blue-600">Still stuck?</u> Your friendly system
            administrator is just a call away for a swift rescue!
          </p>
        </div>
      </div>
    </div>
  );
}

export function LoadingSkeletonState({
  description,
  className,
}: {
  description: string;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "flex h-75 w-full flex-col items-center justify-center rounded-md border border-border",
        className,
      )}
    >
      <div className="relative size-full">
        <Skeleton className="size-full" />
        <span className="absolute inset-0 flex items-center justify-center text-sm text-muted-foreground">
          <TextShimmer duration={1.5}>{description}</TextShimmer>
        </span>
      </div>
    </div>
  );
}
