// material-ui
import { Box, LinearProgress, LinearProgressProps, Typography } from '@mui/material';

// ==============================|| PROGRESS - LINEAR WITH LABEL ||============================== //

export default function LinearWithLabel({ value, ...others }: LinearProgressProps) {
  return (
    <Box sx={{ display: 'flex', alignItems: 'center' }}>
      <Box sx={{ width: '100%', mr: 1 }}>
        <LinearProgress variant="determinate" value={value} {...others} />
      </Box>
      <Box sx={{ minWidth: 35 }}>
        <Typography variant="body2" color="text.secondary">{`${Math.round(value!)}%`}</Typography>
      </Box>
    </Box>
  );
}
