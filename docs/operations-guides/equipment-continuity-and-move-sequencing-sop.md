# Equipment Continuity and Move Sequencing SOP

## Purpose

This standard operating procedure explains how dispatch and operations staff should work with:

- trailer continuity
- trailer locate
- in-progress equipment controls
- actual arrival and departure time validation

Use this guide during training, onboarding, and daily operations.

## Who Should Use This SOP

This SOP is for:

- dispatchers
- load planners
- operations coordinators
- supervisors reviewing shipment execution

## Operational Goal

The system is built to keep equipment flow realistic and sequential.

That means:

- the trailer must be at the pickup before it can be assigned
- the same tractor or trailer cannot be active on two moves at once
- the same primary worker cannot be recorded on overlapping work
- actual stop times must reflect a believable order of events

## Standard Workflow

### Step 1: Plan the next move

When planning a move, confirm:

- pickup location
- delivery location
- assigned tractor
- assigned trailer
- assigned primary worker

Planning multiple future moves with the same tractor or trailer is allowed.

### Step 2: Check trailer location before assignment

Before assigning a trailer, verify whether the trailer is already at the move's pickup location.

The trailer's current location is based on:

- the trailer's last completed move
- or a manual trailer locate

If the trailer is already at the correct pickup, continue with assignment.

If the trailer is not at the correct pickup, do not force the assignment. Use the trailer locate process first.

### Step 3: Assign the move

Assign the move only when:

- the trailer is at the correct pickup location
- or the trailer is brand new and has no prior continuity history

If the trailer is not at the correct pickup, the system will block the assignment.

Example error:

`Trailer TEST-53 is currently located at Madison Yard which doesn't match this move's current pickup location. Locate the trailer before assigning or assign a different trailer`

### Step 4: Start the move

When a move is ready to run, move it into active execution.

Before doing that, confirm:

- the tractor is not already active on another move
- the trailer is not already active on another move

If that equipment is already in use, the system will block the move from becoming active.

Example errors:

- `Tractor is currently in progress on another move`
- `Trailer is currently in progress on another move`

### Step 5: Enter actual stop times in order

When entering actual arrival and departure times:

- enter them in the real order the work happened
- make sure they do not overlap with another move for the same tractor
- make sure they do not overlap with another move for the same primary worker

If the entered time conflicts with another move, the system will reject the save.

## Trailer Locate SOP

### When to use it

Use `Locate Trailer` when the trailer's current system location does not match the next move's pickup location.

### When not to use it

Do not use `Locate Trailer` when:

- the trailer is already at the correct location
- the trailer is currently in progress on another move
- the trailer is brand new and has never established continuity

### What the system does

When `Locate Trailer` is used, the system:

1. creates a system-generated empty reposition move
2. creates a completed assignment on that move for the located trailer
3. marks the move as completed
4. adds a shipment comment explaining the system-generated move
5. updates trailer continuity to the new location

This move is operationally a reposition move. It is not a loaded move.

### Dispatcher procedure

1. Identify the trailer continuity mismatch.
2. Open the trailer locate workflow.
3. Enter the new location.
4. Submit the locate action.
5. Confirm the locate completed successfully.
6. Return to the shipment and assign the trailer again.

## Brand New Trailer SOP

If a trailer has no continuity history yet, treat it as brand new.

For a brand new trailer:

- you do not need to locate it before dispatch
- you can assign it directly

Once that trailer completes a move or is manually located, the system will begin enforcing continuity from that point forward.

## Canceled Move SOP

If a move is canceled, that canceled work should not be treated as the trailer's real completed location.

Operationally, that means:

- canceled work should not be used to justify the next pickup location
- canceled work should not be treated as completed movement

If a user believes the system is using canceled work as the trailer's current location, escalate that case for review.

## In-Progress Equipment Rules

### Rule

A tractor or trailer may be planned on multiple future moves, but only one move may actively use that equipment at a time.

### What counts as a violation

This is a violation:

