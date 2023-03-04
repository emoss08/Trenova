// @ts-nocheck
/* eslint-disable jsx-a11y/anchor-is-valid */
import React, {useEffect, useRef} from 'react'
import ApexCharts, {ApexOptions} from 'apexcharts'
import {getCSSVariableValue} from '../../../assets/ts/_utils'
import {useThemeMode} from '../../layout/theme-mode/ThemeModeProvider'

type Props = {
  className: string
  chartHeight: string
  backGroundColor: string
}

const MixedWidget13: React.FC<Props> = ({className, backGroundColor, chartHeight}) => {
  const chartRef = useRef<HTMLDivElement | null>(null)
  const {mode} = useThemeMode()

  useEffect(() => {
    const chart = refreshChart()

    return () => {
      if (chart) {
        chart.destroy()
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [chartRef, mode])

  const refreshChart = () => {
    if (!chartRef.current) {
      return
    }

    const chart = new ApexCharts(chartRef.current, chartOptions(chartHeight))
    if (chart) {
      chart.render()
    }

    return chart
  }

  return (
    <div
      className={`card ${className} theme-dark-bg-body`}
      style={{backgroundColor: backGroundColor}}
    >
      {/* begin::Body */}
      <div className='card-body d-flex flex-column'>
        {/* begin::Wrapper */}
        <div className='d-flex flex-column flex-grow-1'>
          {/* begin::Title                    */}
          <a href='#' className='text-dark text-hover-primary fw-bolder fs-3'>
            Earnings
          </a>
          {/* end::Title */}

          <div
            ref={chartRef}
            className='mixed-widget-13-chart'
            style={{height: chartHeight, minHeight: chartHeight}}
          ></div>
        </div>
        {/* end::Wrapper */}

        {/* begin::Stats */}
        <div className='pt-5'>
          {/* begin::Symbol */}
          <span className='text-dark fw-bolder fs-2x lh-0'>$</span>
          {/* end::Symbol */}

          {/* begin::Number */}
          <span className='text-dark fw-bolder fs-3x me-2 lh-0'>560</span>
          {/* end::Number */}

          {/* begin::Text */}
          <span className='text-dark fw-bolder fs-6 lh-0'>+ 28% this week</span>
          {/* end::Text */}
        </div>
        {/* end::Stats */}
      </div>
    </div>
  )
}

const chartOptions = (chartHeight: string): ApexOptions => {
  const labelColor = getCSSVariableValue('--bs-gray-800')
  const strokeColor = getCSSVariableValue('--bs-gray-300') as string

  return {
    series: [
      {
        name: 'Net Profit',
        data: [15, 25, 15, 40, 20, 50],
      },
    ],
    grid: {
      show: false,
      padding: {
        top: 0,
        bottom: 0,
        left: 0,
        right: 0,
      },
    },
    chart: {
      fontFamily: 'inherit',
      type: 'area',
      height: chartHeight,
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
      type: 'gradient',
      gradient: {
        opacityFrom: 0.4,
        opacityTo: 0,
        stops: [20, 120, 120, 120],
      },
    },
    stroke: {
      curve: 'smooth',
      show: true,
      width: 3,
      colors: ['#FFFFFF'],
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
          color: strokeColor,
          width: 1,
          dashArray: 3,
        },
      },
      tooltip: {
        enabled: true,
        formatter: undefined,
        offsetY: 0,
        style: {
          fontSize: '12px',
        },
      },
    },
    yaxis: {
      min: 0,
      max: 60,
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
    colors: ['#ffffff'],
    markers: {
      colors: [labelColor],
      strokeColor: [strokeColor],
      strokeWidth: 3,
    },
  }
}

export {MixedWidget13}
