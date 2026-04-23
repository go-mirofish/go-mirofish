<template>
  <div
    class="history-database"
    :class="{ 'no-projects': projects.length === 0 && !loading }"
    ref="historyContainer"
  >
    <div class="history-eyebrow">
      <div class="history-eyebrow-line" aria-hidden="true"></div>
      <p class="history-eyebrow-title">{{ $t('history.sectionTitle') }}</p>
      <div class="history-eyebrow-line" aria-hidden="true"></div>
    </div>

    <div v-if="projects.length > 0 || loading" class="history-stage">
      <div class="history-vignette" aria-hidden="true"></div>
      <div class="history-subtle-grid" aria-hidden="true"></div>

      <div class="history-panel-head">
        <span class="history-panel-line" aria-hidden="true"></span>
        <h2 class="history-panel-title">{{ $t('history.panelTitle') }}</h2>
        <div class="history-actions">
          <button class="history-action-btn" type="button" @click="toggleEditMode">
            {{ isEditMode ? $t('history.done') : $t('history.edit') }}
          </button>
        </div>
        <span class="history-panel-line" aria-hidden="true"></span>
      </div>

      <div v-if="projects.length > 0 && isEditMode" class="history-bulkbar">
        <label class="bulk-select">
          <input type="checkbox" :checked="allSelected" @change="toggleSelectAll" />
          <span>{{ $t('history.selectAll') }}</span>
        </label>
        <span class="bulk-count mono">{{ $t('history.selectedCount', { count: selectedIds.size }) }}</span>
        <button
          class="history-action-btn history-action-btn--danger"
          type="button"
          :disabled="selectedIds.size === 0 || isDeleting"
          @click="deleteSelected"
        >
          {{ isDeleting ? $t('history.deleting') : $t('history.deleteSelected') }}
        </button>
      </div>

    <div v-if="projects.length > 0" class="cards-container" :class="{ expanded: isExpanded }" :style="containerStyle" ref="cardsContainer">
      <div 
        v-for="(project, index) in projects" 
        :key="project.simulation_id"
        class="project-card"
        :class="{ expanded: isExpanded, hovering: hoveringCard === index }"
        :style="getCardStyle(index)"
        @mouseenter="hoveringCard = index"
        @mouseleave="hoveringCard = null"
        @click="handleCardClick(project)"
      >
        <div class="card-header">
          <span class="card-id">{{ formatSimulationId(project.simulation_id) }}</span>
          <label v-if="isEditMode" class="card-select" @click.stop>
            <input
              type="checkbox"
              :checked="selectedIds.has(project.simulation_id)"
              @change="toggleSelected(project.simulation_id)"
            />
          </label>
          <div class="card-status-icons">
            <span 
              class="status-icon" 
              :class="{ available: project.project_id, unavailable: !project.project_id }"
              :title="$t('history.graphBuild')"
            >◇</span>
            <span 
              class="status-icon available" 
              :title="$t('history.envSetup')"
            >◈</span>
            <span 
              class="status-icon" 
              :class="{ available: project.report_id, unavailable: !project.report_id }"
              :title="$t('history.analysisReport')"
            >◆</span>
          </div>
        </div>

        <div class="card-files-wrapper">
          <div class="files-list" v-if="project.files && project.files.length > 0">
            <div 
              v-for="(file, fileIndex) in project.files.slice(0, 3)" 
              :key="fileIndex"
              class="file-item"
            >
              <span class="file-tag" :class="getFileType(file.filename)">{{ getFileTypeLabel(file.filename) }}</span>
              <span class="file-name">{{ truncateFilename(file.filename, 20) }}</span>
            </div>
            <div v-if="project.files.length > 3" class="files-more">
              {{ $t('history.moreFiles', { count: project.files.length - 3 }) }}
            </div>
          </div>
          <div class="files-empty" v-else>
            <span class="empty-file-icon">◇</span>
            <span class="empty-file-text">{{ $t('history.noFiles') }}</span>
          </div>
        </div>

        <h3 class="card-title">{{ getSimulationTitle(project.simulation_requirement) }}</h3>

        <p class="card-desc">{{ truncateText(project.simulation_requirement, 55) }}</p>

        <div class="card-footer">
          <span class="card-datetime">{{ formatDateTime(project.created_at) }}</span>
          <span class="card-progress" :class="getProgressClass(project)">
            <span class="status-dot">●</span> {{ formatRounds(project) }}
          </span>
        </div>

        <div class="card-bottom-line"></div>
      </div>
    </div>

    <div v-if="loading" class="loading-state">
      <span class="loading-spinner"></span>
      <span class="loading-text">{{ $t('history.loadingText') }}</span>
    </div>
    </div>

    <Teleport to="body">
      <Transition name="modal">
        <div v-if="selectedProject" class="modal-overlay" @click.self="closeModal">
          <div class="modal-content">
            <div class="modal-header">
              <div class="modal-title-section">
                <span class="modal-id">{{ formatSimulationId(selectedProject.simulation_id) }}</span>
                <span class="modal-progress" :class="getProgressClass(selectedProject)">
                  <span class="status-dot">●</span> {{ formatRounds(selectedProject) }}
                </span>
                <span class="modal-create-time">{{ formatDate(selectedProject.created_at) }} {{ formatTime(selectedProject.created_at) }}</span>
              </div>
            <div class="modal-header-actions">
              <button
                class="modal-delete"
                type="button"
                :disabled="isDeleting"
                @click="deleteSingle(selectedProject.simulation_id)"
              >
                {{ isDeleting ? $t('history.deleting') : $t('history.delete') }}
              </button>
              <button class="modal-close" @click="closeModal">×</button>
            </div>
            </div>

            <div class="modal-body">
              <div class="modal-section">
                <div class="modal-label">{{ $t('history.simRequirement') }}</div>
                <div class="modal-requirement">{{ selectedProject.simulation_requirement || $t('common.none') }}</div>
              </div>

              <div class="modal-section">
                <div class="modal-label">{{ $t('history.relatedFiles') }}</div>
                <div class="modal-files" v-if="selectedProject.files && selectedProject.files.length > 0">
                  <div v-for="(file, index) in selectedProject.files" :key="index" class="modal-file-item">
                    <span class="file-tag" :class="getFileType(file.filename)">{{ getFileTypeLabel(file.filename) }}</span>
                    <span class="modal-file-name">{{ file.filename }}</span>
                  </div>
                </div>
                <div class="modal-empty" v-else>{{ $t('history.noRelatedFiles') }}</div>
              </div>
            </div>

            <div class="modal-divider">
              <span class="divider-line"></span>
              <span class="divider-text">{{ $t('history.replayTitle') }}</span>
              <span class="divider-line"></span>
            </div>

            <div class="modal-actions">
              <button 
                class="modal-btn btn-project" 
                @click="goToProject"
                :disabled="!selectedProject.project_id"
              >
                <span class="btn-step">Step1</span>
                <span class="btn-icon">◇</span>
                <span class="btn-text">{{ $t('history.step1Button') }}</span>
              </button>
              <button 
                class="modal-btn btn-simulation" 
                @click="goToSimulation"
              >
                <span class="btn-step">Step2</span>
                <span class="btn-icon">◈</span>
                <span class="btn-text">{{ $t('history.step2Button') }}</span>
              </button>
              <button 
                class="modal-btn btn-report" 
                @click="goToReport"
                :disabled="!selectedProject.report_id"
              >
                <span class="btn-step">Step4</span>
                <span class="btn-icon">◆</span>
                <span class="btn-text">{{ $t('history.step4Button') }}</span>
              </button>
            </div>
            <div class="modal-playback-hint">
              <span class="hint-text">{{ $t('history.replayHint') }}</span>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, onActivated, watch, nextTick } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { getSimulationHistory, deleteSimulation } from '../api/simulation'

