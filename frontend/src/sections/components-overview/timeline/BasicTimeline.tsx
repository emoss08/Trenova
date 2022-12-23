// material-ui
import { Timeline, TimelineConnector, TimelineContent, TimelineDot, TimelineItem, TimelineSeparator } from '@mui/lab';

// project import
import MainCard from 'components/MainCard';

// ==============================|| TIMELINE - BASIC ||============================== //

export default function BasicTimeline() {
  const basicTimelineCodeString = `<Timeline>
  <TimelineItem>
    <TimelineSeparator>
      <TimelineDot variant="outlined" color="primary" />
      <TimelineConnector />
    </TimelineSeparator>
    <TimelineContent>Eat</TimelineContent>
  </TimelineItem>
  <TimelineItem>
    <TimelineSeparator>
      <TimelineDot variant="outlined" color="primary" />
      <TimelineConnector />
    </TimelineSeparator>
    <TimelineContent>Code</TimelineContent>
  </TimelineItem>
  <TimelineItem sx={{ minHeight: 'auto' }}>
    <TimelineSeparator>
      <TimelineDot variant="outlined" color="primary" />
    </TimelineSeparator>
    <TimelineContent>Sleep</TimelineContent>
  </TimelineItem>
</Timeline>`;

  return (
    <MainCard title="Basic" codeString={basicTimelineCodeString}>
      <Timeline>
        <TimelineItem>
          <TimelineSeparator>
            <TimelineDot variant="outlined" color="primary" />
            <TimelineConnector />
          </TimelineSeparator>
          <TimelineContent>Eat</TimelineContent>
        </TimelineItem>
        <TimelineItem>
          <TimelineSeparator>
            <TimelineDot variant="outlined" color="primary" />
            <TimelineConnector />
          </TimelineSeparator>
          <TimelineContent>Code</TimelineContent>
        </TimelineItem>
        <TimelineItem sx={{ minHeight: 'auto' }}>
          <TimelineSeparator>
            <TimelineDot variant="outlined" color="primary" />
          </TimelineSeparator>
          <TimelineContent>Sleep</TimelineContent>
        </TimelineItem>
      </Timeline>
    </MainCard>
  );
}
