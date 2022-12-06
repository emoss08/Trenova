/* eslint-disable jsx-a11y/anchor-is-valid */
import React, {useEffect, useRef} from 'react'
import ApexCharts, {ApexOptions} from 'apexcharts'
import {getCSS, getCSSVariableValue} from '../../../assets/ts/_utils'
import {useThemeMode} from '../../layout/theme-mode/ThemeModeProvider'

type Props = {
  className: string
  title: string
  description: string
  change: string
  color: string
}

const StatisticsWidget3: React.FC<Props> = ({className, title, description, change, color}) => {
  const chartRef = useRef<HTMLDivElement | null>(null)
  const {mode} = useThemeMode()
  const refreshChart = () => {
    if (!chartRef.current) {
      return
    }

    const height = parseInt(getCSS(chartRef.current, 'height'))
    const labelColor = getCSSVariableValue('--kt-gray-800')
    const baseColor = getCSSVariableValue('--kt-' + color)
    const lightColor = getCSSVariableValue('--kt-' + color + '-light')

    const chart = new ApexCharts(
      chartRef.current,
      getChartOptions(height, labelColor, baseColor, lightColor)
    )
    if (chart) {
      chart.render()
    }

    return chart
  }

  useEffect(() => {
    const chart = refreshChart()

    return () => {
      if (chart) {
        chart.destroy()
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [chartRef, mode])

  return (
    <div className={`card ${className}`}>
      {/* begin::Body */}
      <div className='card-body d-flex flex-column p-0'>
        <div className='d-flex flex-stack flex-grow-1 card-p'>
          <div className='d-flex flex-column me-2'>
            <a href='#' className='text-dark text-hover-primary fw-bold fs-3'>
              {title}
            </a>

            <span
              className='text-muted fw-semibold mt-1'
              dangerouslySetInnerHTML={{__html: description}}
            ></span>
          </div>

          <span className='symbol symbol-50px'>
            <span className={`symbol-label fs-5 fw-bold bg-light-${color} text-${color}`}>
              {change}
            </span>
          </span>
        </div>
        <div
          ref={chartRef}
          className='statistics-widget-3-chart card-rounded-bottom'
          style={{height: '150px'}}
        ></div>
      </div>
      {/* end::Body */}
    </div>
  )
}

export {StatisticsWidget3}

function getChartOptions(
  height: number,
  labelColor: string,
  baseColor: string,
  lightColor: string
): ApexOptions {
  const options: ApexOptions = {
    series: [
      {
        name: 'Net Profit',
        data: [30, 45, 32, 70, 40],
      },
    ],
    chart: {
      fontFamily: 'inherit',
      type: 'area',
      height: height,
      toolbar: {
        show: false,
      },
      zoom: {
        enabled: false,
      },
      sparkline: {
        enabled: true,
      },
    },
    plotOptions: {},
    legend: {
      show: false,
    },
    dataLabels: {
      enabled: false,
    },
    fill: {
      type: 'solid',
      opacity: 1,
    },
    stroke: {
      curve: 'smooth',
      show: true,
      width: 3,
      colors: [baseColor],
    },
    xaxis: {
      categories: ['Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul'],
      axisBorder: {
        show: false,
      },
      axisTicks: {
        show: false,
      },
      labels: {
        show: false,
        style: {
          colors: labelColor,
          fontSize: '12px',
        },
      },
      crosshairs: {
        show: false,
        position: 'front',
        stroke: {
          color: '#E4E6EF',
          width: 1,
          dashArray: 3,
        },
      },
      tooltip: {
        enabled: false,
      },
    },
    yaxis: {
      min: 0,
      max: 80,
      labels: {
        show: false,
        style: {
          colors: labelColor,
          fontSize: '12px',
        },
      },
    },
    states: {
      normal: {
        filter: {
          type: 'none',
          value: 0,
        },
      },
      hover: {
        filter: {
          type: 'none',
          value: 0,
        },
      },
      active: {
        allowMultipleDataPointsSelection: false,
        filter: {
          type: 'none',
          value: 0,
        },
      },
    },
    tooltip: {
      style: {
        fontSize: '12px',
      },
      y: {
        formatter: function (val) {
          return '$' + val + ' thousands'
        },
      },
    },
    colors: [lightColor],
    markers: {
      colors: [lightColor],
      strokeColors: [baseColor],
      strokeWidth: 3,
    },
  }
  return options
}
