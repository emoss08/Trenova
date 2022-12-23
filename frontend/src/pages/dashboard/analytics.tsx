import { useState } from 'react';

// material-ui
import { useTheme } from '@mui/material/styles';
import {
  Box,
  Grid,
  List,
  ListItemButton,
  ListItemText,
  MenuItem,
  Select,
  SelectChangeEvent,
  Stack,
  TextField,
  ToggleButton,
  ToggleButtonGroup,
  Typography
} from '@mui/material';

// project import
import IncomeChart from 'sections/dashboard/analytics/IncomeChart';
import MarketingCardChart from 'sections/dashboard/analytics/MarketingCardChart';
import OrdersCardChart from 'sections/dashboard/analytics/OrdersCardChart';
import OrdersList from 'sections/dashboard/analytics/OrdersList';
import PageViews from 'sections/dashboard/analytics/PageViews';
import ReportChart from 'sections/dashboard/analytics/ReportChart';
import SalesCardChart from 'sections/dashboard/analytics/SalesCardChart';
import SalesChart from 'sections/dashboard/SalesChart';
import TransactionHistory from 'sections/dashboard/analytics/TransactionHistory';
import UsersCardChart from 'sections/dashboard/analytics/UsersCardChart';
import LabelledTasks from 'sections/dashboard/analytics/LabelledTasks';
import ReaderCard from 'sections/dashboard/analytics/ReaderCard';
import AcquisitionChannels from 'sections/dashboard/analytics/AcquisitionChannels';
import MainCard from 'components/MainCard';
import IconButton from 'components/@extended/IconButton';
import AnalyticsDataCard from 'components/cards/statistics/AnalyticsDataCard';

// assets
import { DownloadOutlined, CaretDownOutlined } from '@ant-design/icons';

// sales report status
const status = [
  {
    value: 'today',
    label: 'Today'
  },
  {
    value: 'month',
    label: 'This Month'
  },
  {
    value: 'year',
    label: 'This Year'
  }
];

// ==============================|| DASHBOARD - ANALYTICS ||============================== //

