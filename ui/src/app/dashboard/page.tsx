/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

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
