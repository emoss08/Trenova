// material-ui
import { Grid } from '@mui/material';

// project import
import ComponentHeader from 'components/cards/ComponentHeader';
import ComponentWrapper from 'sections/components-overview/ComponentWrapper';
import ComponentSkeleton from 'sections/components-overview/ComponentSkeleton';
import BasicTimeline from 'sections/components-overview/timeline/BasicTimeline';
import LeftPositionedTimeline from 'sections/components-overview/timeline/LeftPositionedTimeline';
import AlternateTimeline from 'sections/components-overview/timeline/AlternateTimeline';
import ColorsTimeline from 'sections/components-overview/timeline/ColorsTimeline';
import OppositeContentTimeline from 'sections/components-overview/timeline/OppositeContentTimeline';
import CustomizedTimeline from 'sections/components-overview/timeline/CustomizedTimeline';

// ==============================|| COMPONENTS - TIMELINE ||============================== //

const ComponentTimeline = () => (
  <ComponentSkeleton>
    <ComponentHeader
      title="Timeline"
      caption="The timeline displays a list of events in chronological order."
      directory="src/pages/components-overview/timeline"
      link="https://mui.com/material-ui/react-timeline/"
    />
    <ComponentWrapper>
      <Grid container spacing={3}>
        <Grid item xs={12} sm={6}>
          <BasicTimeline />
        </Grid>
        <Grid item xs={12} sm={6}>
          <LeftPositionedTimeline />
        </Grid>
        <Grid item xs={12} sm={6}>
          <AlternateTimeline />
        </Grid>
        <Grid item xs={12} sm={6}>
          <ColorsTimeline />
        </Grid>
        <Grid item xs={12} lg={6}>
          <OppositeContentTimeline />
        </Grid>
        <Grid item xs={12} lg={6}>
          <CustomizedTimeline />
        </Grid>
      </Grid>
    </ComponentWrapper>
  </ComponentSkeleton>
);

export default ComponentTimeline;
