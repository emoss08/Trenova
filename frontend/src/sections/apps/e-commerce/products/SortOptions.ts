// ==============================|| PRODUCT GRID - SORT FILTER ||============================== //
// project imports
import { SortOptionsProps } from 'types/e-commerce';

const SortOptions: SortOptionsProps[] = [
  {
    value: 'high',
    label: 'Price: High To Low'
  },
  {
    value: 'low',
    label: 'Price: Low To High'
  },
  {
    value: 'popularity',
    label: 'Popularity'
  },
  {
    value: 'discount',
    label: 'Discount'
  },
  {
    value: 'new',
    label: 'Fresh Arrivals'
  }
];

export default SortOptions;
