/**
 * 存储库类型
 * @type {[{code: number, name: string}, {code: number, name: string}, null]}
 */
export const repoTypeList = [
  { code: 1, name: 'S3', tips: 'Minio' },
  { code: 3, name: 'Sftp', tips: '不推荐使用' },
  { code: 4, name: 'Local', tips: '测试推荐' },
  { code: 5, name: 'Rest', tips: '推荐' },
  { code: 6, name: 'HwObs', tips: '华为云对象存储' },
  { code: 2, name: 'AliOos', tips: '阿里云对象存储' },
  { code: 7, name: 'TxCos', tips: '腾讯云对象存储' }
]

/**
 * 存储库类型（英文）
 * @type {[{code: number, name: string}, {code: number, name: string}, null]}
 */
export const repoTypeListEN = [
  { code: 1, name: 'S3', tips: 'Minio' },
  { code: 3, name: 'Sftp', tips: 'Not recommended' },
  { code: 4, name: 'Local', tips: 'Recommended for testing' },
  { code: 5, name: 'Rest', tips: 'Recommended' },
  { code: 6, name: 'HwObs', tips: 'Huawei Cloud Object Storage' },
  { code: 2, name: 'AliOos', tips: 'Alibaba Cloud Object Storage' },
  { code: 7, name: 'TxCos', tips: 'Tencent Cloud Object Storage' }
]

/**
 * 清除策略类型
 * @type {[{code: number, name: string, tips: string},{code: string, name: string, tips: string},{code: number, name: string, tips: string},{code: number, name: string, tips: string},{code: number, name: string, tips: string},null]}
 */
export const ForgetTypeList = [
  { code: 'last', name: '份', tips: '份' },
  { code: 'hourly', name: '小时', tips: '小时' },
  { code: 'daily', name: '天', tips: '天' },
  { code: 'weekly', name: '周', tips: '周' },
  { code: 'monthly', name: '月', tips: '月' },
  { code: 'yearly', name: '年', tips: '年' }
]

export const ForgetTypeListEN = [
  { code: 'last', name: 'last', tips: '份' },
  { code: 'hourly', name: 'hourly', tips: '小时' },
  { code: 'daily', name: 'daily', tips: '天' },
  { code: 'weekly', name: 'weekly', tips: '周' },
  { code: 'monthly', name: 'monthly', tips: '月' },
  { code: 'yearly', name: 'yearly', tips: '年' }
]

/**
 * 存储库连接状态
 * @type {[{code: number, color: string, name: string}, {code: number, color: string, name: string}, {code: number, color: string, name: string}]}
 */
export const repoStatusList = [
  { code: 1, name: '获取中', color: 'info' },
  { code: 2, name: '正常', color: 'success' },
  { code: 3, name: '错误', color: 'danger' }
]

export const repoStatusListEN = [
  { code: 1, name: 'Loading', color: 'info' },
  { code: 2, name: 'Normal', color: 'success' },
  { code: 3, name: 'Error', color: 'danger' }
]

/**
 * 日志级别
 * @type {[{code: number, color: string, name: string}, {code: number, color: string, name: string}]}
 */
export const LoglevelList = [
  { code: 1, name: 'Info', color: 'info' },
  { code: 2, name: 'Warning', color: 'warning' },
  { code: 3, name: 'Success', color: 'success' },
  { code: 4, name: 'Error', color: 'error' }
]

/**
 * 压缩级别
 * @type {[{code: number, color: string, name: string},{code: number, color: string, name: string},{code: number, color: string, name: string}]}
 */
export const compressionList = [
  { code: 0, name: '自动', color: 'success' },
  { code: 1, name: '关闭', color: 'info' },
  { code: 2, name: '最大', color: 'warning' }
]

export const compressionListEN = [
  { code: 0, name: 'Auto', color: 'success' },
  { code: 1, name: 'Close', color: 'info' },
  { code: 2, name: 'Max', color: 'warning' }
]
