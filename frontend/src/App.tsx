// project import
import Routes from 'routes';
import ThemeCustomization from 'themes';
import Locales from 'components/Locales';
// import RTLLayout from 'components/RTLLayout';
import ScrollTop from 'components/ScrollTop';
import Snackbar from 'components/@extended/Snackbar';

// auth provider
import { AuthProvider } from 'contexts/MontaContext';

// ==============================|| APP - THEME, ROUTER, LOCAL  ||============================== //

const App = () => (
  <ThemeCustomization>
    {/* <RTLLayout> */}
    <Locales>
      <ScrollTop>
        <AuthProvider>
          <>
            <Routes />
            <Snackbar />
          </>
        </AuthProvider>
      </ScrollTop>
    </Locales>
    {/* </RTLLayout> */}
  </ThemeCustomization>
);

export default App;
