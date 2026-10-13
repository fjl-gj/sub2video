<script setup lang="ts">
import ProductHeroFilm from './ProductHeroFilm.vue'
import ProductModeShowcase from './ProductModeShowcase.vue'
import './home.css'
</script>

<template>
  <div class="s2v-home">
    <section class="s2v-home-hero" aria-labelledby="product-home-title">
      <ProductHeroFilm />
      <div class="s2v-home-hero-content">
        <h1 id="product-home-title">一句话，一个镜头。</h1>
        <RouterLink to="/app/create" class="s2v-home-primary-link">开始创作</RouterLink>
      </div>
      <ProductModeShowcase />
    </section>
  </div>
</template>
