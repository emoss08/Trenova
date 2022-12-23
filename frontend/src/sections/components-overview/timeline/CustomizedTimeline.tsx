// material-ui
import Typography from '@mui/material/Typography';
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

// assets
import { CoffeeOutlined, DesktopOutlined, GiftOutlined, RetweetOutlined } from '@ant-design/icons';

// ==============================|| TIMELINE - CUSTOMIZED ||============================== //

export default function CustomizedTimeline() {
  const customTimelineCodeString = `<Timeline
  position="alternate"
  sx={{
    '& .MuiTimelineItem-root': { minHeight: 90 },
    '& .MuiTimelineOppositeContent-root': { mt: 0.5 },
    '& .MuiTimelineDot-root': {
      borderRadius: 1.25,
      boxShadow: 'none',
      margin: 0,
      ml: 1.25,
      mr: 1.25,
      p: 1,
      '& .MuiSvgIcon-root': { fontSize: '1.2rem' }
    },
    '& .MuiTimelineContent-root': { borderRadius: 1, bgcolor: 'secondary.lighter', height: '100%' },
    '& .MuiTimelineConnector-root': { border: '1px dashed', borderColor: 'secondary.light', bgcolor: 'transparent' }
  }}
>
  <TimelineItem>
    <TimelineOppositeContent align="right" variant="body2" color="text.secondary">
      9:30 am
    </TimelineOppositeContent>
    <TimelineSeparator>
      <TimelineDot sx={{ color: 'primary.main', bgcolor: 'primary.lighter' }}>
        <CoffeeOutlined style={{ fontSize: '1.3rem' }} />
      </TimelineDot>
      <TimelineConnector />
    </TimelineSeparator>
    <TimelineContent>
      <Typography variant="h6" component="span">
        Eat
      </Typography>
      <Typography color="textSecondary">Because you need strength</Typography>
    </TimelineContent>
  </TimelineItem>
  <TimelineItem>
    <TimelineOppositeContent variant="body2" color="text.secondary">
      10:00 am
    </TimelineOppositeContent>
    <TimelineSeparator>
      <TimelineDot sx={{ color: 'success.main', bgcolor: 'success.lighter' }}>
        <DesktopOutlined style={{ fontSize: '1.3rem' }} />
      </TimelineDot>
      <TimelineConnector />
    </TimelineSeparator>
    <TimelineContent>
      <Typography variant="h6" component="span">
        Code
      </Typography>
      <Typography color="textSecondary">Because it&apos;s awesome!</Typography>
    </TimelineContent>
  </TimelineItem>
  <TimelineItem>
    <TimelineOppositeContent align="right" variant="body2" color="text.secondary">
      11:30 am
    </TimelineOppositeContent>
    <TimelineSeparator>
      <TimelineDot sx={{ color: 'warning.main', bgcolor: 'warning.lighter' }}>
        <GiftOutlined style={{ fontSize: '1.3rem' }} />
      </TimelineDot>
      <TimelineConnector />
    </TimelineSeparator>
    <TimelineContent>
      <Typography variant="h6" component="span">
        Gift
      </Typography>
      <Typography color="textSecondary">Because you need.</Typography>
    </TimelineContent>
  </TimelineItem>
  <TimelineItem>
    <TimelineOppositeContent align="right" variant="body2" color="text.secondary">
      12:30 am
    </TimelineOppositeContent>
    <TimelineSeparator>
      <TimelineDot sx={{ color: 'error.main', bgcolor: 'error.lighter' }}>
        <RetweetOutlined style={{ fontSize: '1.3rem' }} />
      </TimelineDot>
      <TimelineConnector />
    </TimelineSeparator>
    <TimelineContent>
      <Typography variant="h6" component="span">
        Repeat
      </Typography>
      <Typography color="textSecondary">This is the life you love!</Typography>
    </TimelineContent>
  </TimelineItem>
</Timeline>`;

  return (
    <MainCard title="Customized" codeString={customTimelineCodeString}>
      <Timeline
        position="alternate"
        sx={{
          '& .MuiTimelineItem-root': { minHeight: 90 },
          '& .MuiTimelineOppositeContent-root': { mt: 0.5 },
          '& .MuiTimelineDot-root': {
            borderRadius: 1.25,
            boxShadow: 'none',
            margin: 0,
            ml: 1.25,
            mr: 1.25,
            p: 1,
            '& .MuiSvgIcon-root': { fontSize: '1.2rem' }
          },
          '& .MuiTimelineContent-root': { borderRadius: 1, bgcolor: 'secondary.lighter', height: '100%' },
          '& .MuiTimelineConnector-root': { border: '1px dashed', borderColor: 'secondary.light', bgcolor: 'transparent' }
        }}
      >
        <TimelineItem>
          <TimelineOppositeContent align="right" variant="body2" color="text.secondary">
            9:30 am
          </TimelineOppositeContent>
          <TimelineSeparator>
            <TimelineDot sx={{ color: 'primary.main', bgcolor: 'primary.lighter' }}>
              <CoffeeOutlined style={{ fontSize: '1.3rem' }} />
            </TimelineDot>
            <TimelineConnector />
          </TimelineSeparator>
          <TimelineContent>
            <Typography variant="h6" component="span">
              Eat
            </Typography>
            <Typography color="textSecondary">Because you need strength</Typography>
          </TimelineContent>
        </TimelineItem>
        <TimelineItem>
          <TimelineOppositeContent variant="body2" color="text.secondary">
            10:00 am
          </TimelineOppositeContent>
          <TimelineSeparator>
            <TimelineDot sx={{ color: 'success.main', bgcolor: 'success.lighter' }}>
              <DesktopOutlined style={{ fontSize: '1.3rem' }} />
            </TimelineDot>
            <TimelineConnector />
          </TimelineSeparator>
          <TimelineContent>
            <Typography variant="h6" component="span">
              Code
            </Typography>
            <Typography color="textSecondary">Because it&apos;s awesome!</Typography>
          </TimelineContent>
        </TimelineItem>
        <TimelineItem>
          <TimelineOppositeContent align="right" variant="body2" color="text.secondary">
            11:30 am
          </TimelineOppositeContent>
          <TimelineSeparator>
            <TimelineDot sx={{ color: 'warning.main', bgcolor: 'warning.lighter' }}>
              <GiftOutlined style={{ fontSize: '1.3rem' }} />
            </TimelineDot>
            <TimelineConnector />
          </TimelineSeparator>
          <TimelineContent>
            <Typography variant="h6" component="span">
              Gift
            </Typography>
            <Typography color="textSecondary">Because you need.</Typography>
          </TimelineContent>
        </TimelineItem>
        <TimelineItem>
          <TimelineOppositeContent align="right" variant="body2" color="text.secondary">
            12:30 am
          </TimelineOppositeContent>
          <TimelineSeparator>
            <TimelineDot sx={{ color: 'error.main', bgcolor: 'error.lighter' }}>
              <RetweetOutlined style={{ fontSize: '1.3rem' }} />
            </TimelineDot>
            <TimelineConnector />
          </TimelineSeparator>
          <TimelineContent>
            <Typography variant="h6" component="span">
              Repeat
            </Typography>
            <Typography color="textSecondary">This is the life you love!</Typography>
          </TimelineContent>
        </TimelineItem>
      </Timeline>
    </MainCard>
  );
}
