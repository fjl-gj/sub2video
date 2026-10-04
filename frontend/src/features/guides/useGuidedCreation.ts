import { computed, ref, watch, type Ref } from 'vue'
import { useRoute } from 'vue-router'
import type { CreationCatalog, CreationOptions } from '@/api/client'
import type { WorkflowInputType } from '@/shared/lib/create-draft'
import { findGuide } from './catalog'

export function useGuidedCreation(state: {
  mode: Ref<WorkflowInputType>
  prompt: Ref<string>
  duration: Ref<string>
  resolution: Ref<string>
  ratio: Ref<string>
  catalog: Ref<CreationCatalog>
  catalogLoaded: Ref<boolean>
  hasMaterials: () => boolean
  disableAudio: () => void
}) {
  const route = useRoute()
  const requested = computed(() => route.query.guide !== undefined || route.query.shot !== undefined)
  const guide = computed(() => findGuide(route.query.guide))
  const shot = computed(() => guide.value?.shots.find((item) => item.id === route.query.shot))
  const invalid = computed(() => requested.value && (!guide.value || !shot.value))
  const unavailable = computed(() => Boolean(guide.value && state.catalogLoaded.value
    && !state.catalog.value.modes.some((mode) => mode.enabled && mode.inputType === guide.value!.mode)))
  const applied = ref(false)
  const adjusted = ref(false)

  function apply() {
    if (!guide.value || !shot.value || unavailable.value) return
    state.mode.value = guide.value.mode
    state.prompt.value = shot.value.prompt
    state.duration.value = String(shot.value.durationSeconds)
    state.resolution.value = shot.value.resolution
    state.ratio.value = shot.value.aspectRatio
    state.disableAudio()
    applied.value = true
    adjusted.value = false
  }

  function initialize() {
    applied.value = false
    adjusted.value = false
    if (!state.prompt.value.trim() && !state.hasMaterials()) apply()
  }

  function checkOptions(options: CreationOptions) {
    if (!applied.value || !shot.value) return
    adjusted.value = !options.durations.includes(shot.value.durationSeconds)
      || !options.resolutions.includes(shot.value.resolution)
      || !options.aspectRatios.includes(shot.value.aspectRatio)
  }

  watch(() => [route.query.guide, route.query.shot], initialize)
  return { requested, guide, shot, invalid, unavailable, applied, adjusted, apply, initialize, checkOptions }
}
