<template>
  <div class="pipeline-step-nav" role="navigation" :aria-label="$t('pipelineNav.aria')">
    <button
      type="button"
      class="pipeline-step-nav__btn"
      :disabled="!canPrev || busy"
      :title="!canPrev ? $t('pipelineNav.prevDisabled') : undefined"
      @click="goPrev"
    >
      {{ $t('pipelineNav.prev') }}
    </button>
    <span class="pipeline-step-nav__pos">{{ $t('pipelineNav.position', { current: currentStep, total: 5 }) }}</span>
    <button
      type="button"
      class="pipeline-step-nav__btn pipeline-step-nav__btn--next"
      :disabled="!canNext || busy"
      :title="!canNext ? $t('pipelineNav.nextDisabled') : undefined"
      @click="goNext"
    >
      <span v-if="busy" class="pipeline-step-nav__spinner" aria-hidden="true" />
      <span v-else>{{ $t('pipelineNav.next') }}</span>
    </button>
    <p v-if="message" class="pipeline-step-nav__msg" role="status">{{ message }}</p>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { getReportBySimulation } from '../api/report'

const props = defineProps({
  /** 1 = graph, 2 = env, 3 = run sim, 4 = report, 5 = interaction */
  currentStep: { type: Number, required: true },
  projectId: { type: String, default: null },
  simulationId: { type: String, default: null },
  reportId: { type: String, default: null },
})

const router = useRouter()
const { t } = useI18n()
const busy = ref(false)
const message = ref('')

const canPrev = computed(() => {
  if (props.currentStep <= 1) return false
  if (props.currentStep === 2) return !!props.projectId
  if (props.currentStep === 3) return !!props.simulationId
  if (props.currentStep === 4) return !!props.simulationId
  if (props.currentStep === 5) return !!props.reportId
  return true
})

const canNext = computed(() => {
  if (props.currentStep >= 5) return false
  if (props.currentStep === 1) return !!props.projectId
  if (props.currentStep === 2) return !!props.simulationId
  if (props.currentStep === 3) return !!props.simulationId
  if (props.currentStep === 4) return !!props.reportId
  return true
})

const goPrev = async () => {
  message.value = ''
  if (!canPrev.value) return
  const step = props.currentStep
  try {
    if (step === 2 && props.projectId) {
      await router.push({ name: 'Process', params: { projectId: props.projectId }, query: { step: '1' } })
      return
    }
    if (step === 3 && props.simulationId) {
      await router.push({ name: 'Simulation', params: { simulationId: props.simulationId } })
      return
    }
    if (step === 4 && props.simulationId) {
      await router.push({ name: 'SimulationRun', params: { simulationId: props.simulationId } })
      return
    }
    if (step === 5 && props.reportId) {
      await router.push({ name: 'Report', params: { reportId: props.reportId } })
    }
  } catch (e) {
    message.value = e?.message || t('pipelineNav.navFailed')
  }
}

const goNext = async () => {
  message.value = ''
  if (!canNext.value) return
  const step = props.currentStep
  busy.value = true
  try {
    if (step === 1 && props.projectId) {
      await router.push({ name: 'Process', params: { projectId: props.projectId }, query: { step: '2' } })
      return
    }
    if (step === 2 && props.simulationId) {
      await router.push({ name: 'SimulationRun', params: { simulationId: props.simulationId } })
      return
    }
    if (step === 3 && props.simulationId) {
      if (props.reportId) {
        await router.push({ name: 'Report', params: { reportId: props.reportId } })
        return
      }
      const res = await getReportBySimulation(props.simulationId)
      if (res?.success && res?.data) {
        const rid = res.data.report_id
        if (typeof rid === 'string' && rid) {
          await router.push({ name: 'Report', params: { reportId: rid } })
          return
        }
      }
      message.value = t('pipelineNav.noReportYet')
      return
    }
    if (step === 4 && props.reportId) {
      await router.push({ name: 'Interaction', params: { reportId: props.reportId } })
    }
  } catch (e) {
    message.value = e?.message || t('pipelineNav.navFailed')
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.pipeline-step-nav {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: center;
  gap: 10px 16px;
  padding: 8px clamp(8px, 2vw, 16px);
  border-bottom: 1px solid var(--doc-border, #e5e7eb);
  background: var(--doc-workbench-mid, #f5f5f5);
  color: var(--doc-text, #111827);
  font-size: 12px;
  min-width: 0;
  max-width: 100%;
  box-sizing: border-box;
}
.pipeline-step-nav__pos {
  font-weight: 600;
  color: var(--doc-muted, #6b7280);
  font-family: var(--doc-font-mono, ui-monospace, monospace);
  order: 0;
}
.pipeline-step-nav__btn {
  padding: 6px 14px;
  font-size: 12px;
  font-weight: 600;
  border-radius: 6px;
  border: 1px solid var(--doc-border, #e5e7eb);
  background: var(--doc-surface, #fff);
  color: var(--doc-text, #111827);
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
  min-width: 0;
}
.pipeline-step-nav__btn:hover:not(:disabled) {
  border-color: var(--doc-accent, #00add8);
  background: var(--doc-upload-surface, #f9fafb);
}
.pipeline-step-nav__btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.pipeline-step-nav__btn--next {
  background: var(--doc-cta-primary-bg, #111827);
  color: var(--doc-cta-primary-fg, #fff);
  border-color: var(--doc-cta-primary-bg, #111827);
}
.pipeline-step-nav__btn--next:hover:not(:disabled) {
  filter: brightness(0.95);
}
.pipeline-step-nav__spinner {
  display: inline-block;
  width: 14px;
  height: 14px;
  border: 2px solid color-mix(in srgb, var(--doc-cta-primary-fg, #fff) 35%, transparent);
  border-top-color: var(--doc-cta-primary-fg, #fff);
  border-radius: 50%;
  animation: pnav-spin 0.7s linear infinite;
  vertical-align: middle;
}
@keyframes pnav-spin {
  to {
    transform: rotate(360deg);
  }
}
.pipeline-step-nav__msg {
  width: 100%;
  text-align: center;
  font-size: 11px;
  color: #b45309;
  margin: 0;
  padding: 0 4px 2px;
}
</style>
