import { MetaTags } from "@/components/meta-tags";
import { version } from "react";

export function Dashboard() {
  return (
    <>
      <MetaTags title="Dashboard" description="Dashboard" />
      <span>{`${version}`}</span>
    </>
  );
}

