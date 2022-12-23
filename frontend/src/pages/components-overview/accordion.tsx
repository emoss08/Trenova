// material-ui
import { Grid } from '@mui/material';

// project import
import ComponentHeader from 'components/cards/ComponentHeader';
import ComponentWrapper from 'sections/components-overview/ComponentWrapper';
import ComponentSkeleton from 'sections/components-overview/ComponentSkeleton';
import BasicAccordion from 'sections/components-overview/accordion/BasicAccordion';
import DisabledAccordion from 'sections/components-overview/accordion/DisabledAccordion';
import ControlledAccordion from 'sections/components-overview/accordion/ControlledAccordion';
import FixedAccordion from 'sections/components-overview/accordion/FixedAccordion';
import CustomizedAccordion from 'sections/components-overview/accordion/CustomizedAccordion';

// ==============================|| COMPONENTS - ACCORDION ||============================== //

const ComponentAccordion = () => (
  <ComponentSkeleton>
    <ComponentHeader
      title="Accordion"
      caption="Lists are continuous, vertical indexes of text or images."
      directory="src/pages/components-overview/accordion"
      link="https://mui.com/material-ui/react-accordion/"
    />
    <ComponentWrapper>
      <Grid container spacing={3}>
        <Grid item xs={12} lg={6}>
          <BasicAccordion />
        </Grid>
        <Grid item xs={12} lg={6}>
          <DisabledAccordion />
        </Grid>
        <Grid item xs={12} lg={6}>
          <ControlledAccordion />
        </Grid>
        <Grid item xs={12} lg={6}>
          <FixedAccordion />
        </Grid>
        <Grid item xs={12} lg={6}>
          <CustomizedAccordion />
        </Grid>
      </Grid>
    </ComponentWrapper>
  </ComponentSkeleton>
);

export default ComponentAccordion;