const router = useRouter()
const route = useRoute()
const { t } = useI18n()

const projects = ref([])
const loading = ref(true)
const isExpanded = ref(false)
const isEditMode = ref(false)
const isDeleting = ref(false)
const hoveringCard = ref(null)
const historyContainer = ref(null)
const cardsContainer = ref(null)
const selectedProject = ref(null)
const selectedIds = ref(new Set())
let observer = null
let isAnimating = false
let expandDebounceTimer = null
let pendingState = null

const viewportWidth = ref(typeof window !== 'undefined' ? window.innerWidth : 1440)

const CARD_WIDTH = computed(() => {
  if (viewportWidth.value <= 768) return 200
  if (viewportWidth.value <= 1200) return 240
  return 280
})
const CARD_HEIGHT = 280 
const CARD_GAP = 24

const cardsPerRow = computed(() => {
  const total = projects.value.length
  if (total <= 1) return 1

  const containerWidth =
    cardsContainer.value?.clientWidth ||
    historyContainer.value?.clientWidth ||
    viewportWidth.value

  const perRow = Math.max(1, Math.floor((containerWidth + CARD_GAP) / (CARD_WIDTH.value + CARD_GAP)))
  return Math.min(perRow, total)
})

const containerStyle = computed(() => {
  if (!isExpanded.value) {
    return { minHeight: '420px' }
  }
  
  const total = projects.value.length
  if (total === 0) {
    return { minHeight: '280px' }
  }
  
  const rows = Math.ceil(total / cardsPerRow.value)
  const expandedHeight = rows * CARD_HEIGHT + (rows - 1) * CARD_GAP + 10
  
  return { minHeight: `${expandedHeight}px` }
})

