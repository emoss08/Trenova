// material-ui
import { alpha, useTheme } from '@mui/material/styles';

// ==============================|| USER PROFILE - CARD BACK RIGHT ||============================== //

const UserProfileBackRight = () => {
  const theme = useTheme();

  return (
    <svg width="447" height="116" viewBox="0 0 447 116" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path
        opacity="0.4"
        d="M55.2678 22.3777C-49.5465 -14.1611 7.16534 -48.8529 136.242 -34.0647L214.579 -30.0724L448.26 -8.82579L459.956 104.858C396.401 148.386 406.862 51.7166 297.501 67.1292C188.139 82.5419 225.278 33.322 176.928 20.0906C128.579 6.8592 91.4243 34.9821 55.2678 22.3777Z"
        fill={alpha(theme.palette.primary.light, 0.4)}
      />
    </svg>
  );
};

export default UserProfileBackRight;
