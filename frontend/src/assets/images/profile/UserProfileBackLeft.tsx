// material-ui
import { alpha, useTheme } from '@mui/material/styles';

// ==============================|| USER PROFILE - CARD BACK LEFT ||============================== //

const UserProfileBackLeft = () => {
  const theme = useTheme();

  return (
    <svg width="333" height="61" viewBox="0 0 333 61" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path
        opacity="0.4"
        d="M-0.322477 0.641086L-0.418408 0.55164L-9.20939 59.4297L23.6588 106.206L154.575 130.423C236.759 117.931 383.93 93.3326 315.142 94.879C246.355 96.4253 215.362 64.2785 215.362 64.2785C215.362 64.2785 185.497 26.9117 117.864 33.4279C42.6115 40.6783 10.6143 10.8399 -0.322477 0.641086Z"
        fill={alpha(theme.palette.primary.light, 0.4)}
      />
    </svg>
  );
};

export default UserProfileBackLeft;
