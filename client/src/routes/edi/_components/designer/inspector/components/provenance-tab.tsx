import type { InspectorContext } from "../inspector-context";
import InspectorGrid from "./inspector-grid";

export default function ProvenanceTab({ context }: { context: InspectorContext }) {
  return <InspectorGrid rows={context.provenanceRows ?? []} />;
}
