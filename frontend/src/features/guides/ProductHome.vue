<script setup lang="ts">
import { ArrowDown, ArrowRight, ArrowUpRight, ChevronDown, Film, Images, Type } from 'lucide-vue-next'
import ProductHeroFilm from './ProductHeroFilm.vue'
import GuideImage from './GuideImage.vue'
import { shotGuides } from './catalog'
import './home.css'

const featured = shotGuides[0]
const modes = [
  { id: 'text', name: '文生视频', title: '先写下，脑海中的画面。', description: '描述场景、动作与镜头，让一段文字成为故事的开场。', material: '从一段画面描述开始', icon: Type, guide: shotGuides[0] },
  { id: 'multi_image', name: '多图生成视频', title: '让你的素材，有了下一步。', description: '带上自己的参考图片，探索产品、人物或场景的动态表达。', material: '使用你准备的参考图片', icon: Images, guide: shotGuides[1] },
  { id: 'first_last', name: '首尾帧生成视频', title: '给出起点，也设定终点。', description: '选定开始与结束的画面，再用描述连接中间发生的变化。', material: '准备首帧与尾帧各一张', icon: Film, guide: shotGuides[2] },
]
const steps = [
  { title: '准备这一镜', text: '写下画面描述，或上传自己的参考素材。教程可以提供一个起点。' },
  { title: '选参数，看估价', text: '选择模型、时长、分辨率与画幅。先确认估算费用，再决定是否提交。' },
  { title: '生成，等待结果', text: '提交后进入任务详情，查看进度和结果。生成过程无需停留在创作页。' },
  { title: '下载，留下好镜头', text: '下载已交付的视频，或保存到作品。用你熟悉的剪辑工具完成整支短片。' },
]
const questions = [
  { title: '第一次使用，需要准备什么？', answer: '文生视频只需要一段画面描述。多图模式需要准备自己的参考图片；首尾帧模式需要首帧与尾帧各一张。请使用你有权使用的素材，具体输入要求会在创作页展示。' },
  { title: '视频生成的费用怎么算？', answer: '创作页会根据你选择的时长、分辨率与音频选项给出实时估算。估价不会扣费；确认提交时会重新计算，并按当时价格创建任务与消费订单。' },
  { title: '可以直接制作一整部短片吗？', answer: '你可以在这里逐个生成镜头、查看结果、下载视频并保存到作品。整片拼接、字幕和后期剪辑仍需在你熟悉的剪辑软件中完成。建议先从一个清楚的短镜头开始。' },
  { title: '首页的影像是 Sub2Video 生成的吗？', answer: '首屏使用已授权的冰岛海岸实拍影像，作者与许可链接标注在画面下方。教程中的图片是构图参考，均不是平台生成成片或客户案例。你的生成结果取决于实际输入和所选模型。' },
]
</script>

