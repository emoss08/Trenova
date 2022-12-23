// material-ui
import { Grid } from '@mui/material';

// project import
import ComponentHeader from 'components/cards/ComponentHeader';
import ComponentWrapper from 'sections/components-overview/ComponentWrapper';
import ComponentSkeleton from 'sections/components-overview/ComponentSkeleton';
import SimpleSpeedDials from 'sections/components-overview/speeddial/SimpleSpeedDials';
import OpenIconSpeedDial from 'sections/components-overview/speeddial/OpenIconSpeedDial';
import SpeedDialTooltipOpen from 'sections/components-overview/speeddial/SpeedDialTooltipOpen';

// ==============================|| COMPONENTS - SPEED DIAL ||============================== //

const ComponentSpeeddial = () => (
  <ComponentSkeleton>
    <ComponentHeader
      title="Speed Dial"
      caption="When pressed, a floating action button can display three to six related actions in the form of a speed dial."
      directory="src/pages/components-overview/speeddial"
      link="https://mui.com/material-ui/react-speed-dial/"
    />
    <ComponentWrapper>
      <Grid container spacing={3}>
        <Grid item xs={12} lg={6}>
          <SimpleSpeedDials />
        </Grid>
        <Grid item xs={12} sm={6} lg={3}>
          <OpenIconSpeedDial />
        </Grid>
        <Grid item xs={12} sm={6} lg={3}>
          <SpeedDialTooltipOpen />
        </Grid>
      </Grid>
    </ComponentWrapper>
  </ComponentSkeleton>
);

export default ComponentSpeeddial;
