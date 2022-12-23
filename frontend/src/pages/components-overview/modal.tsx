// material-ui
import { Grid } from '@mui/material';

// project import
import ComponentHeader from 'components/cards/ComponentHeader';
import ComponentWrapper from 'sections/components-overview/ComponentWrapper';
import ComponentSkeleton from 'sections/components-overview/ComponentSkeleton';
import ServerModal from 'sections/components-overview/modal/ServerModal';
import BasicModal from 'sections/components-overview/modal/BasicModal';
import NestedModal from 'sections/components-overview/modal/NestedModal';
import TransitionsModal from 'sections/components-overview/modal/TransitionsModal';

// ==============================|| COMPONENTS - MODAL ||============================== //

const ComponentModal = () => (
  <ComponentSkeleton>
    <ComponentHeader
      title="Modal"
      caption="The modal component provides a solid foundation for creating dialogs, popovers, lightboxes, or whatever else."
      directory="src/pages/components-overview/modal"
      link="https://mui.com/material-ui/react-modal"
    />
    <ComponentWrapper>
      <Grid container spacing={3}>
        <Grid item xs={12}>
          <ServerModal />
        </Grid>
        <Grid item xs={12} sm={6} md={4}>
          <BasicModal />
        </Grid>
        <Grid item xs={12} sm={6} md={4}>
          <NestedModal />
        </Grid>
        <Grid item xs={12} sm={6} md={4}>
          <TransitionsModal />
        </Grid>
      </Grid>
    </ComponentWrapper>
  </ComponentSkeleton>
);

export default ComponentModal;
