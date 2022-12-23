// project import
import SimpleBar from 'components/third-party/SimpleBar';
import Navigation from './Navigation';

// ==============================|| DRWAER CONTENT ||============================== //

const DrawerContent = ({ searchValue }: { searchValue?: string }) => (
  <SimpleBar
    sx={{
      height: { xs: 'calc(100vh - 70px)', md: 'calc(100% - 70px)' },
      '& .simplebar-content': {
        display: 'flex',
        flexDirection: 'column'
      }
    }}
  >
    <Navigation searchValue={searchValue} />
  </SimpleBar>
);

export default DrawerContent;
