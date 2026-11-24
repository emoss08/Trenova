-- A team-scoped worker reader saw only their team's workers, but the rows that exist only as part
-- of a worker were read with no scope of their own: the agent's credential list, a profile read by
-- worker id or a training list could name a worker the reader may not see. Each of these rows now
-- follows its worker for every reader: the EXISTS reads workers under that table's own read_scope
-- policy, so the team rule is written once, on workers.
--
-- Left out on purpose: rows read across workers by a grant of their own (timesheets, shift
-- assignments, leave, pay, settlements, safety and HOS records, telematics) and rows consulted by
-- dispatch checks, where a hidden row would read as "nothing in the way".
DROP POLICY IF EXISTS read_scope ON public.worker_credentials;

CREATE POLICY read_scope ON public.worker_credentials AS RESTRICTIVE FOR SELECT TO trenova_tenant
    USING (
        (SELECT trenova_rls.read_scope('worker')) <> 'team'
        OR EXISTS (
            SELECT 1
            FROM public.workers w
            WHERE w.id = worker_credentials.worker_id
                AND w.organization_id = worker_credentials.organization_id
                AND w.business_unit_id = worker_credentials.business_unit_id
        )
    );

DROP POLICY IF EXISTS read_scope ON public.worker_profiles;

CREATE POLICY read_scope ON public.worker_profiles AS RESTRICTIVE FOR SELECT TO trenova_tenant
    USING (
        (SELECT trenova_rls.read_scope('worker')) <> 'team'
        OR EXISTS (
            SELECT 1
            FROM public.workers w
            WHERE w.id = worker_profiles.worker_id
                AND w.organization_id = worker_profiles.organization_id
                AND w.business_unit_id = worker_profiles.business_unit_id
        )
    );

DROP POLICY IF EXISTS read_scope ON public.worker_checklists;

CREATE POLICY read_scope ON public.worker_checklists AS RESTRICTIVE FOR SELECT TO trenova_tenant
    USING (
        (SELECT trenova_rls.read_scope('worker')) <> 'team'
        OR EXISTS (
            SELECT 1
            FROM public.workers w
            WHERE w.id = worker_checklists.worker_id
                AND w.organization_id = worker_checklists.organization_id
                AND w.business_unit_id = worker_checklists.business_unit_id
        )
    );

DROP POLICY IF EXISTS read_scope ON public.worker_training_records;

CREATE POLICY read_scope ON public.worker_training_records AS RESTRICTIVE FOR SELECT TO trenova_tenant
    USING (
        (SELECT trenova_rls.read_scope('worker')) <> 'team'
        OR EXISTS (
            SELECT 1
            FROM public.workers w
            WHERE w.id = worker_training_records.worker_id
                AND w.organization_id = worker_training_records.organization_id
                AND w.business_unit_id = worker_training_records.business_unit_id
        )
    );