const getCardStyle = (index) => {
  const total = projects.value.length
  
  if (isExpanded.value) {
    const transition = 'transform 700ms cubic-bezier(0.23, 1, 0.32, 1), opacity 700ms cubic-bezier(0.23, 1, 0.32, 1), box-shadow 0.3s ease, border-color 0.3s ease'

    const perRow = cardsPerRow.value
    const col = index % perRow
    const row = Math.floor(index / perRow)
    
    const currentRowStart = row * perRow
    const currentRowCards = Math.min(perRow, total - currentRowStart)
    
    const rowWidth = currentRowCards * CARD_WIDTH.value + (currentRowCards - 1) * CARD_GAP
    
    const startX = -(rowWidth / 2) + (CARD_WIDTH.value / 2)
    const colInRow = index % perRow
    const x = startX + colInRow * (CARD_WIDTH.value + CARD_GAP)
    
    const y = 20 + row * (CARD_HEIGHT + CARD_GAP)

    return {
      transform: `translate(${x}px, ${y}px) rotate(0deg) scale(1)`,
      zIndex: 100 + index,
      opacity: 1,
      transition: transition
    }
  } else {
    const transition = 'transform 700ms cubic-bezier(0.23, 1, 0.32, 1), opacity 700ms cubic-bezier(0.23, 1, 0.32, 1), box-shadow 0.3s ease, border-color 0.3s ease'

    const centerIndex = (total - 1) / 2
    const offset = index - centerIndex
    
    const x = offset * 35
    const y = 25 + Math.abs(offset) * 8
    const r = offset * 3
    const s = 0.95 - Math.abs(offset) * 0.05
    
    return {
      transform: `translate(${x}px, ${y}px) rotate(${r}deg) scale(${s})`,
      zIndex: 10 + index,
      opacity: 1,
      transition: transition
    }
  }
}

const handleResize = () => {
  viewportWidth.value = window.innerWidth
}

const getProgressClass = (simulation) => {
  const current = simulation.current_round || 0
  const total = simulation.total_rounds || 0
  
  if (total === 0 || current === 0) {
    return 'not-started'
  } else if (current >= total) {
    return 'completed'
  } else {
    return 'in-progress'
  }
}

const formatDate = (dateStr) => {
  if (!dateStr) return ''
  try {
    const date = new Date(dateStr)
    return date.toISOString().slice(0, 10)
  } catch {
    return dateStr?.slice(0, 10) || ''
  }
}

const formatTime = (dateStr) => {
  if (!dateStr) return ''
  try {
    const date = new Date(dateStr)
    const hours = date.getHours().toString().padStart(2, '0')
    const minutes = date.getMinutes().toString().padStart(2, '0')
    return `${hours}:${minutes}`
  } catch {
    return ''
  }
}

const formatDateTime = (dateStr) => {
  if (!dateStr) return ''
  try {
    const d = new Date(dateStr)
    const y = d.getFullYear()
    const m = (d.getMonth() + 1).toString().padStart(2, '0')
    const day = d.getDate().toString().padStart(2, '0')
    const hours = d.getHours().toString().padStart(2, '0')
    const minutes = d.getMinutes().toString().padStart(2, '0')
    return `${y}-${m}-${day} ${hours}:${minutes}`
  } catch {
    return ''
  }
}

const truncateText = (text, maxLength) => {
  if (!text) return ''
  return text.length > maxLength ? text.slice(0, maxLength) + '...' : text
}

const getSimulationTitle = (requirement) => {
  if (!requirement) return t('history.untitledSimulation')
  const title = requirement.slice(0, 20)
  return requirement.length > 20 ? title + '...' : title
}

const formatSimulationId = (simulationId) => {
  if (!simulationId) return 'SIM_UNKNOWN'
  const prefix = simulationId.replace('sim_', '').slice(0, 6)
  return `SIM_${prefix.toUpperCase()}`
}

const formatRounds = (simulation) => {
  const current = simulation.current_round || 0
  const total = simulation.total_rounds || 0
  if (total === 0) return t('history.notStarted')
  return t('history.roundsProgress', { current, total })
}

const getFileType = (filename) => {
  if (!filename) return 'other'
  const ext = filename.split('.').pop()?.toLowerCase()
  const typeMap = {
    'pdf': 'pdf',
    'doc': 'doc', 'docx': 'doc',
    'xls': 'xls', 'xlsx': 'xls', 'csv': 'xls',
    'ppt': 'ppt', 'pptx': 'ppt',
    'txt': 'txt', 'md': 'txt', 'json': 'code',
    'jpg': 'img', 'jpeg': 'img', 'png': 'img', 'gif': 'img',
    'zip': 'zip', 'rar': 'zip', '7z': 'zip'
  }
  return typeMap[ext] || 'other'
}

const getFileTypeLabel = (filename) => {
  if (!filename) return 'FILE'
  const ext = filename.split('.').pop()?.toUpperCase()
  return ext || 'FILE'
}

const truncateFilename = (filename, maxLength) => {
  if (!filename) return t('history.unknownFile')
  if (filename.length <= maxLength) return filename
  
  const ext = filename.includes('.') ? '.' + filename.split('.').pop() : ''
  const nameWithoutExt = filename.slice(0, filename.length - ext.length)
  const truncatedName = nameWithoutExt.slice(0, maxLength - ext.length - 3) + '...'
  return truncatedName + ext
}

const navigateToProject = (simulation) => {
  selectedProject.value = simulation
}

const handleCardClick = (simulation) => {
  if (isEditMode.value) {
    toggleSelected(simulation.simulation_id)
    return
  }
  navigateToProject(simulation)
}

const toggleEditMode = () => {
  isEditMode.value = !isEditMode.value
  if (!isEditMode.value) {
    selectedIds.value = new Set()
  }
}

const toggleSelected = (simulationId) => {
  const next = new Set(selectedIds.value)
  if (next.has(simulationId)) next.delete(simulationId)
  else next.add(simulationId)
  selectedIds.value = next
}

