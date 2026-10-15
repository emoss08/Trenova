DROP POLICY IF EXISTS read_scope ON public.workers;

--bun:split
DROP POLICY IF EXISTS read_scope ON public.invoices;

--bun:split
DROP POLICY IF EXISTS read_scope ON public.orders;

--bun:split
DROP POLICY IF EXISTS read_scope ON public.shipments;

--bun:split
DROP FUNCTION IF EXISTS trenova_rls.read_scope_ids(text);

--bun:split
DROP FUNCTION IF EXISTS trenova_rls.read_scope(text);
