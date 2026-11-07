-- What the best driver for a move pays in preferences, as the plan measured it. Every
-- candidate's cost to the plan is its own cost less this, so the console can show the
-- same number the plan weighed against the coverage price.
ALTER TABLE "dispatch_plan_prices"
    ADD COLUMN IF NOT EXISTS "preference_floor_cents" bigint NOT NULL DEFAULT 0;