const allSelected = computed(() => {
  return projects.value.length > 0 && selectedIds.value.size === projects.value.length
})

const toggleSelectAll = () => {
  if (allSelected.value) {
    selectedIds.value = new Set()
    return
  }
  selectedIds.value = new Set(projects.value.map(p => p.simulation_id))
}

const removeFromList = (ids) => {
  const idSet = new Set(ids)
  projects.value = projects.value.filter(p => !idSet.has(p.simulation_id))
  selectedIds.value = new Set([...selectedIds.value].filter(id => !idSet.has(id)))
  if (selectedProject.value && idSet.has(selectedProject.value.simulation_id)) {
    selectedProject.value = null
  }
}

const deleteSingle = async (simulationId) => {
  if (!simulationId) return
  if (!window.confirm(t('history.deleteConfirmSingle'))) return

  isDeleting.value = true
  try {
    await deleteSimulation({ simulation_id: simulationId })
    removeFromList([simulationId])
  } catch (err) {
    console.error('Delete failed:', err)
    window.alert(t('history.deleteFailed', { error: err.message || t('common.unknownError') }))
  } finally {
    isDeleting.value = false
  }
}

const deleteSelected = async () => {
  const ids = Array.from(selectedIds.value)
  if (ids.length === 0) return
  if (!window.confirm(t('history.deleteConfirmBulk', { count: ids.length }))) return

  isDeleting.value = true
  try {
    const results = await Promise.allSettled(ids.map(id => deleteSimulation({ simulation_id: id })))
    const ok = []
    const failed = []
    results.forEach((r, idx) => {
      if (r.status === 'fulfilled') ok.push(ids[idx])
      else failed.push(ids[idx])
    })

    if (ok.length) removeFromList(ok)
    if (failed.length) {
      window.alert(t('history.deletePartialFailed', { count: failed.length }))
    }
  } finally {
    isDeleting.value = false
  }
}

const closeModal = () => {
  selectedProject.value = null
}

const goToProject = () => {
  if (selectedProject.value?.project_id) {
    router.push({
      name: 'Process',
      params: { projectId: selectedProject.value.project_id }
    })
    closeModal()
  }
}

const goToSimulation = () => {
  if (selectedProject.value?.simulation_id) {
    router.push({
      name: 'Simulation',
      params: { simulationId: selectedProject.value.simulation_id }
    })
    closeModal()
  }
}

const goToReport = () => {
  if (selectedProject.value?.report_id) {
    router.push({
      name: 'Report',
      params: { reportId: selectedProject.value.report_id }
    })
    closeModal()
  }
}

const loadHistory = async () => {
  try {
    loading.value = true
    const response = await getSimulationHistory(20)
    if (response.success) {
      projects.value = response.data || []
    }
  } catch (error) {
    console.error('Failed to load project history:', error)
    projects.value = []
  } finally {
    loading.value = false
  }
}

const initObserver = () => {
  if (observer) {
    observer.disconnect()
  }
  
  observer = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        const shouldExpand = entry.isIntersecting
        
        pendingState = shouldExpand
        
        if (expandDebounceTimer) {
          clearTimeout(expandDebounceTimer)
          expandDebounceTimer = null
        }
        
        if (isAnimating) return
        
        if (shouldExpand === isExpanded.value) {
          pendingState = null
          return
        }
        
        const delay = shouldExpand ? 50 : 200
        
        expandDebounceTimer = setTimeout(() => {
          if (isAnimating) return
          
          if (pendingState === null || pendingState === isExpanded.value) return
          
          isAnimating = true
          isExpanded.value = pendingState
          pendingState = null
          
          setTimeout(() => {
            isAnimating = false
            
            if (pendingState !== null && pendingState !== isExpanded.value) {
              expandDebounceTimer = setTimeout(() => {
                if (pendingState !== null && pendingState !== isExpanded.value) {
                  isAnimating = true
                  isExpanded.value = pendingState
                  pendingState = null
                  setTimeout(() => {
                    isAnimating = false
                  }, 750)
                }
              }, 100)
            }
          }, 750)
        }, delay)
      })
    },
    {
      threshold: [0.4, 0.6, 0.8],
      rootMargin: '0px 0px -150px 0px'
    }
  )
  
  if (historyContainer.value) {
    observer.observe(historyContainer.value)
  }
}

watch(() => route.path, (newPath) => {
  if (newPath === '/') {
    loadHistory()
  }
})

onMounted(async () => {
  await nextTick()
  await loadHistory()
  
  setTimeout(() => {
    initObserver()
  }, 100)

  if (typeof window !== 'undefined') {
    window.addEventListener('resize', handleResize, { passive: true })
  }
})

onActivated(() => {
  loadHistory()
})

onUnmounted(() => {
  if (observer) {
    observer.disconnect()
    observer = null
  }
  if (expandDebounceTimer) {
    clearTimeout(expandDebounceTimer)
    expandDebounceTimer = null
  }

  if (typeof window !== 'undefined') {
    window.removeEventListener('resize', handleResize)
  }
})
</script>

<style scoped>
.history-database {
  position: relative;
  width: 100%;
  min-height: 160px;
  margin-top: 0;
  padding: 0 0 1.5rem;
  overflow: visible;
}

