import { useState, useEffect } from 'react';

// material-ui
import { useTheme } from '@mui/material/styles';

// project import
import useConfig from 'hooks/useConfig';

// third-party
import ReactApexChart, { Props as ChartProps } from 'react-apexcharts';

// ==============================|| SALES CARD CHART ||============================== //

const SalesCardChart = () => {
  const theme = useTheme();
  const { mode } = useConfig();

  const areaChartOptions = {
    chart: {
      id: 'new-stack-chart',
      sparkline: {
        enabled: true
      },
      height: 100,
      type: 'bar',
      toolbar: {
        show: false
      }
    },
    plotOptions: {
      bar: {
        borderRadius: 0
      }
    },
    dataLabels: {
      enabled: false,
      offsetY: -20,
      style: {
        fontSize: '12px',
        colors: ['#304758']
      }
    },
    xaxis: {
      axisBorder: {
        show: false
      },
      axisTicks: {
        show: false
      },
      labels: {
        show: false
      },
      crosshairs: {
        fill: {
          type: 'gradient',
          gradient: {
            colorFrom: '#D8E3F0',
            colorTo: '#BED1E6',
            stops: [0, 100],
            opacityFrom: 0.4,
            opacityTo: 0.5
          }
        }
      }
    },
    yaxis: {
      axisBorder: {
        show: false
      },
      axisTicks: {
        show: false
      },
      labels: {
        show: false
      }
    },
    tooltip: {
      theme: mode === 'dark' ? 'dark' : 'light',
      x: {
        show: false
      },
      y: {
        formatter(val: number) {
          return `$ ${val}`;
        }
      }
    },
    grid: {
      show: false
    }
  };

  const { primary, secondary } = theme.palette.text;
  const line = theme.palette.divider;

  const [options, setOptions] = useState<ChartProps>(areaChartOptions);

  useEffect(() => {
    setOptions((prevState) => ({
      ...prevState,
      colors: [theme.palette.warning.main],
      tooltip: {
        theme: mode === 'dark' ? 'dark' : 'light'
      }
    }));
  }, [mode, primary, secondary, line, theme]);

  const [series] = useState([
    {
      name: 'Sales',
      data: [
        220, 230, 240, 220, 225, 215, 205, 195, 185, 150, 185, 195, 80, 205, 215, 225, 240, 225, 215, 205, 80, 215, 225, 240, 215, 210, 180
      ]
    }
  ]);

  return <ReactApexChart options={options} series={series} type="bar" height={100} />;
};

export default SalesCardChart;
