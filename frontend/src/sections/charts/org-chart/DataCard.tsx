// material-ui
import { useTheme } from '@mui/material/styles';
import { Chip, Stack, Typography } from '@mui/material';

// project imports
import Avatar from 'components/@extended/Avatar';
import IconButton from 'components/@extended/IconButton';
import MainCard from 'components/MainCard';
import { DataCardMiddleware } from 'types/org-chart';

// assets
import { FacebookFilled, LinkedinFilled, TwitterSquareFilled } from '@ant-design/icons';

// ==============================|| ORGANIZATION CHART - DATACARD||============================== //

function DataCard({ name, role, avatar, linkedin, facebook, skype, root }: DataCardMiddleware) {
  const linkHandler = (link: string) => {
    window.open(link);
  };
  const theme = useTheme();

  const subTree = theme.palette.secondary.lighter + 40;
  const rootTree = theme.palette.primary.lighter + 60;

  return (
    <MainCard
      sx={{
        bgcolor: root ? rootTree : subTree,
        border: root ? `1px solid ${theme.palette.primary.light} !important` : `1px solid ${theme.palette.secondary.light} !important`,
        width: 'max-content',
        m: '0px auto',
        p: 1.5
      }}
      border={false}
      content={false}
    >
      <Stack direction="row" spacing={2}>
        <Avatar sx={{ mt: 0.3 }} src={avatar} size="sm" />
        <Stack spacing={1.5}>
          <Stack alignItems="flex-start">
            <Typography variant="subtitle1" sx={{ color: root ? theme.palette.primary.main : theme.palette.text.primary }}>
              {name}
            </Typography>
            {!root && (
              <Chip
                label={role}
                sx={{ fontSize: '0.675rem', '& .MuiChip-label': { px: 0.75 }, width: 'max-content' }}
                color="primary"
                variant="outlined"
                size="small"
              />
            )}
            {root && (
              <Typography sx={{ color: theme.palette.primary.darker }} variant="caption">
                {role}
              </Typography>
            )}
          </Stack>
          <Stack spacing={0} direction="row">
            <IconButton color="secondary" onClick={() => linkHandler(linkedin)} size="small">
              <LinkedinFilled style={{ fontSize: '1.15rem', color: theme.palette.secondary[600] }} />
            </IconButton>
            <IconButton color="primary" onClick={() => linkHandler(facebook)} size="small">
              <FacebookFilled style={{ fontSize: '1.15rem', color: theme.palette.primary[600] }} />
            </IconButton>
            <IconButton color="info" onClick={() => linkHandler(skype)} size="small">
              <TwitterSquareFilled style={{ fontSize: '1.15rem' }} />
            </IconButton>
          </Stack>
        </Stack>
      </Stack>
    </MainCard>
  );
}

export default DataCard;
