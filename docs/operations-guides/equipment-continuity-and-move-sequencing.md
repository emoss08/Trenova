# Equipment Continuity and Move Sequencing Guide

## Purpose

This guide explains how the system now controls trailer location continuity, move sequencing, equipment usage, and actual-time validation.

This is an operational guide for dispatchers, planners, and load coordinators. It is not a developer document.

## Why This Exists

The system is designed to reflect real-world operations.

That means:

- a trailer cannot appear at a pickup if it was last completed somewhere else
- a tractor or trailer cannot be actively running on two moves at the same time
- a primary worker cannot be recorded on overlapping moves
- actual stop times must create a believable timeline across loads

The goal is to keep planning, dispatch, and execution in the correct order.

## Core Concepts

### Trailer continuity

Trailer continuity means the system tracks where a trailer is currently considered to be.

That location comes from:

- the delivery location of the trailer's last completed move
- or a manual trailer locate action

If trailer continuity is enabled for your organization, a trailer can only be assigned to a move when the trailer's current known location matches that move's pickup location.

### In-progress equipment lock

The system allows planners to assign the same tractor or trailer to future moves. That is normal planning behavior.

However, once a move goes `In Transit`, that tractor and trailer are considered actively in use. No other move can also go `In Transit` with that same equipment until the first move is completed.

### Actual-time sequencing

Actual arrival and departure times are validated across shipments and moves for:

- the assigned tractor
- the assigned primary worker

This prevents impossible timelines where the same tractor or worker appears to be working two overlapping moves.

## How Trailer Continuity Works

### When the system checks continuity

Trailer continuity is checked when assigning a trailer to a move.

If the trailer's current location does not match the move's pickup location, the assignment is blocked.

Example:

1. Trailer `TEST-53` completes a move in `Madison, GA`.
2. A dispatcher tries to assign that same trailer to a new move that picks up in `Greensboro, NC`.
3. The system blocks the assignment because the trailer is still considered to be in `Madison, GA`.

The assignment error will explain which trailer is affected and where the system currently believes that trailer is located.

### What counts as the trailer's current location

The system uses the latest valid continuity event.

Today that means:

- a completed move
- a manual trailer locate

An in-progress move does not replace the trailer's last completed or manually located position for continuity.

### Brand new trailers

A brand new trailer does not need to be located before it can be dispatched.

If the trailer has no continuity history yet, the system treats it as dispatchable.

### Canceled work

Canceled work does not establish continuity.

If a move was started and later canceled, that canceled work should not be treated as the trailer's real completed location.

## Manual Trailer Locate

### When to use locate

Use `Locate Trailer` when the trailer needs to be repositioned operationally before it can be assigned to the next pickup.

Typical example:

1. The trailer's last completed move ends in `Denver Drop Point`.
2. The next planned move starts in `Los Angeles Terminal`.
3. The trailer must be relocated before it can be used on that move.

### What the system does during locate

When a trailer is located:

- the system creates a system-generated empty reposition move
- that move is created on the shipment that last established the trailer's location
- the move is marked as empty, not loaded
- the move is auto-completed by the system
- a completed assignment is created on that move so the system knows which trailer was located
- a shipment comment is added to show that the move was system-generated
- trailer continuity is advanced to the new location

The user does not need to enter stop times for this move. The system creates placeholder planned and actual timestamps internally.

### When locate is blocked

You cannot locate a trailer when:

- the trailer is already in progress on another move
- the trailer has no continuity history and does not need a manual locate
- the requested location is the same as the trailer's current known location
- the system does not have a valid previous shipment association for that trailer's continuity history

## In-Progress Locking

### Tractor and trailer lock

A tractor or trailer may be assigned to more than one future move, but only one move can actively run with that equipment at a time.

If one move is already `In Transit`, another move cannot also be put `In Transit` with the same:

- tractor
- trailer

The system will return an error such as:

