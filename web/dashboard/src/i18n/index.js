import Vue from 'vue'
import VueI18n from 'vue-i18n'

Vue.use(VueI18n) // 全局挂载
// element-ui自带多语言配置
import zhLocale from 'element-ui/lib/locale/lang/zh-CN'
import enLocale from 'element-ui/lib/locale/lang/en'

export const i18n = new VueI18n({
  locale: localStorage.getItem('locale') || 'en-US', // 从localStorage中获取 默认英文
  messages: {
    'zh-CN': {
      ...require('./zh'),
      ...zhLocale
    }, // 中文语言包
    'en-US': {
      ...require('./en'),
      ...enLocale
    } // 英文语言包
  }
})
