// third-party
import { EventInput } from '@fullcalendar/common';

// ==============================|| CALENDAR TYPES  ||============================== //

export type DateRange = { start: number | Date; end: number | Date };
export type CalendarView = 'dayGridMonth' | 'timeGridWeek' | 'timeGridDay' | 'listWeek';

export type CalendarProps = {
  calendarView: CalendarView;
  error: boolean;
  events: EventInput[];
  isLoader: boolean;
  isModalOpen: boolean;
  selectedEventId: null | string;
  selectedRange: null | { start: Date; end: Date };
};
