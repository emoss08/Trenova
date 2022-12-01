import moment from "moment";
import type { EventInput } from "@fullcalendar/core";

const todayDate = moment().startOf("day");
const YM = todayDate.format("YYYY-MM");
const YESTERDAY = todayDate.clone().subtract(1, "day").format("YYYY-MM-DD");
const TODAY = todayDate.format("YYYY-MM-DD");
const TOMORROW = todayDate.clone().add(1, "day").format("YYYY-MM-DD");

const events: Array<EventInput> = [
  {
    title: "All Day Event",
    start: YM + "-01",
    description: "Toto lorem ipsum dolor sit incid idunt ut",
    className: "fc-event-danger fc-event-solid-warning",
  },
  {
    title: "Reporting",
    start: YM + "-14T13:30:00",
    description: "Lorem ipsum dolor incid idunt ut labore",
    end: YM + "-14",
    className: "fc-event-success",
  },
  {
    title: "Company Trip",
    start: YM + "-02",
    description: "Lorem ipsum dolor sit tempor incid",
    end: YM + "-03",
    className: "fc-event-primary",
  },
  {
    title: "ICT Expo 2017 - Product Release",
    start: YM + "-03",
    description: "Lorem ipsum dolor sit tempor inci",
    end: YM + "-05",
    className: "fc-event-light fc-event-solid-primary",
  },
  {
    title: "Dinner",
    start: YM + "-12",
    description: "Lorem ipsum dolor sit amet, conse ctetur",
    end: YM + "-10",
  },
  {
    id: "999",
    title: "Repeating Event",
    start: YM + "-09T16:00:00",
    description: "Lorem ipsum dolor sit ncididunt ut labore",
    className: "fc-event-danger",
  },
  {
    id: "1000",
    title: "Repeating Event",
    description: "Lorem ipsum dolor sit amet, labore",
    start: YM + "-16T16:00:00",
  },
  {
    title: "Conference",
    start: YESTERDAY,
    end: TOMORROW,
    description: "Lorem ipsum dolor eius mod tempor labore",
    className: "fc-event-primary",
  },
  {
    title: "Meeting",
    start: TODAY + "T10:30:00",
    end: TODAY + "T12:30:00",
    description: "Lorem ipsum dolor eiu idunt ut labore",
  },
  {
    title: "Lunch",
    start: TODAY + "T12:00:00",
    className: "fc-event-info",
    description: "Lorem ipsum dolor sit amet, ut labore",
  },
  {
    title: "Meeting",
    start: TODAY + "T14:30:00",
    className: "fc-event-warning",
    description: "Lorem ipsum conse ctetur adipi scing",
  },
  {
    title: "Happy Hour",
    start: TODAY + "T17:30:00",
    className: "fc-event-info",
    description: "Lorem ipsum dolor sit amet, conse ctetur",
  },
  {
    title: "Dinner",
    start: TOMORROW + "T05:00:00",
    className: "fc-event-solid-danger fc-event-light",
    description: "Lorem ipsum dolor sit ctetur adipi scing",
  },
  {
    title: "Birthday Party",
    start: TOMORROW + "T07:00:00",
    className: "fc-event-primary",
    description: "Lorem ipsum dolor sit amet, scing",
  },
  {
    title: "Click for Google",
    url: "http://google.com/",
    start: YM + "-28",
    className: "fc-event-solid-info fc-event-light",
    description: "Lorem ipsum dolor sit amet, labore",
  },
];

export default events;

export { todayDate, YM, YESTERDAY, TODAY, TOMORROW };
