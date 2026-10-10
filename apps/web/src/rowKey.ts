/**
 * 稳定行键工厂（KNOWN_ISSUES #76）：为表格 / 列表的 `v-for` 生成「进程内自增、各组件独立」
 * 的稳定 key。此前这 3 行在 7 个组件（TagsDialog / BucketTags / HeadersDialog /
 * BatchMetadataDialog / LifecycleDialog / BucketCors / BucketPolicyVisualEditor）里逐字复制；
 * 收敛为一个 canonical helper。
 *
 * 为什么不用数组下标或随机值：下标会在删除中间行时让其余行的 key 变化，触发 DOM 重建
 * （输入框失焦、动画重置）；随机值不稳定、同样会重建。自增序号保证「删除某行时其余行
 * key 不变」——各组件「删除中间行保留原 DOM 节点」的行为测试即钉住这一点。
 */
export function createRowKey(prefix = 'row'): () => string {
  let seq = 0
  return () => `${prefix}-${++seq}`
}
