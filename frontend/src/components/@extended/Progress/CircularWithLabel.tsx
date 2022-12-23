// material-ui
import { Box, CircularProgress, CircularProgressProps, Typography } from '@mui/material';

// ==============================|| PROGRESS - CIRCULAR LABEL ||============================== //

export default function CircularWithLabel({ value, ...others }: CircularProgressProps) {
  return (
    <Box sx={{ position: 'relative', display: 'inline-flex' }}>
      <CircularProgress variant="determinate" value={value} {...others} />
      <Box
        sx={{
          top: 0,
          left: 0,
          bottom: 0,
          right: 0,
          position: 'absolute',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center'
        }}
      >
        <Typography variant="caption" component="div" color="text.secondary">{`${Math.round(value!)}%`}</Typography>
      </Box>
    </Box>
  );
}
