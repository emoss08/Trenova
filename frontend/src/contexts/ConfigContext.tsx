import { createContext, ReactNode } from 'react';

// project import
import config from 'config';
import useLocalStorage from 'hooks/useLocalStorage';

// types
import { CustomizationProps, FontFamily, PresetColor, ThemeDirection, ThemeMode } from 'types/config';

// initial state
const initialState: CustomizationProps = {
  ...config,
  onChangeContainer: () => {},
  onChangeMode: (mode: ThemeMode) => {},
  onChangePresetColor: (theme: PresetColor) => {},
  onChangeDirection: (direction: ThemeDirection) => {},
  onChangeMiniDrawer: (miniDrawer: boolean) => {},
  onChangeFontFamily: (fontFamily: FontFamily) => {}
};

// ==============================|| CONFIG CONTEXT & PROVIDER ||============================== //

const ConfigContext = createContext(initialState);

type ConfigProviderProps = {
  children: ReactNode;
};

function ConfigProvider({ children }: ConfigProviderProps) {
  const [config, setConfig] = useLocalStorage('mantis-react-ts-config', initialState);

  const onChangeContainer = () => {
    setConfig({
      ...config,
      container: !config.container
    });
  };

  const onChangeMode = (mode: ThemeMode) => {
    setConfig({
      ...config,
      mode
    });
  };

  const onChangePresetColor = (theme: PresetColor) => {
    setConfig({
      ...config,
      presetColor: theme
    });
  };

  const onChangeDirection = (direction: ThemeDirection) => {
    setConfig({
      ...config,
      themeDirection: direction
    });
  };

  const onChangeMiniDrawer = (miniDrawer: boolean) => {
    setConfig({
      ...config,
      miniDrawer
    });
  };

  const onChangeFontFamily = (fontFamily: FontFamily) => {
    setConfig({
      ...config,
      fontFamily
    });
  };

  return (
    <ConfigContext.Provider
      value={{
        ...config,
        onChangeContainer,
        onChangeMode,
        onChangePresetColor,
        onChangeDirection,
        onChangeMiniDrawer,
        onChangeFontFamily
      }}
    >
      {children}
    </ConfigContext.Provider>
  );
}

export { ConfigProvider, ConfigContext };
