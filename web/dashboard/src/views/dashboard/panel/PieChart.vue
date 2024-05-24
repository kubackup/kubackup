<template>
  <div class="chart-wrapper" :style="{height:height,width:width}" />
</template>

<script>
import echarts from 'echarts'
import resize from '../mixins/resize'

require('echarts/theme/dark') // echarts theme
require('echarts/theme/macarons')
require('echarts/theme/shine')
require('echarts/theme/vintage')
require('echarts/theme/roma')
require('echarts/theme/infographic')

export default {
  mixins: [resize],
  props: {
    title: {
      type: String,
      default: ''
    },
    subtitle: {
      type: String,
      default: ''
    },
    seriesname: {
      type: String,
      default: ''
    },
    cdata: {
      type: Array,
      default: {}
    },
    theme: {
      type: String,
      default: ''
    },
    width: {
      type: String,
      default: '100%'
    },
    height: {
      type: String,
      default: '300px'
    }
  },
  data() {
    return {
      chart: null
    }
  },
  watch: {
    cdata(val) {
      if (this.chart) {
        this.refresh(val)
      }
    }
  },
  mounted() {
    this.$nextTick(() => {
      this.initChart()
    })
  },
  beforeDestroy() {
    if (!this.chart) {
      return
    }
    this.chart.dispose()
    this.chart = null
  },
  methods: {
    refresh(data) {
      var opt = this.chart.getOption()
      opt.series[0].data = data
      this.chart.setOption(opt)
    },
    initChart() {
      this.chart = echarts.init(this.$el, this.theme)

      this.chart.setOption({
        title: {
          text: this.title,
          subtext: this.subtitle,
          left: 'center'
        },
        tooltip: {
          trigger: 'item'
        },
        legend: {
          orient: 'vertical',
          left: 'left'
        },
        series: [
          {
            name: this.seriesname,
            type: 'pie',
            radius: '50%',
            data: this.cdata,
            emphasis: {
              itemStyle: {
                shadowBlur: 10,
                shadowOffsetX: 0,
                shadowColor: 'rgba(0, 0, 0, 0.5)'
              }
            }
          }
        ]
      }, true)
    }
  }
}
</script>
<style lang="scss" scoped>
.chart-wrapper {
  background: #fff;
  padding: 16px 16px 0;
  margin-bottom: 32px;
  box-shadow: 4px 4px 40px #ecf2f3;
  border-color: #ecf2f3;
}
</style>