1. Move A uses tractor `T100` and trailer `TR500`
2. Move A is already in progress
3. Move B is then also started with tractor `T100` or trailer `TR500`

The system should block Move B from becoming active.

### Dispatcher response

If you receive an in-progress equipment error:

1. identify the move currently using the equipment
2. confirm whether that move should be completed first
3. if the equipment assignment is wrong, correct the assignment
4. only restart the blocked move after the equipment is no longer active elsewhere

## Actual Time Entry SOP

### Rule

Actual stop times must reflect real execution.

The same tractor or primary worker cannot be recorded on overlapping time windows across different moves.

### Example

Move 1:

- pickup actual arrival: March 12, 2026 8:00 AM
- pickup actual departure: March 13, 2026 8:30 AM
- delivery actual arrival: March 14, 2026 8:00 AM
- delivery actual departure: March 15, 2026 8:30 AM

Move 2 on the same tractor or primary worker:

- pickup actual arrival: March 13, 2026 8:00 AM

That second time should be rejected because the resource is already in use during that period.

### Dispatcher procedure when entering actuals

1. Confirm you are updating the correct move.
2. Confirm the tractor and primary worker on that move.
3. Enter the actual arrival and departure in real sequence.
4. Save the shipment.
5. If the system rejects the time, review the prior and current moves for the same tractor or primary worker.

### Typical error message

`Arrival time cannot be at March 18, 2026 8:31:00 AM EDT because this tractor and primary worker are already in use from March 18, 2026 8:00:00 AM EDT to March 18, 2026 8:32:00 AM EDT`

## Common Scenarios

### Scenario 1: Trailer mismatch at assignment

Situation:

- trailer last completed in `Madison, GA`
- next move picks up in `Greensboro, NC`

Correct response:

1. do not keep retrying the assignment
2. use `Locate Trailer`
3. relocate the trailer to `Greensboro, NC`
4. assign the trailer after locate completes

### Scenario 2: Same trailer planned on multiple future loads

Situation:

- the trailer is assigned on multiple future moves
- none of those moves is in progress yet

Correct response:

- this is allowed as planning
- keep the move sequence realistic
- make sure only one move becomes active at a time

### Scenario 3: Move will not go active

Situation:

- user tries to start a move
- system says the tractor or trailer is already in progress on another move

Correct response:

1. find the active move using that equipment
2. complete or correct the active move first
3. then retry the blocked move

### Scenario 4: Actual times will not save

Situation:

- user enters actual times
- system says the tractor or primary worker is already in use

Correct response:

1. compare the entered times to the other move using that tractor or primary worker
2. fix whichever move has the incorrect actual times
3. save again once the sequence is realistic

## Do and Do Not

### Do

- plan future work in sequence
- use `Locate Trailer` when the trailer is not at the next pickup
- complete active moves before starting the next move on the same equipment
- enter actual stop times in true operational order

### Do Not

- assign a trailer to a different pickup and assume it can teleport
- start two active moves on the same tractor
- start two active moves on the same trailer
- locate a trailer that is already in progress on another move
- enter actual times that overlap another move for the same tractor or primary worker

## Supervisor Review Checklist

When reviewing dispatch behavior, confirm:

- trailers are being located before mismatched assignments are retried
- system-generated reposition moves are understood and not mistaken for loaded freight moves
- only one active move exists per tractor
- only one active move exists per trailer
- actual times are being entered in real-world order

## Escalation Guidance

Escalate the issue if:

- a trailer appears to have the wrong current location after a completed move
- a canceled move appears to be controlling continuity
- a system-generated reposition move appears loaded
- a move is blocked but operations believes the equipment is no longer active elsewhere
- actual-time conflicts appear even though no overlapping work exists

## Final Reminder

The system now expects operational sequence to match physical reality:

- where the trailer is
- what equipment is currently active
- when the work actually happened

If users follow the real movement of the equipment and enter times in the order the work occurred, the system should support that flow cleanly.
