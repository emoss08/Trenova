// material-ui
import {
  Timeline,
  TimelineConnector,
  TimelineContent,
  TimelineDot,
  TimelineItem,
  TimelineOppositeContent,
  TimelineSeparator
} from '@mui/lab';

// project import
import MainCard from 'components/MainCard';

// ==============================|| TIMELINE - OPPOSITE CONTENT ||============================== //

export default function OppositeContentTimeline() {
  const oppositeTimelineCodeString = `<Timeline position="alternate">
  <TimelineItem>
    <TimelineOppositeContent color="text.secondary">09:30 am</TimelineOppositeContent>
    <TimelineSeparator>
      <TimelineDot variant="outlined" color="primary" />
      <TimelineConnector />
    </TimelineSeparator>
    <TimelineContent>Eat</TimelineContent>
  </TimelineItem>
  <TimelineItem>
    <TimelineOppositeContent color="text.secondary">10:00 am</TimelineOppositeContent>
    <TimelineSeparator>
      <TimelineDot variant="outlined" color="primary" />
      <TimelineConnector />
    </TimelineSeparator>
    <TimelineContent>Code</TimelineContent>
  </TimelineItem>
  <TimelineItem>
    <TimelineOppositeContent color="text.secondary">12:00 am</TimelineOppositeContent>
    <TimelineSeparator>
      <TimelineDot variant="outlined" color="primary" />
      <TimelineConnector />
    </TimelineSeparator>
    <TimelineContent>Sleep</TimelineContent>
  </TimelineItem>
  <TimelineItem sx={{ minHeight: 'auto' }}>
    <TimelineOppositeContent color="text.secondary">9:00 am</TimelineOppositeContent>
    <TimelineSeparator>
      <TimelineDot variant="outlined" color="primary" />
      <TimelineConnector />
    </TimelineSeparator>
    <TimelineContent>Repeat</TimelineContent>
  </TimelineItem>
</Timeline>`;

  return (
    <MainCard title="Opposite Content" codeString={oppositeTimelineCodeString}>
      <Timeline position="alternate">
        <TimelineItem>
          <TimelineOppositeContent color="text.secondary">09:30 am</TimelineOppositeContent>
          <TimelineSeparator>
            <TimelineDot variant="outlined" color="primary" />
            <TimelineConnector />
          </TimelineSeparator>
          <TimelineContent>Eat</TimelineContent>
        </TimelineItem>
        <TimelineItem>
          <TimelineOppositeContent color="text.secondary">10:00 am</TimelineOppositeContent>
          <TimelineSeparator>
            <TimelineDot variant="outlined" color="primary" />
            <TimelineConnector />
          </TimelineSeparator>
          <TimelineContent>Code</TimelineContent>
        </TimelineItem>
        <TimelineItem>
          <TimelineOppositeContent color="text.secondary">12:00 am</TimelineOppositeContent>
          <TimelineSeparator>
            <TimelineDot variant="outlined" color="primary" />
            <TimelineConnector />
          </TimelineSeparator>
          <TimelineContent>Sleep</TimelineContent>
        </TimelineItem>
        <TimelineItem sx={{ minHeight: 'auto' }}>
          <TimelineOppositeContent color="text.secondary">9:00 am</TimelineOppositeContent>
          <TimelineSeparator>
            <TimelineDot variant="outlined" color="primary" />
            <TimelineConnector />
          </TimelineSeparator>
          <TimelineContent>Repeat</TimelineContent>
        </TimelineItem>
      </Timeline>
    </MainCard>
  );
}
