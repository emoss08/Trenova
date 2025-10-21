import { QueryLazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { queries } from "@/lib/queries";
import { lazy, memo } from "react";

const DispatchControlForm = lazy(
  () => import("./_components/dispatch-control-form"),
);

export function DispatchControl() {
  return (
    <DispatchControlInner>
      <MetaTags title="Dispatch Control" description="Dispatch Control" />
      <Header />
      <QueryLazyComponent
        queryKey={queries.organization.getDispatchControl._def}
      >
        <DispatchControlForm />
      </QueryLazyComponent>
    </DispatchControlInner>
  );
}

function DispatchControlInner({ children }: { children: React.ReactNode }) {
  return <div className="flex flex-col gap-y-3">{children}</div>;
}

const Header = memo(() => {
  return (
    <div className="flex justify-between items-center">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Dispatch Control</h1>
        <p className="text-muted-foreground">
          Configure and manage your dispatch control settings
        </p>
      </div>
    </div>
  );
});
Header.displayName = "Header";
