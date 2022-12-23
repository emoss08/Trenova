// material-ui
import { useTheme } from '@mui/material/styles';

const StandardLogo = () => {
  const theme = useTheme();

  return (
    <svg width="36" height="18" viewBox="0 0 36 18" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path d="M18 0.251007L35.5 17.751H28.4137L18 7.33735L7.58635 17.751H0.5L18 0.251007Z" fill={theme.palette.primary.main} />
    </svg>
  );
};

export default StandardLogo;
