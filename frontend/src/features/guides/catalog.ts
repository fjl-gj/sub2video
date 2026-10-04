import type { WorkflowInputType } from '@/shared/lib/create-draft'

export interface GuideShot {
  id: string
  title: string
  intent: string
  prompt: string
  durationSeconds: number
  resolution: string
  aspectRatio: string
}

export interface ShotGuide {
  id: string
  title: string
  category: string
  description: string
  mode: WorkflowInputType
  modeName: string
  poster: string
  materials: string
  preparation: string[]
  finishing: string
  shots: GuideShot[]
}

const shot = (id: string, title: string, intent: string, prompt: string): GuideShot => ({ id, title, intent, prompt, durationSeconds: 5, resolution: '768P', aspectRatio: '16:9' })

// Editorial exercises, not generated work or live capability/price data. Photos
// illustrate composition only; none are uploaded as a user's input material.
export const shotGuides: ShotGuide[] = [
  {
    id: 'coast-letter', title: '海岸来信', category: '故事练习', mode: 'text', modeName: '文生视频',
    description: '用三个简单镜头练习故事的起承转合：建立场景、观察动作，再让情绪安静地落下。',
    poster: 'https://images.unsplash.com/photo-1507525428034-b723cf961d3e?auto=format&fit=crop&w=1400&q=85',
    materials: '准备一段自己的故事描述，无需参考图片。',
    preparation: ['先确定时间、地点与一种情绪，三个镜头保持一致。', '每个镜头先描述一个主要动作，再补充运镜与光线。', '先生成短镜头，查看结果后再调整描述。'],
    finishing: '下载选中的镜头后，在你熟悉的剪辑软件中拼接；字幕、配乐和整片剪辑由你完成。',
    shots: [
      shot('arrival', '建立场景', '让观众先知道故事发生在哪里。', '清晨的海岸，远景，柔和晨光照亮海面，细小波浪缓缓涌向沙滩，镜头平稳向前推进，安静自然的电影感，保持地平线稳定。'),
      shot('detail', '观察动作', '将注意力放在一个简单、可辨认的动作上。', '清晨海岸的浅水边，近景，一封没有文字的折叠信纸轻轻落在干燥沙滩上，微风吹动纸角，镜头缓慢靠近，柔和自然光，真实细腻的质感。'),
      shot('departure', '情绪收束', '让镜头回到环境，给故事留一点余白。', '清晨海岸，镜头从沙滩缓慢抬向远处海平线，波浪舒缓起伏，天空透亮，柔和自然光，稳定运镜，安静留白，自然收束。'),
    ],
  },
  {
    id: 'product-light', title: '产品光影', category: '产品展示', mode: 'multi_image', modeName: '多图生成视频',
    description: '用自己的产品图练习全貌、材质与细节镜头，找到适合商品展示的节奏。',
    poster: 'https://images.unsplash.com/photo-1523275335684-37898b6baf30?auto=format&fit=crop&w=1200&q=85',
    materials: '上传自己的产品参考图，数量以当前生成能力要求为准。',
    preparation: ['准备清晰、主体完整的产品图，避免过度压缩。', '多张参考图尽量保持产品颜色和款式一致。', '每次先尝试一个运镜方向，检查外观与细节后再继续。'],
    finishing: '挑选产品外观准确的镜头，再在剪辑软件里加入品牌文字、价格与音乐。',
    shots: [
      shot('arrival', '展示全貌', '清楚呈现产品的轮廓与摆放关系。', '以我上传的产品参考图为准，产品置于干净桌面，中景，镜头缓慢向前移动，柔和侧光，保留产品形态与颜色，背景简洁，无额外文字。'),
      shot('detail', '突出材质', '靠近材质，控制镜头移动幅度。', '以我上传的产品参考图为准，产品材质近景，镜头平缓横移，柔和侧光掠过表面，强调真实纹理，保持产品外形与颜色一致，无额外物体。'),
      shot('departure', '保留细节', '用稳定构图结束这段产品展示。', '以我上传的产品参考图为准，关键细节特写，镜头缓慢后移，最终停留在产品完整轮廓，简洁背景，柔和光线，动作逐渐停止。'),
    ],
  },
  {
    id: 'scene-transition', title: '场景转场', category: '运镜练习', mode: 'first_last', modeName: '首尾帧生成视频',
    description: '先确定开始与结束的画面，再练习描述两张图片之间如何自然连接。',
    poster: 'https://images.unsplash.com/photo-1441974231531-c6227db76b6e?auto=format&fit=crop&w=1200&q=85',
    materials: '每个镜头准备自己的首帧和尾帧各一张。',
    preparation: ['首尾帧使用相近的画幅与视觉风格。', '先尝试空间变化较小的一组画面，观察生成结果。', '提示词描述运动过程，避免同时要求多个互相矛盾的动作。'],
    finishing: '检查首尾衔接和运动方向，挑选自然的镜头后再进行整片剪辑。',
    shots: [
      shot('arrival', '进入场景', '从起点画面平稳进入新的空间。', '从我上传的首帧自然过渡到尾帧，镜头缓慢沿林间小径向前移动，保持树林结构与光线连贯，平稳自然，不出现突兀变化。'),
      shot('detail', '推进主体', '用首尾画面控制运镜的方向。', '以首帧为起点、尾帧为终点，镜头平缓推进到林间主体，保留关键物体轮廓，光影连续，运动方向稳定，避免突然切换。'),
      shot('departure', '离开空间', '让运动逐渐停下，落在尾帧构图。', '从首帧平滑移动到尾帧，镜头缓慢抬高，光线穿过树叶，运动逐渐放缓并稳定在尾帧构图，保持场景自然连贯。'),
    ],
  },
]

export function findGuide(id: unknown) { return typeof id === 'string' ? shotGuides.find((guide) => guide.id === id) : undefined }
export function guideCreateRoute(guide: ShotGuide, item = guide.shots[0]) {
  return { path: '/app/create', query: { guide: guide.id, shot: item.id } }
}