<template>
  <div class="s2v-home">
    <section class="s2v-home-hero" aria-labelledby="product-home-title">
      <ProductHeroFilm />
      <div class="s2v-home-hero-topline" aria-hidden="true"><span>SUB2VIDEO / CREATIVE STUDIO</span><span>想象，即将开场</span></div>
      <div class="s2v-home-hero-content">
        <p class="s2v-home-kicker">创作者的 AI 视频工作台</p>
        <h1 id="product-home-title"><span>把一个想法，</span><span>拍成你的下一幕。</span></h1>
        <p class="s2v-home-hero-description">从一句描述、一组图片，到一个会动的镜头。<br />让灵感有画面，让故事往前走。</p>
        <div class="s2v-home-hero-actions">
          <RouterLink to="/app/create" class="s2v-guide-link-button s2v-guide-link-primary">开始创作<ArrowUpRight :size="17" /></RouterLink>
          <RouterLink to="/#guides" class="s2v-home-hero-secondary">浏览镜头教程<ArrowDown :size="16" /></RouterLink>
        </div>
      </div>
    </section>

    <div class="s2v-home-intro"><p>不必等一个完整的剧本。<br /><strong>先让你最想表达的那一幕，动起来。</strong></p><span>文字 · 图片 · 首尾帧<br />从你的素材出发</span></div>

    <section id="modes" class="s2v-home-section" aria-labelledby="home-modes-title">
      <div class="s2v-home-section-heading"><div><p class="s2v-home-kicker">01 / 开始的方式</p><h2 id="home-modes-title">灵感不同，起点也不同。</h2></div><p>用文字构思，用图片参考。<br />选一个适合这一镜的创作方式。</p></div>
      <div class="s2v-home-modes">
        <article v-for="(mode, index) in modes" :key="mode.id" class="s2v-home-mode">
          <div class="s2v-home-mode-art"><GuideImage :src="mode.guide.poster" :alt="`${mode.guide.title}构图参考`" /><span class="s2v-home-mode-number" aria-hidden="true">0{{ index + 1 }}</span></div>
          <div class="s2v-home-mode-copy"><p class="s2v-home-mode-label"><component :is="mode.icon" :size="16" />{{ mode.name }}</p><h3>{{ mode.title }}</h3><p>{{ mode.description }}</p><span class="s2v-home-material">{{ mode.material }}</span><RouterLink :to="{ path: '/app/create', query: { mode: mode.id } }" class="s2v-home-text-link">尝试{{ mode.name }}<ArrowUpRight :size="17" /></RouterLink></div>
        </article>
      </div>
      <p class="s2v-home-capability-note">可用模型、时长、分辨率与音频选项，以创作页当前支持的能力为准。</p>
    </section>

    <section id="guides" class="s2v-home-section s2v-home-tutorials" aria-labelledby="home-guides-title">
      <div class="s2v-home-section-heading"><div><p class="s2v-home-kicker">02 / 给灵感一个方向</p><h2 id="home-guides-title">好故事，从一个小镜头开始。</h2></div><p>不用面对空白页。跟着教程拆解画面，<br />再换成你自己的故事。</p></div>
      <div class="s2v-home-guide-layout">
        <RouterLink :to="`/guides/${featured.id}`" class="s2v-home-featured-guide">
          <GuideImage :src="featured.poster" alt="海岸来信构图参考" />
          <div class="s2v-home-featured-copy"><span class="s2v-home-kicker">本期镜头练习 / {{ featured.category }}</span><h3>{{ featured.title }}<ArrowUpRight :size="24" /></h3><p>{{ featured.description }}</p><div class="s2v-home-guide-meta"><span>建立场景</span><ArrowRight :size="13" /><span>观察动作</span><ArrowRight :size="13" /><span>情绪收束</span></div><span class="s2v-home-text-link">阅读镜头教程<ArrowRight :size="16" /></span></div>
        </RouterLink>
        <div class="s2v-home-guide-side">
          <RouterLink v-for="guide in shotGuides.slice(1)" :key="guide.id" :to="`/guides/${guide.id}`" class="s2v-home-small-guide"><GuideImage :src="guide.poster" :alt="`${guide.title}构图参考`" /><div><span class="s2v-home-kicker">{{ guide.category }}</span><h3>{{ guide.title }}<ArrowUpRight :size="19" /></h3><p>{{ guide.description }}</p><span class="s2v-home-small-guide-meta">{{ guide.shots.length }} 个镜头 · {{ guide.modeName }}</span></div></RouterLink>
          <div class="s2v-home-guide-note"><span>一个动作，一种情绪。</span><p>教程提供镜头思路与建议参数。参考图用于理解构图，实际创作请准备自己的素材。</p></div>
        </div>
      </div>
    </section>

    <section id="process" class="s2v-home-process" aria-labelledby="home-process-title"><div><p class="s2v-home-kicker">03 / 从开始到交付</p><h2 id="home-process-title">把心里的画面，<br />一步步留在眼前。</h2><p>先尝试，再调整。<br />每一次创作，都从一个明确的镜头开始。</p><RouterLink to="/guides/coast-letter" class="s2v-home-text-link">从「海岸来信」开始<ArrowRight :size="16" /></RouterLink></div><ol><li v-for="(step, index) in steps" :key="step.title"><span class="s2v-home-step-index" aria-hidden="true">0{{ index + 1 }}</span><div><h3>{{ step.title }}</h3><p>{{ step.text }}</p></div></li></ol></section>

    <section class="s2v-home-faq" aria-labelledby="home-faq-title"><div><p class="s2v-home-kicker">开始之前</p><h2 id="home-faq-title">你可能想知道。</h2></div><div class="s2v-home-questions"><details v-for="question in questions" :key="question.title"><summary>{{ question.title }}<ChevronDown :size="18" aria-hidden="true" /></summary><p>{{ question.answer }}</p></details></div></section>

    <section class="s2v-home-closing"><p class="s2v-home-kicker">下一幕，留给你的想象</p><h2>你的故事，<span>现在开场。</span></h2><RouterLink to="/app/create" class="s2v-guide-link-button s2v-guide-link-primary">开始我的第一镜<ArrowUpRight :size="17" /></RouterLink><p>从一个短镜头开始，慢慢做成自己的作品。</p></section>
  </div>
</template>
