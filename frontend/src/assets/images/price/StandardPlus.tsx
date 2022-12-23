// material-ui
import { useTheme } from '@mui/material/styles';

const StandardPlus = () => {
  const theme = useTheme();

  return (
    <svg width="36" height="35" viewBox="0 0 36 35" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path
        d="M2.14067 15.527L4.93141 12.7362L4.93432 12.7333H10.1846L8.09556 14.8224L7.55619 15.3617L5.41692 17.501L5.68187 17.7667L17.6666 29.7507L29.9163 17.501L27.7763 15.3617L27.6257 15.2103L25.1486 12.7333H30.3989L30.4018 12.7362L32.5892 14.9235L35.1666 17.501L17.6666 35.001L0.166626 17.501L2.14067 15.527ZM17.6666 0.00100708L27.7785 10.1129H22.5282L17.6666 5.2513L12.805 10.1129H7.55474L17.6666 0.00100708Z"
        fill={theme.palette.primary.main}
      />
    </svg>
  );
};

export default StandardPlus;