.history-database.no-projects {
  min-height: auto;
  padding: 0 0 0.5rem;
}

/* Outer label: “Recent simulations” with rules above and below */
.history-eyebrow {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.75rem;
  margin: 0 0 1.5rem;
  padding: 0 1.5rem;
  text-align: center;
  position: relative;
  z-index: 2;
}

.history-eyebrow-line {
  width: 100%;
  max-width: min(720px, 100%);
  height: 1px;
  background: linear-gradient(
    90deg,
    transparent,
    color-mix(in srgb, var(--doc-border) 80%, var(--doc-text) 4%),
    transparent
  );
}

.history-eyebrow-title {
  margin: 0;
  font-family: var(--doc-font-sans, 'Inter', system-ui, sans-serif);
  font-size: 0.7rem;
  font-weight: 600;
  letter-spacing: 0.28em;
  text-transform: uppercase;
  color: var(--doc-muted);
}

/* Dark “stage” with edge glow + subtle grid */
.history-stage {
  position: relative;
  z-index: 0;
  margin: 0 1.5rem;
  padding: 1.75rem 1.5rem 2rem;
  background: var(--doc-bg);
  border: 1px solid var(--doc-border);
  border-radius: var(--doc-radius, 12px);
  overflow: hidden;
  min-height: 200px;
}

@media (min-width: 1024px) {
  .history-stage {
    margin: 0 2.5rem;
    padding: 1.75rem 2.25rem 2.5rem;
  }
}

.history-vignette {
  position: absolute;
  inset: 0;
  pointer-events: none;
  z-index: 0;
  background:
    linear-gradient(
      to bottom,
      color-mix(in srgb, var(--doc-text) 12%, transparent) 0%,
      transparent 40%
    ),
    linear-gradient(
      to right,
      color-mix(in srgb, var(--doc-text) 8%, transparent) 0%,
      transparent 35%,
      transparent 65%,
      color-mix(in srgb, var(--doc-text) 8%, transparent) 100%
    );
}

