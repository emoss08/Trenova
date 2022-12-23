// material-ui
import { styled } from '@mui/material/styles';

// ==============================|| CALENDAR - STYLED ||============================== //

const ExperimentalStyled = styled('div')(({ theme }) => ({
  width: 'calc(100% + 2px)',
  marginLeft: -1,
  marginBottom: '-50px',

  // hide license message
  '& .fc-license-message': {
    display: 'none'
  },
  '& .fc .fc-daygrid .fc-scroller-liquid-absolute': {
    overflow: 'hidden !important'
  },

  // basic style
  '& .fc': {
    '--fc-bg-event-opacity': 1,
    '--fc-border-color': theme.palette.divider,
    '--fc-daygrid-event-dot-width': '10px',
    '--fc-today-bg-color': theme.palette.primary.lighter,
    '--fc-list-event-dot-width': '10px',
    '--fc-event-border-color': theme.palette.primary.dark,
    '--fc-now-indicator-color': theme.palette.error.main,
    color: theme.palette.text.primary,
    backgroundColor: theme.palette.background.paper,
    fontFamily: theme.typography.fontFamily
  },

  // date text
  '& .fc .fc-daygrid-day-top': {
    display: 'grid',
    '& .fc-daygrid-day-number': {
      textAlign: 'center',
      marginTop: 12,
      marginBottom: 12
    }
  },

  // weekday
  '& .fc .fc-col-header-cell': {
    backgroundColor: theme.palette.grey[100]
  },

  '& .fc .fc-col-header-cell-cushion': {
    color: theme.palette.grey[900],
    padding: 16
  },

  // events
  '& .fc-direction-ltr .fc-daygrid-event.fc-event-end, .fc-direction-rtl .fc-daygrid-event.fc-event-start': {
    marginLeft: 4,
    marginBottom: 6,
    borderRadius: 4,
    backgroundColor: theme.palette.primary.main,
    border: 'none'
  },

  '& .fc-h-event .fc-event-main': {
    padding: 4,
    paddingLeft: 8
  },

  // popover when multiple events
  '& .fc .fc-more-popover': {
    border: 'none',
    borderRadius: 6
  },

  '& .fc .fc-more-popover .fc-popover-body': {
    backgroundColor: theme.palette.grey[200],
    borderBottomLeftRadius: 4,
    borderBottomRightRadius: 4
  },

  '& .fc .fc-popover-header': {
    padding: 12,
    borderTopLeftRadius: 4,
    borderTopRightRadius: 4,
    backgroundColor: theme.palette.grey[200],
    color: theme.palette.text.primary
  },

  // agenda view
  '& .fc-theme-standard .fc-list-day-cushion': {
    backgroundColor: theme.palette.grey[100]
  },

  '& .fc .fc-day': {
    cursor: 'pointer'
  },

  '& .fc .fc-timeGridDay-view .fc-timegrid-slot': {
    backgroundColor: theme.palette.background.paper
  },

  '& .fc .fc-timegrid-slot': {
    cursor: 'pointer'
  },

  '& .fc .fc-list-event:hover td': {
    cursor: 'pointer',
    backgroundColor: theme.palette.grey[100]
  },

  '& .fc-timegrid-event-harness-inset .fc-timegrid-event, .fc-timegrid-event.fc-event-mirror, .fc-timegrid-more-link': {
    padding: 8,
    margin: 2
  }
}));

export default ExperimentalStyled;
