// project imports
import { ColorsOptionsProps } from 'types/e-commerce';

// ==============================|| PRODUCT - COLOR FILTER ||============================== //

const ColorsOptions: ColorsOptionsProps[] = [
  {
    label: 'Light Primary',
    value: 'primary200',
    bg: 'primary.200'
  },
  {
    label: 'Dark Primary',
    value: 'primaryDark',
    bg: 'primary.dark'
  },
  {
    label: 'Light Secondary',
    value: 'secondary200',
    bg: 'secondary.200'
  },
  {
    label: 'Secondary',
    value: 'secondaryMain',
    bg: 'secondary.main'
  },
  {
    label: 'Light Green',
    value: 'successLight',
    bg: 'success.light'
  },
  {
    label: 'Green',
    value: 'successMain',
    bg: 'success.main'
  },
  {
    label: 'Dark Green',
    value: 'successDark',
    bg: 'success.dark'
  },
  {
    label: 'Light Red',
    value: 'errorLight',
    bg: 'error.light'
  },
  {
    label: 'Red',
    value: 'errorMain',
    bg: 'error.main'
  },
  {
    label: 'Dark Red',
    value: 'errorDark',
    bg: 'error.dark'
  },
  {
    label: 'Yellow',
    value: 'warningMain',
    bg: 'warning.main'
  },
  {
    label: 'Dark Yellow',
    value: 'warningDark',
    bg: 'warning.dark'
  }
];

export default ColorsOptions;
