<template>
  <div class="card-panel">
    <div :class="'card-panel-icon-wrapper '+iconColor">
      <i :class="'el-card-icon '+icon" />
    </div>
    <div class="card-panel-description">
      <div class="card-panel-text">
        {{ text }}
      </div>
      <count-to
        v-if="countto"
        :start-val="startVal"
        :end-val="endVal"
        :duration="duration"
        :suffix="suffix"
        :prefix="prefix"
        :separator="separator"
        class="card-panel-num"
      />
      <div v-else class="card-panel-num">{{ cvalue }}</div>
    </div>
  </div>
</template>

<script>
import CountTo from 'vue-count-to'

export default {
  components: {
    CountTo
  },
  props: {
    countto: {
      type: Boolean,
      default: true
    },
    cvalue: {
      type: String,
      default: ''
    },
    startVal: {
      type: Number,
      default: 0
    },
    endVal: {
      type: Number,
      default: 0
    },
    suffix: {
      type: String,
      default: ''
    },
    separator: {
      type: String,
      default: ''
    },
    prefix: {
      type: String,
      default: ''
    },
    text: {
      type: String,
      default: ''
    },
    icon: {
      type: String,
      default: ''
    },
    iconColor: {
      type: String,
      default: ''
    },
    duration: {
      type: Number,
      default: 1000
    }
  },
  methods: {}
}
</script>

<style lang="scss" scoped>
.card-panel {
  height: 108px;
  cursor: pointer;
  font-size: 12px;
  position: relative;
  overflow: hidden;
  color: #666;
  background: #fff;
  box-shadow: 4px 4px 40px #ecf2f3;
  border-color: #ecf2f3;

  &:hover {
    .card-panel-icon-wrapper {
      color: #fff;
    }

    .icon-green {
      background: #40c9c6;
    }

    .icon-blue {
      background: #36a3f7;
    }

    .icon-red {
      background: #f4516c;
    }
  }

  .icon-green {
    color: #40c9c6;
  }

  .icon-blue {
    color: #36a3f7;
  }

  .icon-red {
    color: #f4516c;
  }

  .card-panel-icon-wrapper {
    float: left;
    margin: 14px 0 0 14px;
    padding: 16px;
    transition: all 0.38s ease-out;
    border-radius: 6px;
  }

  .card-panel-icon {
    float: left;
    font-size: 48px;
  }

  .el-card-icon {
    float: left;
    font-size: 48px;
  }

  .card-panel-description {
    float: left;
    font-weight: bold;
    margin: 26px 26px 26px 20px;

    .card-panel-text {
      line-height: 18px;
      color: rgba(0, 0, 0, 0.45);
      font-size: 16px;
      margin-bottom: 12px;
    }

    .card-panel-num {
      font-size: 20px;
    }
  }
}

@media (max-width: 550px) {
  .card-panel-description {
    display: none;
  }

  .card-panel-icon-wrapper {
    float: none !important;
    width: 100%;
    height: 100%;
    margin: 0 !important;

    .svg-icon {
      display: block;
      margin: 14px auto !important;
      float: none !important;
    }
  }
}
</style>