html[data-theme='light'] .history-vignette {
  background:
    linear-gradient(
      to bottom,
      color-mix(in srgb, #ffffff 55%, transparent) 0%,
      transparent 45%
    ),
    linear-gradient(
      to right,
      color-mix(in srgb, #ffffff 40%, transparent) 0%,
      transparent 38%,
      transparent 62%,
      color-mix(in srgb, #ffffff 40%, transparent) 100%
    );
}

.history-subtle-grid {
  position: absolute;
  inset: 0;
  z-index: 0;
  pointer-events: none;
  opacity: 0.2;
  background-image: linear-gradient(
      to right,
      color-mix(in srgb, var(--doc-border) 60%, transparent) 1px,
      transparent 1px
    ),
    linear-gradient(
      to bottom,
      color-mix(in srgb, var(--doc-border) 60%, transparent) 1px,
      transparent 1px
    );
  background-size: 48px 48px;
  -webkit-mask-image: radial-gradient(
    ellipse 75% 65% at 50% 40%,
    #000 0%,
    transparent 72%
  );
  mask-image: radial-gradient(ellipse 75% 65% at 50% 40%, #000 0%, transparent 72%);
}

.history-panel-head {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 1.25rem;
  margin: 0 0 1.5rem;
  padding: 0 0.5rem;
}

.history-actions {
  display: flex;
  align-items: center;
  margin-left: 0.25rem;
}

.history-action-btn {
  border: 1px solid var(--doc-border);
  background: transparent;
  color: var(--doc-muted);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  padding: 6px 10px;
  cursor: pointer;
  transition: background 0.2s ease, border-color 0.2s ease, color 0.2s ease;
}

.history-action-btn:hover:not(:disabled) {
  background: color-mix(in srgb, var(--doc-text) 6%, transparent);
  border-color: color-mix(in srgb, var(--doc-text) 22%, var(--doc-border));
  color: var(--doc-text);
}

.history-action-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.history-action-btn--danger {
  border-color: color-mix(in srgb, #dc2626 35%, var(--doc-border));
  color: #dc2626;
}

.history-action-btn--danger:hover:not(:disabled) {
  background: color-mix(in srgb, #dc2626 10%, transparent);
  border-color: #dc2626;
}

.history-bulkbar {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 10px;
  margin: -0.5rem 0 1rem;
  border: 1px solid var(--doc-border);
  background: color-mix(in srgb, var(--doc-surface) 70%, transparent);
}

.bulk-select {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--doc-text);
  user-select: none;
}

.bulk-select input {
  accent-color: var(--doc-accent);
}

.bulk-count {
  color: var(--doc-muted);
}

.card-select {
  margin-left: auto;
  display: inline-flex;
  align-items: center;
}

.card-select input {
  width: 14px;
  height: 14px;
  accent-color: var(--doc-accent);
}

.modal-header-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.modal-delete {
  border: 1px solid color-mix(in srgb, #dc2626 35%, var(--doc-border));
  background: transparent;
  color: #dc2626;
  padding: 6px 10px;
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
}

.modal-delete:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.modal-delete:hover:not(:disabled) {
  background: color-mix(in srgb, #dc2626 10%, transparent);
  border-color: #dc2626;
}

.history-panel-line {
  flex: 1;
  max-width: 6rem;
  height: 1px;
  background: linear-gradient(90deg, transparent, var(--doc-border), transparent);
}

@media (min-width: 640px) {
  .history-panel-line {
    max-width: 9rem;
  }
}

.history-panel-title {
  margin: 0;
  flex: 0 0 auto;
  max-width: 20rem;
  text-align: center;
  font-family: var(--doc-font-mono, 'JetBrains Mono', ui-monospace, monospace);
  font-size: 0.7rem;
  font-weight: 500;
  letter-spacing: 0.3em;
  text-transform: uppercase;
  color: var(--doc-muted);
}

.cards-container {
  position: relative;
  z-index: 1;
  display: flex;
  justify-content: center;
  align-items: flex-start;
  padding: 0 0.5rem;
  transition: min-height 700ms cubic-bezier(0.23, 1, 0.32, 1);
}

.project-card {
  position: absolute;
  width: 280px;
  background: var(--doc-surface);
  border: 1px solid color-mix(in srgb, var(--doc-text) 16%, var(--doc-border));
  border-radius: 0;
  padding: 0.9rem 1rem 1.1rem;
  cursor: pointer;
  box-shadow: var(--doc-shadow-soft, 0 1px 2px rgba(0, 0, 0, 0.04));
  transition: box-shadow 0.3s ease, border-color 0.3s ease, transform 700ms cubic-bezier(0.23, 1, 0.32, 1), opacity 700ms cubic-bezier(0.23, 1, 0.32, 1);
}

.project-card:hover {
  box-shadow: 0 10px 28px -8px color-mix(in srgb, var(--doc-text) 18%, transparent);
  border-color: color-mix(in srgb, var(--doc-text) 28%, var(--doc-border));
  z-index: 1000 !important;
}

.project-card.hovering {
  z-index: 1000 !important;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.65rem;
  padding-bottom: 0.65rem;
  border-bottom: 1px solid var(--doc-border);
  font-family: var(--doc-font-mono, 'JetBrains Mono', monospace);
  font-size: 0.7rem;
}

.card-id {
  color: var(--doc-muted);
  letter-spacing: 0.04em;
  font-weight: 500;
}

.card-status-icons {
  display: flex;
  align-items: center;
  gap: 6px;
}

.status-icon {
  font-size: 0.75rem;
  transition: all 0.2s ease;
  cursor: default;
}

.status-icon.available {
  opacity: 1;
}

.status-icon:nth-child(1).available { color: #3B82F6; }
.status-icon:nth-child(2).available { color: #F59E0B; }
.status-icon:nth-child(3).available { color: #10B981; }

.status-icon.unavailable {
  color: var(--doc-dashed);
  opacity: 0.6;
}

.card-progress {
  display: flex;
  align-items: center;
  gap: 6px;
  letter-spacing: 0.5px;
  font-weight: 600;
  font-size: 0.65rem;
}

.status-dot {
  font-size: 0.5rem;
}

.card-progress.completed { color: var(--doc-badge-ok-fg, #10b981); }
.card-progress.in-progress { color: #f59e0b; }
.card-progress.not-started { color: var(--doc-muted); }
.card-status.pending { color: var(--doc-muted); }

.card-files-wrapper {
  position: relative;
  width: 100%;
  min-height: 2.4rem;
  max-height: 6.5rem;
  margin-bottom: 0.7rem;
  padding: 0.45rem 0.5rem;
  background: var(--doc-upload-surface, var(--doc-workbench-mid));
  border-radius: 2px;
  border: 1px solid var(--doc-border);
  overflow: hidden;
}

.files-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.files-more {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 3px 6px;
  font-family: 'JetBrains Mono', monospace;
  font-size: 0.6rem;
  color: var(--doc-muted);
  background: color-mix(in srgb, var(--doc-surface) 80%, var(--doc-border));
  border-radius: 3px;
  letter-spacing: 0.3px;
}

.file-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 8px;
  background: color-mix(in srgb, var(--doc-surface) 70%, var(--doc-border) 5%);
  border-radius: 999px;
  border: 1px solid var(--doc-border);
  transition: all 0.2s ease;
}

.file-item:hover {
  background: var(--doc-surface);
  transform: translateX(2px);
  border-color: var(--doc-border);
}

.file-tag {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 16px;
  padding: 0 4px;
  border-radius: 2px;
  font-family: 'JetBrains Mono', monospace;
  font-size: 0.55rem;
  font-weight: 600;
  line-height: 1;
  text-transform: uppercase;
  letter-spacing: 0.2px;
  flex-shrink: 0;
  min-width: 28px;
}

.file-tag.pdf { background: #f2e6e6; color: #a65a5a; }
.file-tag.doc { background: #e6eff5; color: #5a7ea6; }
.file-tag.xls { background: #e6f2e8; color: #5aa668; }
.file-tag.ppt { background: #f5efe6; color: #a6815a; }
.file-tag.txt { background: #f0f0f0; color: #757575; }
.file-tag.code { background: #eae6f2; color: #815aa6; }
.file-tag.img { background: #e6f2f2; color: #5aa6a6; }
.file-tag.zip { background: #f2f0e6; color: #a69b5a; }
.file-tag.other { background: #f3f4f6; color: #6b7280; }

.file-name {
  font-family: var(--doc-font-sans, 'Inter', sans-serif);
  font-size: 0.7rem;
  color: var(--doc-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  letter-spacing: 0.1px;
}

.files-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 2.2rem;
  color: var(--doc-muted);
}

.empty-file-icon {
  font-size: 1rem;
  opacity: 0.5;
}

.empty-file-text {
  font-family: 'JetBrains Mono', monospace;
  font-size: 0.7rem;
  letter-spacing: 0.5px;
}

.project-card:hover .card-files-wrapper {
  border-color: var(--doc-dashed);
  background: color-mix(in srgb, var(--doc-surface) 88%, var(--doc-border) 2%);
}

.card-title {
  font-family: 'Inter', -apple-system, sans-serif;
  font-size: 0.9rem;
  font-weight: 700;
  color: var(--doc-text);
  margin: 0 0 6px 0;
  line-height: 1.4;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  transition: color 0.3s ease;
}

.project-card:hover .card-title {
  color: var(--doc-accent, #00add8);
}

.card-desc {
  font-family: var(--doc-font-sans, 'Inter', sans-serif);
  font-size: 0.75rem;
  color: var(--doc-muted);
  margin: 0 0 0.9rem 0;
  line-height: 1.5;
  height: 2.1rem;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.card-footer {
  position: relative;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 0.65rem;
  border-top: 1px solid var(--doc-border);
  font-family: var(--doc-font-mono, 'JetBrains Mono', monospace);
  font-size: 0.65rem;
  color: var(--doc-muted);
  font-weight: 500;
  gap: 0.5rem;
}

.card-datetime {
  flex: 1;
  min-width: 0;
  letter-spacing: 0.04em;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.card-footer .card-progress {
  display: flex;
  align-items: center;
  gap: 6px;
  letter-spacing: 0.5px;
  font-weight: 600;
  font-size: 0.65rem;
}

.card-footer .status-dot {
  font-size: 0.5rem;
}

.card-footer .card-progress.completed { color: var(--doc-badge-ok-fg, #10b981); }
.card-footer .card-progress.in-progress { color: #f59e0b; }
.card-footer .card-progress.not-started { color: var(--doc-muted); }

.card-bottom-line {
  position: absolute;
  bottom: 0;
  left: 0;
  height: 2px;
  width: 0;
  background-color: var(--doc-cta-primary-bg, #111827);
  transition: width 0.5s cubic-bezier(0.23, 1, 0.32, 1);
  z-index: 20;
}

.project-card:hover .card-bottom-line {
  width: 100%;
}

.empty-state,
.loading-state {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 14px;
  min-height: 200px;
  padding: 2.5rem 1.5rem;
  color: var(--doc-muted);
}

.empty-icon {
  font-size: 2rem;
  opacity: 0.5;
}

.loading-spinner {
  width: 24px;
  height: 24px;
  border: 2px solid var(--doc-border);
  border-top-color: var(--doc-muted);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

@media (max-width: 1200px) {
  .project-card {
    width: 240px;
  }
}

@media (max-width: 768px) {
  .cards-container {
    padding: 0 20px;
  }
  .project-card {
    width: 200px;
  }
}

.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: color-mix(in srgb, #000 48%, transparent);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
  backdrop-filter: blur(4px);
}

.modal-content {
  background: var(--doc-surface);
  width: 560px;
  max-width: 90vw;
  max-height: 85vh;
  overflow-y: auto;
  border: 1px solid var(--doc-border);
  border-radius: 8px;
  box-shadow: var(--doc-shadow-soft);
  color: var(--doc-text);
}

.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.3s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-active .modal-content {
  transition: all 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.modal-leave-active .modal-content {
  transition: all 0.2s ease-in;
}

.modal-enter-from .modal-content {
  transform: scale(0.95) translateY(10px);
  opacity: 0;
}

.modal-leave-to .modal-content {
  transform: scale(0.95) translateY(10px);
  opacity: 0;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 32px;
  border-bottom: 1px solid var(--doc-code-bg);
  background: var(--doc-surface);
}

.modal-title-section {
  display: flex;
  align-items: center;
  gap: 16px;
}

.modal-id {
  font-family: 'JetBrains Mono', monospace;
  font-size: 1rem;
  font-weight: 600;
  color: var(--doc-text);
  letter-spacing: 0.5px;
}

.modal-progress {
  display: flex;
  align-items: center;
  gap: 6px;
  font-family: 'JetBrains Mono', monospace;
  font-size: 0.75rem;
  font-weight: 600;
  padding: 4px 8px;
  border-radius: 4px;
  background: var(--doc-code-bg);
  color: var(--doc-text);
}

.modal-progress.completed {
  color: var(--doc-badge-ok-fg, #10b981);
  background: var(--doc-badge-ok-bg);
}
.modal-progress.in-progress {
  color: #f59e0b;
  background: color-mix(in srgb, #f59e0b 14%, var(--doc-surface));
}
.modal-progress.not-started {
  color: var(--doc-muted);
  background: var(--doc-code-bg);
}

.modal-create-time {
  font-family: 'JetBrains Mono', monospace;
  font-size: 0.75rem;
  color: var(--doc-muted);
  letter-spacing: 0.3px;
}

.modal-close {
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  font-size: 1.5rem;
  color: var(--doc-muted);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
  border-radius: 6px;
}

.modal-close:hover {
  background: var(--doc-code-bg);
  color: var(--doc-text);
}

.modal-body {
  padding: 24px 32px;
}

.modal-section {
  margin-bottom: 24px;
}

.modal-section:last-child {
  margin-bottom: 0;
}

.modal-label {
  font-family: 'JetBrains Mono', monospace;
  font-size: 0.75rem;
  color: var(--doc-muted);
  text-transform: uppercase;
  letter-spacing: 1px;
  margin-bottom: 10px;
  font-weight: 500;
}

.modal-requirement {
  font-size: 0.95rem;
  color: var(--doc-text);
  line-height: 1.6;
  padding: 16px;
  background: var(--doc-code-bg);
  border: 1px solid var(--doc-border);
  border-radius: 8px;
  font-family: var(--doc-font-sans, 'Inter', system-ui, sans-serif);
}

.modal-files {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-height: 200px;
  overflow-y: auto;
  padding-right: 4px;
}

.modal-files::-webkit-scrollbar {
  width: 4px;
}

.modal-files::-webkit-scrollbar-track {
  background: var(--doc-code-bg);
  border-radius: 2px;
}

.modal-files::-webkit-scrollbar-thumb {
  background: var(--doc-dashed);
  border-radius: 2px;
}

.modal-files::-webkit-scrollbar-thumb:hover {
  background: var(--doc-muted);
}

.modal-file-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 14px;
  background: var(--doc-surface);
  border: 1px solid var(--doc-border);
  border-radius: 6px;
  transition: all 0.2s ease;
}

.modal-file-item:hover {
  border-color: var(--doc-dashed);
  box-shadow: 0 1px 2px 0 color-mix(in srgb, var(--doc-text) 8%, transparent);
}

.modal-file-name {
  font-size: 0.85rem;
  color: var(--doc-text);
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.modal-empty {
  font-size: 0.85rem;
  color: var(--doc-muted);
  padding: 16px;
  background: var(--doc-upload-surface, var(--doc-code-bg));
  border: 1px dashed var(--doc-border);
  border-radius: 6px;
  text-align: center;
}

.modal-divider {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 10px 32px 0;
  background: var(--doc-surface);
}

.divider-line {
  flex: 1;
  height: 1px;
  background: linear-gradient(90deg, transparent, var(--doc-border), transparent);
}

.divider-text {
  font-family: 'JetBrains Mono', monospace;
  font-size: 0.7rem;
  color: var(--doc-muted);
  letter-spacing: 2px;
  text-transform: uppercase;
  white-space: nowrap;
}

.modal-actions {
  display: flex;
  gap: 16px;
  padding: 20px 32px;
  background: var(--doc-surface);
}

.modal-btn {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 16px;
  border: 1px solid var(--doc-border);
  border-radius: 8px;
  background: var(--doc-surface);
  cursor: pointer;
  transition: all 0.2s ease;
  position: relative;
  overflow: hidden;
}

.modal-btn:hover:not(:disabled) {
  border-color: var(--doc-text);
  transform: translateY(-2px);
  box-shadow: 0 4px 6px -1px color-mix(in srgb, var(--doc-text) 12%, transparent);
}

.modal-btn:disabled {
  opacity: 0.55;
  cursor: not-allowed;
  background: var(--doc-cta-locked-bg, var(--doc-upload-surface));
  border-color: var(--doc-border);
  border-style: dashed;
  transform: none;
  box-shadow: none;
}

.modal-btn:disabled .btn-text,
.modal-btn:disabled .btn-step {
  color: var(--doc-cta-locked-fg, var(--doc-muted));
}

.modal-btn:disabled .btn-icon {
  opacity: 0.45;
}

.btn-step {
  font-family: 'JetBrains Mono', monospace;
  font-size: 0.6rem;
  font-weight: 500;
  color: var(--doc-muted);
  letter-spacing: 0.5px;
  text-transform: uppercase;
}

.btn-icon {
  font-size: 1.4rem;
  line-height: 1;
  transition: color 0.2s ease;
}

.btn-text {
  font-family: 'JetBrains Mono', monospace;
  font-size: 0.75rem;
  font-weight: 600;
  letter-spacing: 0.5px;
  color: var(--doc-text);
}

.modal-btn.btn-project .btn-icon { color: #3B82F6; }
.modal-btn.btn-simulation .btn-icon { color: #F59E0B; }
.modal-btn.btn-report .btn-icon { color: #10B981; }

.modal-btn:hover:not(:disabled) .btn-text {
  color: var(--doc-text);
}

.modal-playback-hint {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 32px 20px;
  background: var(--doc-surface);
}

.hint-text {
  font-family: 'JetBrains Mono', monospace;
  font-size: 0.7rem;
  color: var(--doc-muted);
  letter-spacing: 0.3px;
  text-align: center;
  line-height: 1.5;
}
</style>
