<script setup lang="ts">
import type { Project } from '@/types'

defineProps<{
  project: Project
  generating: boolean
}>()

const emit = defineEmits<{
  generate: [slug: string]
  delete: [slug: string]
}>()

function langBadgeClass(lang: string): string {
  const map: Record<string, string> = {
    laravel: 'bg-warning/10 text-warning',
    dotnet: 'bg-accent-dim text-accent',
  }
  return map[lang] || 'bg-surface-2 text-text-muted'
}
</script>

<template>
  <div class="bg-surface border border-border rounded-xl p-6 flex flex-col gap-4 hover:border-accent transition-colors group">
    <div class="flex items-start justify-between">
      <div class="flex-1 min-w-0">
        <h3 class="text-lg font-bold text-text truncate">{{ project.name }}</h3>
        <p class="text-text-muted text-xs font-mono mt-0.5">{{ project.slug }}</p>
      </div>
      <span :class="[langBadgeClass(project.lang), 'text-[10px] font-mono font-medium px-2.5 py-0.5 rounded-md uppercase tracking-wider shrink-0']">
        {{ project.lang }}
      </span>
    </div>

    <p class="text-text-muted text-sm leading-relaxed line-clamp-2">{{ project.title }}</p>

    <div class="text-xs font-mono text-text-muted space-y-1">
      <div class="truncate" :title="project.routes">rutas: {{ project.routes }}</div>
      <div class="truncate" :title="project.root">root: {{ project.root }}</div>
    </div>

    <div class="flex items-center gap-2 mt-auto pt-2 border-t border-border">
      <button
        @click="emit('generate', project.slug)"
        :disabled="generating"
        class="flex-1 text-center text-xs font-semibold py-2 rounded-md transition-all cursor-pointer disabled:opacity-50 disabled:cursor-wait"
        :class="generating ? 'bg-surface-2 text-text-muted' : 'bg-accent text-bg hover:opacity-90'"
      >
        {{ generating ? 'Generando...' : 'Generar' }}
      </button>

      <router-link
        v-if="project.has_docs"
        :to="{ name: 'docs-viewer', params: { slug: project.slug } }"
        class="text-xs font-medium text-success bg-success/10 px-3 py-2 rounded-md hover:bg-success/20 transition-colors"
      >
        Ver docs
      </router-link>

      <router-link
        :to="{ name: 'project-edit', params: { slug: project.slug } }"
        class="text-xs text-text-muted hover:text-text px-2 py-2 transition-colors"
        title="Editar"
      >
        ✎
      </router-link>

      <button
        @click="emit('delete', project.slug)"
        class="text-xs text-text-muted hover:text-danger px-2 py-2 transition-colors cursor-pointer"
        title="Eliminar"
      >
        ✕
      </button>
    </div>
  </div>
</template>
