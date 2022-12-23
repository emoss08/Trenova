// material-ui
import { Grid } from '@mui/material';

// project imports
import ToDoList from 'sections/widget/data/ToDoList';
import TrafficSources from 'sections/widget/data/TrafficSources';
import TeamMembers from 'sections/widget/data/TeamMembers';

import UserActivity from 'sections/widget/data/UserActivity';
import LatestMessages from 'sections/widget/data/LatestMessages';

import ProjectTable from 'sections/widget/data/ProjectTable';
import ProductSales from 'sections/widget/data/ProductSales';

import TasksCard from 'sections/widget/data/TasksCard';
import ApplicationSales from 'sections/widget/data/ApplicationSales';

import ActiveTickets from 'sections/widget/data/ActiveTickets';
import LatestPosts from 'sections/widget/data/LatestPosts';

import FeedsCard from 'sections/widget/data/FeedsCard';
import LatestCustomers from 'sections/widget/data/LatestCustomers';

import LatestOrder from 'sections/widget/data/LatestOrder';

import IncomingRequests from 'sections/widget/data/IncomingRequests';
import TotalRevenue from 'sections/widget/data/TotalRevenue';
import NewCustomers from 'sections/widget/data/NewCustomers';

import RecentTickets from 'sections/widget/data/RecentTickets';

// ===========================|| WIDGET - DATA ||=========================== //

const WidgetData = () => (
  <Grid container spacing={3}>
    {/* row 1 */}
    <Grid item xs={12} lg={4} md={6}>
      <ToDoList />
    </Grid>
    <Grid item xs={12} lg={4} md={6}>
      <TrafficSources />
    </Grid>
    <Grid item xs={12} lg={4} md={12}>
      <TeamMembers />
    </Grid>

    {/* row 2 */}
    <Grid item xs={12} md={7} lg={6}>
      <LatestMessages />
    </Grid>
    <Grid item xs={12} md={5} lg={6}>
      <UserActivity />
    </Grid>

    {/* row 3 */}
    <Grid item xs={12} lg={6} md={6}>
      <ProjectTable />
    </Grid>
    <Grid item xs={12} lg={6} md={6}>
      <ProductSales />
    </Grid>

    {/* row 4 */}
    <Grid item xs={12} md={4}>
      <TasksCard />
    </Grid>
    <Grid item xs={12} md={8}>
      <ApplicationSales />
    </Grid>

    {/* row 5 */}
    <Grid item xs={12} md={8}>
      <ActiveTickets />
    </Grid>
    <Grid item xs={12} md={4}>
      <LatestPosts />
    </Grid>

    {/* row 6 */}
    <Grid item xs={12} md={5} lg={4}>
      <FeedsCard />
    </Grid>
    <Grid item xs={12} md={7} lg={8}>
      <LatestCustomers />
    </Grid>

    {/* row 7 */}
    <Grid item xs={12}>
      <LatestOrder />
    </Grid>

    {/* row 8 */}
    <Grid item xs={12} lg={4} md={6}>
      <IncomingRequests />
    </Grid>
    <Grid item xs={12} lg={4} md={6}>
      <TotalRevenue />
    </Grid>
    <Grid item xs={12} lg={4} md={12}>
      <NewCustomers />
    </Grid>

    {/* row 9 */}
    <Grid item xs={12} md={12} lg={8}>
      <RecentTickets />
    </Grid>
  </Grid>
);

export default WidgetData;
