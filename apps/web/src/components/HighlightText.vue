<script setup lang="ts">
import { computed } from 'vue'
import { highlightSegments } from '../highlight'

/**
 * 高亮命中片段的对象名。用 `<template v-for>` 逐段渲染而不是 `v-html`——
 * 对象名来自外部（S3 key），任何拼接 HTML 的写法都是注入面（仓库硬约束：禁 v-html）。
 */
const props = defineProps<{
  text: string
  /** 当前过滤词；空串时不产生 `<mark>`。 */
  query: string
}>()

const segments = computed(() => highlightSegments(props.text, props.query))
</script>

<template>
  <span class="hl-text"><template v-for="(seg, i) in segments" :key="i"><mark v-if="seg.hit" class="hl">{{ seg.text }}</mark><template v-else>{{ seg.text }}</template></template></span>
</template>

<style scoped>
.hl {
  background: var(--primary-soft);
  color: var(--text);
  border-radius: 3px;
  padding: 0 1px;
}
</style>
