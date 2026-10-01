<script lang="ts">
import type { PreviewKind } from '../preview'

/**
 * 预览状态。**不携带 URL**：`<img>`/`<video>`/`<iframe>` 直连代理 URL 无法携带
 * Authorization 头，启用 S3C_TOKEN 的部署下会 401（review §C1）；取回统一在
 * 本组件内以带 Bearer 的 fetch → blob → objectURL 完成，故 URL 属于组件内部状态。
 */
export interface PreviewState {
  key: string
  kind: PreviewKind
}
</script>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { toErrorMessage } from '../errors'

import { api } from '../api'
import { downloadProxyObject, fetchProxy, fetchProxyBlob } from '../proxy'
import { toast } from '../store'
import { t, tf } from '../i18n'
import { useFocusTrap } from '../composables/useFocusTrap'
import { useKeydownStack } from '../composables/useKeydownStack'

const props = defineProps<{
  preview: PreviewState | null
  accountId: string
  bucket: string
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const text = ref('')
const truncated = ref(false)
const loading = ref(false)
/** 媒体/PDF 的 inline objectURL（取回成功后渲染；切换/卸载时释放）。 */
const mediaUrl = ref('')
/** 媒体/PDF 取回失败原因（非空时渲染失败占位，绝不渲染错误体）。 */
const mediaError = ref('')

let fetchCtrl: AbortController | null = null

function cancelFetch() {
  fetchCtrl?.abort()
  fetchCtrl = null
}

function releaseMediaUrl() {
  if (mediaUrl.value) {
    URL.revokeObjectURL(mediaUrl.value)
    mediaUrl.value = ''
  }
}

/** 是否为需要「带鉴权取回字节后渲染」的媒体/PDF 类型。 */
function isMediaKind(kind: PreviewState['kind']): boolean {
  return kind === 'image' || kind === 'video' || kind === 'audio' || kind === 'pdf'
}

const open = computed(() => !!props.preview)
const card = ref<HTMLElement>()
const { trapTab } = useFocusTrap(card, open)

// Escape 关闭预览（与 ModalDialog 行为一致；经 keydown 栈，仅顶层生效）。
function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && props.preview) {
    e.preventDefault()
    emit('close')
    return
  }
  trapTab(e)
}

useKeydownStack(onKeydown, open)
onBeforeUnmount(() => {
  cancelFetch()
  releaseMediaUrl()
})

/** 附件下载：带 Bearer 经代理取回后落盘；失败只提示，不把错误体当文件保存。 */
async function downloadPreview(key: string) {
  try {
    await downloadProxyObject(
      { accountId: props.accountId, bucket: props.bucket, key, apiBase: api.base, token: api.token },
      key.split('/').pop() || 'object',
    )
  } catch (err) {
    toast(tf('preview.downloadFail', { msg: toErrorMessage(err) }), 'err')
  }
}

watch(() => props.preview, async (p) => {
  cancelFetch()
  releaseMediaUrl()
  mediaError.value = ''
  if (!p) {
    text.value = ''
    truncated.value = false
    loading.value = false
    return
  }
  text.value = ''
  truncated.value = false
  if (p.kind === 'text') {
    const ctrl = new AbortController()
    fetchCtrl = ctrl
    loading.value = true
    try {
      const res = await fetchProxy({
        accountId: props.accountId,
        bucket: props.bucket,
        mode: 'text',
        key: p.key,
        apiBase: api.base,
        token: api.token,
        signal: ctrl.signal,
      })
      if (fetchCtrl !== ctrl) return
      text.value = await res.text()
      if (fetchCtrl !== ctrl) return
      truncated.value = res.headers.get('X-Preview-Truncated') === '1'
    } catch (err) {
      if (ctrl.signal.aborted || fetchCtrl !== ctrl) return
      text.value = tf('preview.fail', { msg: toErrorMessage(err) })
    } finally {
      if (fetchCtrl === ctrl) loading.value = false
    }
  } else if (isMediaKind(p.kind)) {
    const ctrl = new AbortController()
    fetchCtrl = ctrl
    loading.value = true
    try {
      const blob = await fetchProxyBlob({
        accountId: props.accountId,
        bucket: props.bucket,
        mode: 'inline',
        key: p.key,
        apiBase: api.base,
        token: api.token,
        signal: ctrl.signal,
      })
      if (fetchCtrl !== ctrl) return
      mediaUrl.value = URL.createObjectURL(blob)
    } catch (err) {
      if (ctrl.signal.aborted || fetchCtrl !== ctrl) return
      mediaError.value = toErrorMessage(err)
    } finally {
      if (fetchCtrl === ctrl) loading.value = false
    }
  }
})
</script>

