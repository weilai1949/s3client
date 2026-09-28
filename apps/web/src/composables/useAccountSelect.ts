import { state, rememberedAccountId } from '../store'

/**
 * 账号回退选择（`BucketsPanel` / `RecycleBinPanel` / `App.loadAccounts` 共用）。
 *
 * 此前三处逐字复制同一段「初始化选中账号」逻辑（review Nit：账号回退初始化逻辑逐字复制）。
 * 统一优先级：**remembered（须仍在账号列表中） > 仍有效的 currentAccountId > 第一个账号**，
 * 无账号可选时返回 `''`。
 *
 * 纯函数：只读 store，不落地选择——调用方各自保留自己的副作用
 * （面板写 `accSel.value`，App 调 `selectAccount`），便于独立测试。
 */
export function resolveAccountSelect(): string {
  const remembered = rememberedAccountId()
  if (state.accounts.some((a) => a.id === remembered)) return remembered
  if (state.currentAccountId && state.accounts.some((a) => a.id === state.currentAccountId)) {
    return state.currentAccountId
  }
  return state.accounts[0]?.id ?? ''
}
