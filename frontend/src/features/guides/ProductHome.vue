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
        <h1 id="product-home-title"><span>用文字与图片，</span><span>创作你的下一幕。</span></h1>
      </div>
      <RouterLink to="/app/create" class="s2v-home-primary-link">开始创作</RouterLink>
    </section>
    <div class="s2v-home-content">
      <ProductModeShowcase />
    </div>
  </div>
</template>
