<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '@/api/client'

const route = useRoute()
const slug = computed(() => route.params.slug as string)

const exists = ref(false)
const docUrl = ref('')
const loading = ref(true)

async function checkDocs() {
  loading.value = true
  try {
    const res = await api.checkDocs(slug.value)
    exists.value = res.exists
    docUrl.value = res.url
  } catch {
    exists.value = false
  } finally {
    loading.value = false
  }
}

onMounted(checkDocs)
</script>

<template>
  <div class="flex flex-col h-[calc(100vh-8rem)]">
    <div class="flex items-center justify-between mb-4">
      <div class="flex items-center gap-3">
        <router-link to="/" class="text-text-muted text-xs hover:text-text transition-colors">&larr; Proyectos</router-link>
        <span class="text-border">|</span>
        <h1 class="text-lg font-bold text-text">{{ slug }}</h1>
      </div>
      <a
        v-if="exists"
        :href="docUrl"
        target="_blank"
        class="text-xs text-accent hover:underline"
      >
        Abrir en nueva pestaña ↗
      </a>
    </div>

    <div v-if="loading" class="flex-1 flex items-center justify-center text-text-muted text-sm">
      Verificando documentación...
    </div>

    <div v-else-if="!exists" class="flex-1 flex items-center justify-center">
      <div class="text-center">
        <p class="text-text-muted text-sm mb-4">No hay documentación generada para este proyecto.</p>
        <router-link to="/" class="text-accent text-sm hover:underline">
          Volver al dashboard para generar
        </router-link>
      </div>
    </div>

    <iframe
      v-else
      :src="docUrl"
      class="flex-1 w-full rounded-lg border border-border bg-surface"
      frameborder="0"
    />
  </div>
</template>
