// material-ui
import { Grid, Stack } from '@mui/material';

// project import
import ComponentHeader from 'components/cards/ComponentHeader';
import ComponentWrapper from 'sections/components-overview/ComponentWrapper';
import ComponentSkeleton from 'sections/components-overview/ComponentSkeleton';
import BasicTreeView from 'sections/components-overview/tree-view/BasicTreeView';
import MultiSelectTreeView from 'sections/components-overview/tree-view/MultiSelectTreeView';
import ControlledTreeView from 'sections/components-overview/tree-view/ControlledTreeView';
import RichObjectTreeView from 'sections/components-overview/tree-view/RichObjectTreeView';
import DisabledTreeView from 'sections/components-overview/tree-view/DisabledTreeView';
import CustomizedTreeView from 'sections/components-overview/tree-view/CustomizedTreeView';
import GmailTreeView from 'sections/components-overview/tree-view/GmailTreeView';

// ==============================|| COMPONENTS - TREE VIEW ||============================== //

const ComponentTreeView = () => (
  <ComponentSkeleton>
    <ComponentHeader
      title="Tree View"
      caption="A tree view widget presents a hierarchical list."
      directory="src/pages/components-overview/treeview"
      link="https://mui.com/material-ui/react-tree-view/"
    />
    <ComponentWrapper>
      <Grid container spacing={3}>
        <Grid item xs={12} sm={6}>
          <Stack spacing={3}>
            <BasicTreeView />
            <MultiSelectTreeView />
            <ControlledTreeView />
            <RichObjectTreeView />
          </Stack>
        </Grid>
        <Grid item xs={12} sm={6}>
          <Stack spacing={3}>
            <DisabledTreeView />
            <CustomizedTreeView />
            <GmailTreeView />
          </Stack>
        </Grid>
      </Grid>
    </ComponentWrapper>
  </ComponentSkeleton>
);

export default ComponentTreeView;
