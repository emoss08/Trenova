// material-ui
import { Grid, Stack, Typography } from '@mui/material';

// project imports
import MainCard from 'components/MainCard';
import { GenericCardProps } from 'types/root';

// ==============================|| REPORT CARD ||============================== //

interface ReportCardProps extends GenericCardProps {}

const ReportCard = ({ primary, secondary, iconPrimary, color }: ReportCardProps) => {
  const IconPrimary = iconPrimary!;
  const primaryIcon = iconPrimary ? <IconPrimary fontSize="large" /> : null;

  return (
    <MainCard>
      <Grid container justifyContent="space-between" alignItems="center">
        <Grid item>
          <Stack spacing={1}>
            <Typography variant="h4">{primary}</Typography>
            <Typography variant="body1" color="secondary">
              {secondary}
            </Typography>
          </Stack>
        </Grid>
        <Grid item>
          <Typography variant="h2" style={{ color }}>
            {primaryIcon}
          </Typography>
        </Grid>
      </Grid>
    </MainCard>
  );
};

export default ReportCard;
