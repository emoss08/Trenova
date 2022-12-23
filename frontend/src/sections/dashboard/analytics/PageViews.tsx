// material-ui
import { Grid, List, ListItemButton, ListItemText, Stack, Typography } from '@mui/material';

// project import
import MainCard from 'components/MainCard';

// ==============================|| PAGE VIEWS BY PAGE TITLE ||============================== //

function PageViews() {
  return (
    <>
      <Grid container alignItems="center" justifyContent="space-between">
        <Grid item>
          <Typography variant="h5">Page Views by Page Title</Typography>
        </Grid>
        <Grid item />
      </Grid>
      <MainCard sx={{ mt: 2 }} content={false}>
        <List sx={{ p: 0, '& .MuiListItemButton-root': { py: 2 } }}>
          <ListItemButton divider>
            <ListItemText
              primary={<Typography variant="subtitle1">Admin Home</Typography>}
              secondary={
                <Typography color="textSecondary" sx={{ display: 'inline' }}>
                  /demo/admin/index.html
                </Typography>
              }
            />
            <Stack alignItems="flex-end">
              <Typography variant="h5" color="primary">
                7755
              </Typography>
              <Typography variant="body2" color="textSecondary">
                31.74%
              </Typography>
            </Stack>
          </ListItemButton>
          <ListItemButton divider>
            <ListItemText
              primary={<Typography variant="subtitle1">Form Elements</Typography>}
              secondary={
                <Typography color="textSecondary" sx={{ display: 'inline' }}>
                  /demo/admin/forms.html
                </Typography>
              }
            />
            <Stack alignItems="flex-end">
              <Typography variant="h5" color="primary">
                5215
              </Typography>
              <Typography variant="body2" color="textSecondary" sx={{ display: 'block' }}>
                28.53%
              </Typography>
            </Stack>
          </ListItemButton>
          <ListItemButton divider>
            <ListItemText
              primary={<Typography variant="subtitle1">Utilities</Typography>}
              secondary={
                <Typography color="textSecondary" sx={{ display: 'inline' }}>
                  /demo/admin/util.html
                </Typography>
              }
            />
            <Stack alignItems="flex-end">
              <Typography variant="h5" color="primary">
                4848
              </Typography>
              <Typography variant="body2" color="textSecondary" sx={{ display: 'block' }}>
                25.35%
              </Typography>
            </Stack>
          </ListItemButton>
          <ListItemButton divider>
            <ListItemText
              primary={<Typography variant="subtitle1">Form Validation</Typography>}
              secondary={
                <Typography color="textSecondary" sx={{ display: 'inline' }}>
                  /demo/admin/validation.html
                </Typography>
              }
            />
            <Stack alignItems="flex-end">
              <Typography variant="h5" color="primary">
                3275
              </Typography>
              <Typography variant="body2" color="textSecondary" sx={{ display: 'block' }}>
                23.17%
              </Typography>
            </Stack>
          </ListItemButton>
          <ListItemButton divider>
            <ListItemText
              primary={<Typography variant="subtitle1">Modals</Typography>}
              secondary={
                <Typography color="textSecondary" sx={{ display: 'inline' }}>
                  /demo/admin/modals.html
                </Typography>
              }
            />
            <Stack alignItems="flex-end">
              <Typography variant="h5" color="primary">
                3003
              </Typography>
              <Typography variant="body2" color="textSecondary" sx={{ display: 'block' }}>
                22.21%
              </Typography>
            </Stack>
          </ListItemButton>
        </List>
      </MainCard>
    </>
  );
}
export default PageViews;