<template>
  <Teleport to="body">
    <Transition name="modal-fade">
      <div v-if="preview" class="pv-backdrop" @click.self="emit('close')">
        <div ref="card" class="pv-card" role="dialog" aria-modal="true" :aria-label="tf('preview.aria', { key: preview.key })">
          <div class="pv-head">
            <span class="mono" style="word-break:break-all">{{ preview.key }}</span>
            <div class="pv-actions">
              <button class="btn secondary sm" @click="downloadPreview(preview.key)">{{ t('common.download') }}</button>
              <button class="btn secondary sm" @click="emit('close')">{{ t('common.close') }}</button>
            </div>
          </div>
          <div class="pv-body">
            <!-- 文本/代码：服务端强制纯文本，前端转义渲染 -->
            <div v-if="preview.kind === 'text'" class="pv-text-wrap">
              <pre class="pv-pre" v-if="!loading">{{ text }}</pre>
              <div v-else class="empty" style="padding:30px">{{ t('preview.loadingText') }}</div>
              <div v-if="truncated" class="badge" style="color:var(--warn)">
                {{ t('preview.truncated') }}
              </div>
            </div>
            <!-- 图片/PDF/媒体：带 Bearer 取回字节后以 objectURL 渲染 -->
            <template v-else-if="isMediaKind(preview.kind)">
              <div v-if="mediaError" class="empty">{{ tf('preview.fail', { msg: mediaError }) }}</div>
              <div v-else-if="loading" class="empty" style="padding:30px">{{ t('preview.loadingText') }}</div>
              <!-- 图片（含 SVG）：img 上下文脚本不执行 -->
              <img v-else-if="preview.kind === 'image'" :src="mediaUrl" :alt="preview.key" @error="emit('close')" />
              <!-- 视频/音频：原生播放器（服务端代理支持 Range 拖动） -->
              <video v-else-if="preview.kind === 'video'" :src="mediaUrl" controls class="pv-media" />
              <audio v-else-if="preview.kind === 'audio'" :src="mediaUrl" controls class="pv-audio" />
              <!-- PDF：sandbox iframe，禁脚本/弹窗 -->
              <iframe v-else :src="mediaUrl" sandbox="" class="pv-iframe" :title="t('preview.pdfTitle')" />
            </template>
            <!-- 未知类型 -->
            <div v-else class="empty">
              <span class="empty-icon" aria-hidden="true">📄</span>
              {{ t('preview.unsupported') }}
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
/* 图片预览 */
.pv-backdrop {
  position: fixed; inset: 0; z-index: 190;
  background: rgba(5, 10, 8, .7);
  backdrop-filter: blur(4px);
  display: flex; align-items: center; justify-content: center;
  padding: 24px;
}
.pv-card {
  display: flex; flex-direction: column;
  max-width: min(1100px, 100%);
  max-height: 100%;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-lg);
  overflow: hidden;
}
.pv-head {
  display: flex; align-items: center; justify-content: space-between; gap: 12px;
  padding: 10px 14px;
  border-bottom: 1px solid var(--border);
}
.pv-actions { display: flex; gap: 8px; flex: none; }
.pv-body { padding: 12px; overflow: auto; display: flex; align-items: center; justify-content: center; }
.pv-body img { max-width: 100%; max-height: 72vh; border-radius: var(--radius); }
.pv-media { max-width: 100%; max-height: 72vh; border-radius: var(--radius); }
.pv-audio { width: min(560px, 100%); }
.pv-iframe {
  width: min(1100px, 80vw); height: 72vh;
  border: 1px solid var(--border); border-radius: var(--radius);
  background: var(--panel-2);
}
.pv-text-wrap { width: min(900px, 100%); }
.pv-pre {
  margin: 0; padding: 14px;
  max-height: 68vh; overflow: auto;
  background: var(--panel-2);
  border: 1px solid var(--border); border-radius: var(--radius);
  font-family: var(--font-mono); font-size: 12px; line-height: 1.7;
  white-space: pre-wrap; word-break: break-all;
  color: var(--text);
}
</style>
