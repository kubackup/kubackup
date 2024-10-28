import defaultSettings from '@/settings'
import { i18n } from '@/i18n'

export default function getPageTitle(pageTitle) {
  const title = (i18n.locale === 'zh-CN' ? defaultSettings.title : defaultSettings.title_en) || 'KuBackup'
  if (pageTitle) {
    return `${pageTitle} - ${title}`
  }
  return `${title}`
}
