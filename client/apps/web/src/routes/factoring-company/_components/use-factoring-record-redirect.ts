import { panelSearchParamsParser } from "@/hooks/data-table/use-data-table-state";
import { factoring as factoringQueries } from "@/lib/queries/factoring";
import { useQuery } from "@tanstack/react-query";
import { useQueryStates } from "nuqs";
import { useEffect } from "react";
import { factoringCompanyRecordSearchParamsParser } from "../use-factoring-company-state";

/**
 * A link to an agreement or a notice names only that record; both are kept on
 * their factor's panel. Looks up which factor owns the record and opens that
 * factor, leaving the record's own parameters in place so its section can
 * point at it.
 */
export function useFactoringRecordRedirect() {
  const [state, setState] = useQueryStates({
    ...factoringCompanyRecordSearchParamsParser,
    ...panelSearchParamsParser,
  });
  const waiting = !state.panelEntityId;
  const agreementOwner = useQuery({
    ...factoringQueries.agreementOwner(state.agreement ?? ""),
    enabled: waiting && !!state.agreement,
  });
  const noaOwner = useQuery({
    ...factoringQueries.noaOwner(state.noa ?? ""),
    enabled: waiting && !state.agreement && !!state.noa,
  });
  const owner = state.agreement ? agreementOwner.data : noaOwner.data;

  useEffect(() => {
    if (waiting && owner) {
      void setState({ panelType: "edit", panelEntityId: owner });
    }
  }, [waiting, owner, setState]);
}

/** Clears an agreement or notice a record link opened, once its factor's panel closes. */
export function useClearFactoringRecordParams() {
  const [, setState] = useQueryStates(factoringCompanyRecordSearchParamsParser);
  return () => void setState({ section: null, agreement: null, noa: null });
}
