// third-party
import { add, set, sub } from 'date-fns';
import { EventInput } from '@fullcalendar/common';

// event colors - temp
export const EVENT_COLORS = [
  '#8c8c8c', // theme.palette.secondary.main,
  '#fffbe6', // theme.palette.warning.lighter,
  '#faad14', // theme.palette.warning.main,
  '#f6ffed', // theme.palette.success.lighter,
  '#52c41a', // theme.palette.success.main,
  '#1890ff', // theme.palette.primary.main,
  '#f5222d', // theme.palette.error.main,
  '#e6f7ff' // theme.palette.primary.lighter,
];

// calendar events
export const events: EventInput[] = [
  {
    id: '5e8882f1f0c9216397e05a9b',
    allDay: false,
    color: EVENT_COLORS[6],
    description: 'SCRUM Planning',
    start: sub(new Date(), { days: 12, hours: 5, minutes: 45 }),
    end: sub(new Date(), { days: 12, hours: 3, minutes: 30 }),
    title: 'Repeating Event'
  },
  {
    id: '5e8882fcd525e076b3c1542c',
    allDay: true,
    color: EVENT_COLORS[2],
    description: 'Sorry, John!',
    start: sub(new Date(), { days: 8, hours: 0, minutes: 45 }),
    end: sub(new Date(), { days: 8, hours: 0, minutes: 30 }),
    title: 'Conference'
  },
  {
    id: '5e8882e440f6322fa399eeb8',
    allDay: true,
    color: EVENT_COLORS[4],
    description: 'Inform about new contract',
    start: sub(new Date(), { days: 3 }),
    end: sub(new Date(), { days: 4 }),
    title: 'All Day Event'
  },
  {
    id: '5e8882fcd525e076b3c1542c',
    allDay: false,
    color: EVENT_COLORS[3],
    textColor: EVENT_COLORS[4],
    description: 'Sorry, Stebin Ben!',
    start: sub(new Date(), { days: 2, hours: 5, minutes: 0 }),
    end: sub(new Date(), { days: 2, hours: 1, minutes: 30 }),
    title: 'Opening Ceremony'
  },
  {
    id: '5e8882eb5f8ec686220ff131',
    allDay: true,
    color: EVENT_COLORS[0],
    description: 'Discuss about new partnership',
    start: sub(new Date(), { days: 4, hours: 0, minutes: 0 }),
    end: sub(new Date(), { days: 2, hours: 1, minutes: 0 }),
    title: 'Long Event'
  },
  {
    id: '5e88830672d089c53c46ece3',
    allDay: false,
    description: 'Get a new quote for the payment processor',
    start: set(new Date(), { hours: 6, minutes: 30 }),
    end: set(new Date(), { hours: 8, minutes: 30 }),
    title: 'Breakfast'
  },
  {
    id: '5e888302e62149e4b49aa609',
    allDay: false,
    color: EVENT_COLORS[1],
    textColor: EVENT_COLORS[2],
    description: 'Discuss about the new project',
    start: add(new Date(), { hours: 9, minutes: 45 }),
    end: add(new Date(), { hours: 15, minutes: 30 }),
    title: 'Meeting'
  },
  {
    id: '5e888302e62149e4b49aa709',
    allDay: false,
    color: EVENT_COLORS[6],
    description: "Let's Go",
    start: add(new Date(), { hours: 9, minutes: 0 }),
    end: add(new Date(), { hours: 11, minutes: 30 }),
    title: 'Anniversary Celebration'
  },
  {
    id: '5e888302e69651e4b49aa609',
    allDay: false,
    description: 'Discuss about the new project',
    start: add(new Date(), { days: 1, hours: 5, minutes: 25 }),
    end: add(new Date(), { days: 1, hours: 5, minutes: 55 }),
    title: 'Send Gift'
  },
  {
    id: '5e8883062k8149e4b49aa709',
    allDay: false,
    color: EVENT_COLORS[2],
    description: "Let's Go",
    start: add(new Date(), { days: 1, hours: 3, minutes: 45 }),
    end: add(new Date(), { days: 1, hours: 5, minutes: 15 }),
    title: 'Birthday Party'
  },
  {
    id: '5e8882f1f0c9216396e05a9b',
    allDay: false,
    color: EVENT_COLORS[0],
    description: 'SCRUM Planning',
    start: add(new Date(), { days: 1, hours: 3, minutes: 30 }),
    end: add(new Date(), { days: 1, hours: 4, minutes: 30 }),
    title: 'Repeating Event'
  },
  {
    id: '5e888302e62149e4b49aa610',
    allDay: false,
    color: EVENT_COLORS[6],
    description: "Let's Go",
    start: add(new Date(), { days: 1, hours: 3, minutes: 45 }),
    end: add(new Date(), { days: 1, hours: 4, minutes: 50 }),
    title: 'Dinner'
  },
  {
    id: '5e8882eb5f8ec686220ff131',
    allDay: true,
    description: 'Discuss about new partnership',
    start: add(new Date(), { days: 5, hours: 0, minutes: 0 }),
    end: add(new Date(), { days: 8, hours: 1, minutes: 0 }),
    title: 'Long Event'
  },
  {
    id: '5e888302e62349e4b49aa609',
    allDay: false,
    color: EVENT_COLORS[5],
    textColor: EVENT_COLORS[7],
    description: 'Discuss about the project launch',
    start: add(new Date(), { days: 6, hours: 0, minutes: 15 }),
    end: add(new Date(), { days: 6, hours: 0, minutes: 20 }),
    title: 'Meeting'
  },
  {
    id: '5e888302e62149e4b49ab609',
    allDay: false,
    color: EVENT_COLORS[4],
    description: 'Discuss about the tour',
    start: add(new Date(), { days: 12, hours: 3, minutes: 45 }),
    end: add(new Date(), { days: 12, hours: 4, minutes: 50 }),
    title: 'Happy Hour'
  }
];
