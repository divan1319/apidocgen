<script setup lang="ts">
import { ref, onMounted } from 'vue'
import type { Project } from '@/types'
import { api } from '@/api/client'
import ProjectCard from '@/components/ProjectCard.vue'

const projects = ref<Project[]>([])
const loading = ref(true)
const error = ref('')
const generatingSlug = ref<string | null>(null)
const generateLog = ref('')
const showLog = ref(false)

async function loadProjects() {
  try {
    loading.value = true
    error.value = ''
    projects.value = await api.listProjects()
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function handleGenerate(slug: string) {
  generatingSlug.value = slug
  generateLog.value = ''
  showLog.value = true
  try {
    const res = await api.generateDocs(slug)
    generateLog.value = res.log
    await loadProjects()
  } catch (e: any) {
    generateLog.value = `Error: ${e.message}`
  } finally {
    generatingSlug.value = null
  }
}

async function handleDelete(slug: string) {
  if (!confirm(`¿Eliminar proyecto "${slug}" y sus archivos asociados?`)) return
  try {
    await api.deleteProject(slug)
    await loadProjects()
  } catch (e: any) {
    error.value = e.message
  }
}

onMounted(loadProjects)
</script>

<template>
  <div>
    <div class="flex items-end justify-between mb-8">
      <div>
        <h1 class="text-3xl font-extrabold tracking-tight bg-gradient-to-r from-text to-accent bg-clip-text text-transparent">
          Proyectos
        </h1>
        <p class="text-text-muted text-sm mt-1">
          {{ projects.length }} proyecto(s) de documentación
        </p>
      </div>
      <router-link
        to="/projects/new"
        class="bg-accent text-bg text-xs font-semibold px-4 py-2 rounded-md hover:opacity-90 transition-opacity"
      >
        + Nuevo proyecto
      </router-link>
    </div>

    <div v-if="error" class="bg-danger/10 border border-danger/30 text-danger text-sm rounded-lg px-4 py-3 mb-6">
      {{ error }}
    </div>

    <div v-if="loading" class="text-center py-20 text-text-muted text-sm">
      Cargando proyectos...
    </div>

    <div v-else-if="projects.length === 0" class="text-center py-20">
      <p class="text-text-muted text-sm mb-4">No hay proyectos configurados todavía.</p>
      <router-link
        to="/projects/new"
        class="inline-block bg-accent text-bg text-sm font-semibold px-5 py-2.5 rounded-md hover:opacity-90 transition-opacity"
      >
        Crear primer proyecto
      </router-link>
    </div>

    <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-5">
      <ProjectCard
        v-for="p in projects"
        :key="p.slug"
        :project="p"
        :generating="generatingSlug === p.slug"
        @generate="handleGenerate"
        @delete="handleDelete"
      />
    </div>

    <!-- Log modal -->
    <Teleport to="body">
      <div v-if="showLog" class="fixed inset-0 z-50 flex items-center justify-center p-4" @click.self="showLog = false">
        <div class="fixed inset-0 bg-black/60" @click="showLog = false" />
        <div class="relative bg-surface border border-border rounded-xl w-full max-w-2xl max-h-[80vh] flex flex-col overflow-hidden">
          <div class="flex items-center justify-between px-5 py-3 border-b border-border">
            <h3 class="text-sm font-bold text-text">
              {{ generatingSlug ? 'Generando documentación...' : 'Resultado de generación' }}
            </h3>
            <button
              @click="showLog = false"
              :disabled="!!generatingSlug"
              class="text-text-muted hover:text-text text-lg cursor-pointer disabled:opacity-30"
            >
              ✕
            </button>
          </div>
          <div class="flex-1 overflow-auto p-5">
            <pre class="text-xs font-mono text-text-muted whitespace-pre-wrap leading-relaxed">{{ generateLog || 'Esperando respuesta del servidor...' }}</pre>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>
