---
path: /dispatch/outlook
aliases: [freight outlook, freight forecast, where to send trucks, position drivers, reposition, deadhead toward freight, short of trucks, trucks to spare, booking pace, customer behind pace, forecast track record]
related:
  - /dispatch/console
  - /admin/dispatch-controls
  - /hr/workers
---

## What it's for
The freight outlook shows where your trucks will be short or idle over the next two days, before the freight is booked, and the moves that fix it. It is learned from your own shipment history, not market data, and it appears once the forecast has proven itself on that history (**Ready**). Until then the page says why, and selecting **See the forecast in Dispatch controls** shows how it was tested.

The page fills the screen like the dispatch console. Across the top, the figures show **Loads with no truck nearby**, **Drivers with nothing nearby**, **Across the network** (whether you have enough drivers overall, which usually you do, so the trouble is where they are), **Worth acting on** and **Range held** (how many of the last fourteen days landed inside the forecast's range). Under them, a paragraph reads the outlook in words: which areas will run short, how many drivers will sit with little to haul and what moving them is worth, and which customer is behind its usual booking pace. **12 h**, **24 h** and **48 h** at the top right set how far ahead the page reads.

The middle of the page is the board, and the panel on the right is the brief: the track record, the next moves and customer pace in one scroll. Selecting an area turns the panel into that area; **Brief** or Escape brings it back. Select **Hide the brief** (the double arrow in the panel's header) to fold it to a narrow rail and give the board the room; the rail still shows how many moves are waiting, and **Show the brief** brings it back. The page remembers which way you left it.

## Tasks

### Read the board
Keywords: freight forecast chart, when trucks are short, busiest window, area heat map
1. Open [Freight outlook](/dispatch/outlook).
2. The chart at the top of **Loads against drivers** shows every four-hour window across your whole network: the loads still needing a driver with their usual range (the shaded band), the drivers free for that window, and underneath, the loads that will have no truck nearby (amber) and the idle drivers (blue) once each area is counted on its own.
3. Under the chart, every area has a row of the same windows, so a window is a column you read top to bottom: amber with a minus is trucks short, blue with a plus is trucks to spare. Areas are grouped into **Short of trucks**, **Could run short**, **Trucks to spare** and **Balanced** (folded; select it to open). Each row says when the area is worst and why, and on the right what one more driver ending there is worth. An area with drivers to spare, or exactly enough, reads **Not needed**, and an area the forecast has no revenue for yet reads **No price**; hover either for why.
4. Hover a cell or a window for its numbers. Select a window, in the chart or in the row of times, to read the whole page at that hour: the paragraph, the figures, the brief and the areas all change to it. Select it again, or **Show all**, to return.

### Send a driver toward freight
Keywords: reposition driver, deadhead, send empty truck, position driver, move truck before load
1. Open [Freight outlook](/dispatch/outlook) and find a **Send** move under **Next moves** in the brief. Each row says who goes where, where they are free and when, and the empty miles; the figure on the right is what it is worth after them.
2. Rest the pointer on a move to see why: the window it is for, how the figure adds up (what the area gains, what the driver's own area loses and the empty miles) and what the button does. The board draws the move at the same time, from the window the driver frees up in to the window they fill, and dims every other area.
3. Select **Send**. The driver is told in Dash where to head and by when, and the forecast and the planner count them at that area from their arrival.
4. The driver stays in **Next moves** as **Sent** until they arrive. Select **On the way · Undo** to call it off; the driver is told.
5. A driver who is given a load after being sent is no longer counted at that area, so the next load you assign always wins.

### Book, keep or call instead
Keywords: load ending near, swap a load, keep a driver, customer behind pace, ask for tenders
1. **Book** finds an open load that picks up near a driver and delivers near an area that will be short, so the driver gets there loaded instead of empty. Select **See loads** to open the load with the driver picked in the console, and assign it from there.
2. **Keep** names a driver whose next load would carry them out of an area that is about to run short, and another driver who can take that load in time. Select **Review** to open the load with the other driver picked in the console, and reassign it from there.
3. **Call** names a customer that has tendered less than it usually has by this hour for an area. Select **Open** to reach them before the freight goes to another carrier.
4. Select **Not now** (the cross at the end of a move) to hide it for now; the button under the list brings hidden moves back. When no free driver can reach a short area in time, the brief says so under the moves.

### Open an area
Keywords: area outlook, who can get there, one more driver worth
1. Select an area on the board, or an underlined area name anywhere on the page.
2. The panel shows the area: its worst window and what one more driver is worth there, the moves into or out of it, the window by window chart, who books there and whether they are on pace, drivers already on the way, who can get there in time (on a map that outlines the area and draws a line in from each driver, with what each would net, including those who arrive too late or are not worth the miles), and the last eight weeks of that window against its usual range.
3. Scroll over the map to zoom it and drag to move it. Opening another area or window frames that area and its drivers again. The map needs Google Maps connected under Integrations; without it, the panel lists the drivers on their own.
4. Select the button named for the area (Loads into, then the area) to open the shipments list filtered to the open loads delivering to any of your locations there. When none of your locations are in the area yet, the panel says so instead.
5. Select **Brief**, **Close** or press Escape to go back.

### Check how far to trust it
Keywords: forecast accuracy, track record, backtest, range held
1. Open [Freight outlook](/dispatch/outlook) and look at **Track record** at the top of the brief, or select **Range held** in the figures.
2. Each of the last fourteen days shows the range the forecast gave for the whole network, worked out only from the weeks before that day, and the loads that came. A dot outside its range is a miss. Under the chart: how many days were busier than the range, and how far a typical day lands from the middle.
