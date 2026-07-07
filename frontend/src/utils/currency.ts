// 统一的美元金额格式化：≥1 保留两位小数、≥0.01 三位、其余四位。
// 日志列表、图表刻度与供应商卡片共用这一份实现，调整精度只改这里。
export const formatUsd = (value?: number | null): string => {
  if (value === undefined || value === null || Number.isNaN(value)) {
    return '$0.0000'
  }
  if (value >= 1) {
    return `$${value.toFixed(2)}`
  }
  if (value >= 0.01) {
    return `$${value.toFixed(3)}`
  }
  return `$${value.toFixed(4)}`
}