const DashboardAnalytics = () => {
  const theme = useTheme();
  const [value, setValue] = useState('today');
  const [slot, setSlot] = useState('week');
  const [quantity, setQuantity] = useState('By volume');

  const handleQuantity = (e: SelectChangeEvent) => {
    setQuantity(e.target.value as string);
  };

  const handleChange = (event: React.MouseEvent<HTMLElement>, newAlignment: string) => {
    if (newAlignment) setSlot(newAlignment);
  };

  return (
    <Grid container rowSpacing={4.5} columnSpacing={3}>
      {/* row 1 */}

      <Grid item xs={12} sm={6} md={4} lg={3}>
        <AnalyticsDataCard title="Total Users" count="78,250" percentage={70.5}>
          <UsersCardChart />
        </AnalyticsDataCard>
      </Grid>
      <Grid item xs={12} sm={6} md={4} lg={3}>
        <AnalyticsDataCard title="Total Order" count="18,800" percentage={27.4} isLoss color="warning">
          <OrdersCardChart />
        </AnalyticsDataCard>
      </Grid>
      <Grid item xs={12} sm={6} md={4} lg={3}>
        <AnalyticsDataCard title="Total Sales" count="$35,078" percentage={27.4} isLoss color="warning">
          <SalesCardChart />
        </AnalyticsDataCard>
      </Grid>
      <Grid item xs={12} sm={6} md={4} lg={3}>
        <AnalyticsDataCard title="Total Marketing" count="$1,12,083" percentage={70.5}>
          <MarketingCardChart />
        </AnalyticsDataCard>
      </Grid>

      <Grid item md={8} sx={{ display: { sm: 'none', md: 'block', lg: 'none' } }} />

      {/* row 2 */}
      <Grid item xs={12} md={7} lg={8}>
        <Grid container alignItems="center" justifyContent="space-between">
          <Grid item>
            <Typography variant="h5">Income Overview</Typography>
          </Grid>
        </Grid>
        <MainCard content={false} sx={{ mt: 1.5 }}>
          <Grid item>
            <Grid container>
              <Grid item xs={12} sm={6}>
                <Stack sx={{ ml: 2, mt: 3 }} alignItems={{ xs: 'center', sm: 'flex-start' }}>
                  <Stack direction="row" alignItems="center">
                    <CaretDownOutlined style={{ color: theme.palette.error.main, paddingRight: '4px' }} />
                    <Typography color={theme.palette.error.main}>$1,12,900 (45.67%)</Typography>
                  </Stack>
                  <Typography color="textSecondary" sx={{ display: 'block' }}>
                    Compare to : 01 Dec 2021-08 Jan 2022
                  </Typography>
                </Stack>
              </Grid>
              <Grid item xs={12} sm={6}>
                <Stack
                  direction="row"
                  spacing={1}
                  alignItems="center"
                  justifyContent={{ xs: 'center', sm: 'flex-end' }}
                  sx={{ mt: 3, mr: 2 }}
                >
                  <ToggleButtonGroup exclusive onChange={handleChange} size="small" value={slot}>
                    <ToggleButton disabled={slot === 'week'} value="week" sx={{ px: 2, py: 0.5 }}>
                      Week
                    </ToggleButton>
                    <ToggleButton disabled={slot === 'month'} value="month" sx={{ px: 2, py: 0.5 }}>
                      Month
                    </ToggleButton>
                  </ToggleButtonGroup>
                  <Select value={quantity} onChange={handleQuantity} size="small">
                    <MenuItem value="By volume">By Volume</MenuItem>
                    <MenuItem value="By margin">By Margin</MenuItem>
                    <MenuItem value="By sales">By Sales</MenuItem>
                  </Select>
                  <IconButton
                    size="small"
                    sx={{
                      border: `1px solid ${theme.palette.grey[400]}`,
                      '&:hover': { backgroundColor: 'transparent' }
                    }}
                  >
                    <DownloadOutlined style={{ color: theme.palette.grey[900] }} />
                  </IconButton>
                </Stack>
              </Grid>
            </Grid>
          </Grid>
          <Box sx={{ pt: 1 }}>
            <IncomeChart slot={slot} quantity={quantity} />
          </Box>
        </MainCard>
      </Grid>
      <Grid item xs={12} md={5} lg={4}>
        <PageViews />
      </Grid>

      {/* row 3 */}
      <Grid item xs={12} md={7} lg={8}>
        <Grid container alignItems="center" justifyContent="space-between">
          <Grid item>
            <Typography variant="h5">Recent Orders</Typography>
          </Grid>
          <Grid item />
        </Grid>
        <MainCard sx={{ mt: 2 }} content={false}>
          <OrdersList />
        </MainCard>
      </Grid>
      <Grid item xs={12} md={5} lg={4}>
        <Grid container alignItems="center" justifyContent="space-between">
          <Grid item>
            <Typography variant="h5">Analytics Report</Typography>
          </Grid>
          <Grid item />
        </Grid>
        <MainCard sx={{ mt: 2 }} content={false}>
          <List sx={{ p: 0, '& .MuiListItemButton-root': { py: 2 } }}>
            <ListItemButton divider>
              <ListItemText primary="Company Finance Growth" />
              <Typography variant="h5">+45.14%</Typography>
            </ListItemButton>
            <ListItemButton divider>
              <ListItemText primary="Company Expenses Ratio" />
              <Typography variant="h5">0.58%</Typography>
            </ListItemButton>
            <ListItemButton>
              <ListItemText primary="Business Risk Cases" />
              <Typography variant="h5">Low</Typography>
            </ListItemButton>
          </List>
          <ReportChart />
        </MainCard>
      </Grid>

      {/* row 4 */}
      <Grid item xs={12} md={7} lg={8}>
        <Grid container alignItems="center" justifyContent="space-between">
          <Grid item>
            <Typography variant="h5">Sales Report</Typography>
          </Grid>
          <Grid item>
            <TextField
              id="standard-select-currency"
              size="small"
              select
              value={value}
              onChange={(e) => setValue(e.target.value)}
              sx={{ '& .MuiInputBase-input': { py: 0.75, fontSize: '0.875rem' } }}
            >
              {status.map((option) => (
                <MenuItem key={option.value} value={option.value}>
                  {option.label}
                </MenuItem>
              ))}
            </TextField>
          </Grid>
        </Grid>
        <SalesChart />
      </Grid>
      <Grid item xs={12} md={5} lg={4}>
        <TransactionHistory />
      </Grid>

      {/* row 5 */}
      <Grid item xs={12} md={7} lg={8}>
        <Stack spacing={3}>
          <LabelledTasks />
          <ReaderCard />
        </Stack>
      </Grid>
      <Grid item xs={12} md={5} lg={4}>
        <AcquisitionChannels />
      </Grid>
    </Grid>
  );
};

export default DashboardAnalytics;
