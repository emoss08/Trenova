/* eslint-disable jsx-a11y/anchor-is-valid */
import React, {useEffect, useRef} from 'react'
import ApexCharts, {ApexOptions} from 'apexcharts'
import {MTSVG} from '../../../helpers'
import {getCSSVariableValue} from '../../../assets/ts/_utils'
import {Dropdown1} from '../../content/dropdown/Dropdown1'
import {useThemeMode} from '../../layout/theme-mode/ThemeModeProvider'

type Props = {
  className: string
  chartHeight: string
  chartColor: string
}

const MixedWidget6: React.FC<Props> = ({className, chartHeight, chartColor}) => {
  const chartRef = useRef<HTMLDivElement | null>(null)
  const {mode} = useThemeMode()
  const refreshChart = () => {
    if (!chartRef.current) {
      return
    }

    const chart = new ApexCharts(chartRef.current, chartOptions(chartHeight, chartColor))
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
      {/* begin::Beader */}
      <div className='card-header border-0 py-5'>
        <h3 className='card-title align-items-start flex-column'>
          <span className='card-label fw-bold fs-3 mb-1'>Sales Overview</span>

          <span className='text-muted fw-semibold fs-7'>Recent sales statistics</span>
        </h3>

        <div className='card-toolbar'>
          {/* begin::Menu */}
          <button
            type='button'
            className='btn btn-sm btn-icon btn-color-primary btn-active-light-primary'
            data-kt-menu-trigger='click'
            data-kt-menu-placement='bottom-end'
            data-kt-menu-flip='top-end'
          >
            <MTSVG path='/media/icons/duotune/general/gen024.svg' className='svg-icon-2' />
          </button>
          <Dropdown1 />
          {/* end::Menu */}
        </div>
      </div>
      {/* end::Header */}

      {/* begin::Body */}
      <div className='card-body p-0 d-flex flex-column'>
        {/* begin::Stats */}
        <div className='card-p pt-5 bg-body flex-grow-1'>
          {/* begin::Row */}
          <div className='row g-0'>
            {/* begin::Col */}
            <div className='col mr-8'>
              {/* begin::Label */}
              <div className='fs-7 text-muted fw-semibold'>Average Sale</div>
              {/* end::Label */}

              {/* begin::Stat */}
              <div className='d-flex align-items-center'>
                <div className='fs-4 fw-bold'>$650</div>
                <MTSVG
                  path='/media/icons/duotune/arrows/arr066.svg'
                  className='svg-icon-5 svg-icon-success ms-1'
                />
              </div>
              {/* end::Stat */}
            </div>
            {/* end::Col */}

            {/* begin::Col */}
            <div className='col'>
              {/* begin::Label */}
              <div className='fs-7 text-muted fw-semibold'>Commission</div>
              {/* end::Label */}

              {/* begin::Stat */}
              <div className='fs-4 fw-bold'>$233,600</div>
              {/* end::Stat */}
            </div>
            {/* end::Col */}
          </div>
          {/* end::Row */}

          {/* begin::Row */}
          <div className='row g-0 mt-8'>
            {/* begin::Col */}
            <div className='col mr-8'>
              {/* begin::Label */}
              <div className='fs-7 text-muted fw-semibold'>Annual Taxes 2019</div>
              {/* end::Label */}

              {/* begin::Stat */}
              <div className='fs-4 fw-bold'>$29,004</div>
              {/* end::Stat */}
            </div>
            {/* end::Col */}

            {/* begin::Col */}
            <div className='col'>
              {/* begin::Label */}
              <div className='fs-7 text-muted fw-semibold'>Annual Income</div>
              {/* end::Label */}

              {/* begin::Stat */}
              <div className='d-flex align-items-center'>
                <div className='fs-4 fw-bold'>$1,480,00</div>
                <MTSVG
                  path='/media/icons/duotune/arrows/arr065.svg'
                  className='svg-icon-5 svg-icon-danger ms-1'
                />
              </div>
              {/* end::Stat */}
            </div>
            {/* end::Col */}
          </div>
          {/* end::Row */}
        </div>
        {/* end::Stats */}

        {/* begin::Chart */}
        <div ref={chartRef} className='mixed-widget-3-chart card-rounded-bottom' data-kt-chart-color={chartColor} style={{height: chartHeight}}></div>
        {/* end::Chart */}
      </div>
      {/* end::Body */}
    </div>
  )
}

const chartOptions = (chartHeight: string, chartColor: string): ApexOptions => {
  const labelColor = getCSSVariableValue('--kt-gray-800')
  const strokeColor = getCSSVariableValue('--kt-gray-300')
  const baseColor = getCSSVariableValue('--kt-' + chartColor)
  const lightColor = getCSSVariableValue('--kt-' + chartColor + '-light')

  return {
    series: [
      {
        name: 'Net Profit',
        data: [30, 25, 45, 30, 55, 55],
      },
    ],
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
          color: strokeColor,
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
    colors: [lightColor],
    markers: {
      colors: [lightColor],
      strokeColors: [baseColor],
      strokeWidth: 3,
    },
  }
}

export {MixedWidget6}