- `Tractor is currently in progress on another move`
- `Trailer is currently in progress on another move`

This rule applies whether the move is started directly from a move action or indirectly through shipment updates.

### Why this matters

This keeps dispatching sequential.

The system should let you plan the chain, but execution must stay realistic:

1. move A is assigned
2. move A goes in transit
3. move A completes
4. move B can then begin

## Actual Arrival and Departure Validation

### What the system validates

When actual stop times are updated, the system checks whether those times conflict with other recorded work for the same:

- tractor
- primary worker

This validation applies across:

- different shipments
- multiple moves in the same shipment

### Example

Move 1 on tractor `T100` and worker `W100`:

- Dallas pickup
  - actual arrival: March 12, 2026 8:00 AM
  - actual departure: March 13, 2026 8:30 AM
- Miami delivery
  - actual arrival: March 14, 2026 8:00 AM
  - actual departure: March 15, 2026 8:30 AM

Move 2 on that same tractor or worker:

- Miami pickup
  - actual arrival: March 13, 2026 8:00 AM
  - actual departure: March 15, 2026 8:30 AM

Move 2 should be rejected because the tractor and worker are already occupied on Move 1 during that time window.

### What users will see

The validation error is returned on the stop field being edited, for example `actual arrival`.

If both the tractor and the primary worker conflict in the same time window, the system may combine that into one message so the issue is easier to understand.

Typical message:

`Arrival time cannot be at March 18, 2026 8:31:00 AM EDT because this tractor and primary worker are already in use from March 18, 2026 8:00:00 AM EDT to March 18, 2026 8:32:00 AM EDT`

## Operational Rules Summary

### You can do this

- assign the same trailer to future planned moves
- assign the same tractor to future planned moves
- assign a brand new trailer without locating it first
- locate a trailer before assigning it to a different pickup
- update actual times when they do not overlap with existing work

### You cannot do this

- assign a trailer to a pickup that does not match its current continuity location when trailer continuity is enabled
- put two moves in progress with the same tractor
- put two moves in progress with the same trailer
- locate a trailer that is already in progress on another move
- save actual stop times that overlap with another move for the same tractor
- save actual stop times that overlap with another move for the same primary worker

## Recommended Dispatcher Workflow

1. Plan future moves as needed.
2. Before assigning a trailer, confirm whether the trailer is already at the next pickup.
3. If it is not, use `Locate Trailer`.
4. Only start one active move at a time for a tractor or trailer.
5. Enter actual stop times in real execution order.
6. If you receive a sequencing or overlap error, check the equipment and worker's prior move history before saving.

## Important Notes

### Trailer continuity is operational, not DOT compliance

Trailer continuity is separate from DOT compliance enforcement.

DOT compliance enforcement remains focused on:

- medical compliance
- driver qualification compliance
- hazardous material compliance
- drug and alcohol compliance

### Trailer locate is available now

Manual locate is currently available for trailers.

The same operational ideas also apply to tractors, but trailer locate is the user-facing locate workflow available today.

## Troubleshooting

### "The trailer should be here, but the system says it is somewhere else"

Check:

- the trailer's last completed move
- whether a manual locate was already performed
- whether the move you are looking at is only in progress and not yet completed

Remember: an in-progress move does not replace the last completed location for continuity.

### "Why can I assign the same trailer to multiple future loads?"

Because assignment is planning.

The system allows future planning, but it still enforces:

- trailer continuity at assignment when a known current location exists
- in-progress locks during execution

### "Why did the system create another move on the shipment?"

That is usually a system-generated empty reposition move created from a trailer locate action.

It is there to preserve the real movement history of the trailer and to update continuity correctly.

## Bottom Line

The system now enforces a more realistic order of operations:

- equipment must be where the next move says it is
- equipment cannot be actively used on multiple moves at once
- actual execution times must form a believable sequence

If your team follows the real physical flow of the trailer, tractor, and worker, the system should match that flow cleanly.
