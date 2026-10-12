'use strict';
(() => {
  const byId = (id) => document.getElementById(id);
  const poster = '../../../frontend/public/brand/auth/creator-studio-poster.jpg';
  const last = '../../../frontend/public/brand/auth/creator-studio-last.jpg';
  const initialPrompt = '女孩在工作室里轻轻抬头，望向窗外。镜头缓慢推进，保持人物与画风一致，阳光照亮她的侧脸。';
  const state = { scenario: 'draft', mode: 'frames', prompt: initialPrompt, ratio: '16:9', duration: 5, estimated: false, saved: false, busy: false, messages: [], materials: [ {src:poster,name:'示例首帧'}, {src:last,name:'示例尾帧'} ], view:'chat', taskSnapshot:null };
  const taskStates = ['running','review','delivered','refunded'];
  const modes = {
    text: {name:'文生视频',requirement:'写下场景、主体和动作。无需图片，直接准备镜头描述。',response:'这一镜从文字开始。核对描述与参数后，再查看费用。',labels:[]},
    frames: {name:'首尾帧',requirement:'首帧决定起点，尾帧决定落点。两张图片都需要准备。',response:'用首尾帧控制这一镜。先核对画面，再查看费用。',labels:['首帧','尾帧']},
    multi: {name:'多图',requirement:'至少准备一张参考图。分别说明人物、场景与画面的关系。',response:'用参考图片约束画面。先核对素材，再查看费用。',labels:['参考图 1','参考图 2']}
  };
  function escape(value){return String(value).replace(/[&<>"']/g,(c)=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));}
  function money(){return (state.duration * .03).toFixed(2);}
  function feedback(text){byId('inline-feedback').textContent=text;byId('inline-feedback').hidden=false;}
  function view(which){state.view=which;document.querySelector('.workbench').dataset.view=which;document.querySelectorAll('[data-view]').forEach(button=>{if(button.tagName==='BUTTON'){button.classList.toggle('active',button.dataset.view===which);button.setAttribute('aria-pressed',String(button.dataset.view===which));}});}
  function dialog(id){byId(id).showModal();}
  function help(title,body){byId('help-title').textContent=title;byId('help-body').innerHTML=body;dialog('help-dialog');}
  function requiredReady(){return state.mode==='text'||(state.mode==='frames'?Boolean(state.materials[0]?.src&&state.materials[1]?.src):state.materials.some(item=>item.src));}
  function card(){return `<details class="shot-card" ${matchMedia('(min-width:1101px)').matches?'open':''}><summary class="shot-card-head"><span class="draft-title">镜头草稿</span><span>${modes[state.mode].name} · 展开描述</span></summary><p>${escape(state.prompt)}</p><p class="draft-status">${requiredReady()?'描述与素材已准备':'素材尚未准备完整'} · ${state.duration} 秒 · 768P · ${state.ratio}</p></details>`;}
  function conversation(){
    if(state.scenario==='empty'){
      byId('conversation').innerHTML='<div class="empty-intro"><h2>你想拍什么？</h2><p>先说出一个画面，再一起把它准备成可生成的镜头。</p><div class="prompt-examples"><button data-example="anime">动漫人物的一个动作</button><button data-example="product">商品的五秒展示</button><button data-example="scene">一个有氛围的场景</button></div></div>';
      return;
    }
    const response = {unavailable:'暂时无法生成。你可以继续修改描述与参数，已填写内容保留在这一镜里。',running:'这一镜已经提交。新的描述修改用于下次草稿，不改变当前任务。',review:'先核对原提交。你可以记录新的想法，当前不会再次生成。'}[state.scenario]||modes[state.mode].response;
    const userText=state.messages.length?state.messages[state.messages.length-1]:(state.mode==='frames'?'把这两张图做成一镜，让女孩轻轻抬头，镜头慢慢推进。':state.prompt);
    byId('conversation').innerHTML=`<div class="message-user">${escape(userText)}</div><div class="message-assistant"><div class="speaker"><span class="assistant-mark">S2V</span><span>镜头建议 · 示例回复</span></div><p>${response}</p>${card()}<div class="edit-actions"><button data-edit="镜头缓慢推进。">缓慢推进</button><button data-edit="保持人物面容、服装与画风一致。">保持人物一致</button><button data-edit="减小动作幅度，避免快速转身。">降低动作幅度</button></div></div>`;
  }
  function renderMaterials(){
    const labels=modes[state.mode].labels;
    byId('material-section').hidden=state.mode==='text';
    byId('materials').innerHTML=labels.map((label,i)=>`<button class="material" data-material="${i}" aria-label="选择${label}">${state.materials[i]?.src?`<img src="${escape(state.materials[i].src)}" alt="${label}参考预览">`:'<span class="missing-frame" aria-hidden="true">＋</span>'}<div><strong>${label}</strong><span class="frame-name">${escape(state.materials[i]?.name||'选择图片')}</span></div></button>`).join('');
    byId('material-hint').textContent=state.materials.some(item=>item.local)?'本地图片仅预览，刷新后需重新选择。':'示例图片只用于审阅设计。可换成你的本地图片。';
  }
  function action(){
    const node=byId('next-action');
    if(state.scenario==='running'){
      node.innerHTML='<div class="status-message"><h3>正在生成 · 状态示意</h3><p>可以离开当前页面，稍后在任务中查看结果。新的描述修改只用于下一次草稿。</p><div class="task-progress"><span class="done">已提交</span><span class="current">执行中</span><span>等待交付</span></div></div><button class="secondary" data-action="task">查看任务详情</button>';
    }else if(state.scenario==='review'){
      node.innerHTML='<div class="status-message warning"><h3>提交结果待核对</h3><p>暂时无法确认是否已受理。请勿重复生成，费用状态以任务详情为准。</p><button class="text-control" data-action="task">查看任务与处理进度</button></div>';
    }else if(state.scenario==='unavailable'){
      node.innerHTML='<div class="status-message warning"><h3>生成服务暂未就绪</h3><p>当前无法估价或提交。你可以继续整理镜头，生成前不会扣费。</p><button class="text-control" data-action="help">联系帮助与工单</button></div><button class="primary" disabled>暂不可生成</button>';
    }else if(state.scenario==='refunded'){
      node.innerHTML=`<div class="status-message danger"><h3>生成失败 · 已退款示意</h3><p>本次未交付视频。示例退款 ¥${money()}，时间：10 月 09 日 14:32。</p><button class="text-control" data-action="task">查看退款记录</button></div><button class="primary" id="restart">按原参数重新准备</button>`;
    }else if(state.scenario==='delivered'){
      node.innerHTML='<div class="status-message success"><h3>交付后的操作布局</h3><p>上方为品牌参考视频，用于展示播放器和交付操作。它不是本次生成结果。</p></div>';
    }else if(!state.prompt.trim()){
      node.innerHTML='<p>写下镜头描述后，再准备素材与参数。</p><button class="primary" disabled>先描述镜头</button>';
    }else if(!requiredReady()){
      node.innerHTML=`<p>${state.mode==='frames'?'请准备首帧与尾帧。':'请至少添加一张参考图片。'}</p><button class="primary" id="prepare-materials">准备素材</button>`;
    }else if(state.estimated){
      node.innerHTML=`<div class="cost-row"><div><span>本次估价 · 示例金额</span><b>¥${money()}</b></div><button class="text-control" id="reestimate">重新查看费用</button></div><p>按提交时价格预扣；失败后按真实任务状态退款。</p><button class="primary" id="generate">确认生成 · 原型演示</button><p class="cost-detail">5 秒 × ¥0.03 / 秒（仅示例）；不锁价。</p>`;
    }else{
      node.innerHTML='<p>描述、素材和参数准备完成后，先查看费用。</p><button class="primary" id="estimate">查看费用 · 原型演示</button>';
    }
  }
  function render(){
    byId('scenario').value=state.scenario;
    byId('mode-requirement').textContent=modes[state.mode].requirement;
    document.querySelectorAll('[data-mode]').forEach(button=>{button.classList.toggle('active',button.dataset.mode===state.mode);button.setAttribute('aria-pressed',String(button.dataset.mode===state.mode));});
    byId('project-title').textContent=state.scenario==='empty'?'新建一个镜头':'工作室里的下一幕';
    byId('duration-display').textContent=`${state.duration} 秒`;
    byId('ratio-display').textContent=state.ratio;
    const delivered=state.scenario==='delivered';
    const empty=state.scenario==='empty'||(!state.materials.some(item=>item.src)&&state.mode!=='text');
    byId('preview-video').hidden=!delivered;byId('preview-image').hidden=delivered||empty;byId('stage-empty').hidden=!empty;
    byId('media-label').hidden=empty||delivered;byId('delivery-actions').hidden=!delivered;
    byId('preview-caption').textContent=delivered?'参考视频 · 非生成结果':(empty?'等待画面':'参考画面');
    byId('preview-image').src=state.materials.find(item=>item.src)?.src||poster;
    if(!delivered)byId('preview-video').pause();
    byId('wallet').textContent=['running','review','delivered'].includes(state.scenario)?'¥19.85':'¥20.00';
    const persistentStatus={running:'当前任务执行中，修改仅用于下次草稿。',review:'原提交正在核对，请勿重复生成。',unavailable:'生成服务未就绪，仍可继续整理草稿。'}[state.scenario];
    byId('chat-task-status').hidden=!persistentStatus;
    byId('chat-task-status').innerHTML=persistentStatus?`${persistentStatus}<button class="text-control" data-action="${state.scenario==='unavailable'?'help':'task'}">${state.scenario==='unavailable'?'查看帮助':'查看任务'}</button>`:'';
    byId('draft-note').textContent=state.saved?'原型草稿已保存：仅描述、模式与参数。本地图片不持久化。':'当前为设计示例。对话与画面始终围绕同一个镜头。';
    conversation();renderMaterials();action();
  }
  function setScenario(which){
    if(taskStates.includes(which)&&!taskStates.includes(state.scenario)){
      state.taskSnapshot={mode:state.mode,prompt:state.prompt,ratio:state.ratio,duration:state.duration,materials:state.materials.map(item=>({...item}))};
    }
    if(!taskStates.includes(which))state.taskSnapshot=null;
    state.scenario=which;state.busy=false;state.estimated=which==='estimate';
    if(which==='empty'){state.prompt='';state.materials=[{},{}];state.messages=[];state.mode='text';byId('message').value='';}
    else if(!state.prompt){state.prompt=initialPrompt;state.mode='frames';state.materials=[{src:poster,name:'示例首帧'},{src:last,name:'示例尾帧'}];}
    byId('inline-feedback').hidden=true;render();
  }
  function invalidate(){state.estimated=false;if(!['empty','unavailable',...taskStates].includes(state.scenario))state.scenario='draft';state.saved=false;}
  function restoreTaskParameters(){
    if(state.taskSnapshot){Object.assign(state,state.taskSnapshot);state.messages=[];}
    setScenario('draft');view('chat');feedback('已回填原任务参数，未提交或再次扣费。');
  }
  function materialsDialog(){
    if(state.mode==='text'){feedback('文生视频无需图片。可切换首尾帧或多图模式。');return;}
    byId('material-dialog-hint').textContent=modes[state.mode].requirement;
    byId('material-editors').innerHTML=modes[state.mode].labels.map((label,i)=>`<div class="material-editor">${state.materials[i]?.src?`<img src="${escape(state.materials[i].src)}" alt="${label}参考预览">`:''}<label>${label}<input type="file" accept="image/png,image/jpeg,image/webp" data-file="${i}"></label><button class="text-control" data-remove="${i}">移除</button></div>`).join('');
    dialog('materials-dialog');
  }
  function submitDescription(){
    const value=byId('message').value.trim();
    if(!value){feedback('请先写下镜头描述或修改内容。');byId('message').focus();return;}
    const hadDraft=Boolean(state.prompt);
    state.prompt=hadDraft?`${state.prompt}\n修改要求：${value}`:value;state.messages.push(value);invalidate();
    if(state.scenario==='empty')state.scenario='draft';byId('message').value='';render();
    feedback('已整理到镜头草稿。示例为本地规则处理，未调用对话模型。');
  }
  byId('scenario').addEventListener('change',event=>setScenario(event.target.value));
  byId('send-message').addEventListener('click',submitDescription);
  byId('message').addEventListener('keydown',event=>{if(event.key==='Enter'&&(event.ctrlKey||event.metaKey)){event.preventDefault();submitDescription();}});
  byId('open-materials').addEventListener('click',materialsDialog);byId('manage-materials').addEventListener('click',materialsDialog);
  byId('edit-settings').addEventListener('click',()=>{byId('duration-input').value=String(state.duration);byId('ratio-input').value=state.ratio;dialog('settings-dialog');});
  byId('settings-form').addEventListener('submit',event=>{event.preventDefault();state.duration=Number(byId('duration-input').value);state.ratio=byId('ratio-input').value;invalidate();byId('settings-dialog').close();render();feedback('参数已更新，请重新查看费用。');});
  byId('sample-materials').addEventListener('click',()=>{state.materials=[{src:poster,name:'示例首帧'},{src:last,name:'示例尾帧'}];invalidate();byId('materials-dialog').close();render();feedback('已使用示例图片，未上传到服务器。');});
  function saveDraft(){
    try{localStorage.setItem('s2v-chat-design-draft-v1',JSON.stringify({prompt:state.prompt,mode:state.mode,ratio:state.ratio,duration:state.duration}));state.saved=true;render();feedback('描述与参数已保存在本机原型。未保存到真实作品，本地图片不持久化。');}catch{feedback('浏览器不允许本地保存。请复制描述以保留内容。');}
  }
  byId('save-draft').addEventListener('click',saveDraft);byId('save-mobile').addEventListener('click',saveDraft);
  byId('new-draft').addEventListener('click',()=>{
    if(['running','review'].includes(state.scenario)){help('先查看当前任务','<p>当前这一镜仍在执行或核对中。原提交状态会保留，不能通过新建或修改描述将它清除。</p><div class="dialog-actions"><button class="primary" data-action="task">查看任务详情</button></div>');return;}
    help('新建创作','<p>新建会清空本轮原型的描述与素材。可以先保留描述与参数。</p><div class="dialog-actions"><button class="secondary" data-close="help-dialog">继续这一镜</button><button class="primary" id="confirm-new">新建空白镜头</button></div>');
  });
  byId('nav-toggle').addEventListener('click',()=>{const isOpen=document.querySelector('.sidebar').classList.toggle('open');byId('nav-toggle').setAttribute('aria-expanded',String(isOpen));});
  byId('save-work').addEventListener('click',()=>{byId('save-location').hidden=true;dialog('save-dialog');});
  byId('confirm-save').addEventListener('click',()=>{byId('save-location').hidden=false;byId('save-location').textContent=`原型保存位置：资产 / ${byId('destination').value} / 成片。正式保存需服务端成功返回文件记录。`;});
  byId('recreate').addEventListener('click',restoreTaskParameters);
  document.addEventListener('click',event=>{
    const target=event.target.closest('button,a');if(!target)return;
    if(target.dataset.close){byId(target.dataset.close).close();return;}
    if(target.dataset.mode){state.mode=target.dataset.mode;invalidate();render();return;}
    if(target.dataset.view){view(target.dataset.view);return;}
    if(target.dataset.material!==undefined){materialsDialog();return;}
    if(target.dataset.remove!==undefined){state.materials[Number(target.dataset.remove)]={};invalidate();byId('materials-dialog').close();render();feedback('素材已移除，需要重新查看费用。');return;}
    if(target.dataset.edit){state.prompt+=`\n${target.dataset.edit}`;state.messages.push(target.textContent);invalidate();render();feedback(['running','review'].includes(state.scenario)?'修改已保留，原提交仍在执行或核对中。':'修改已加入草稿。生成前需要重新查看费用。');return;}
    if(target.dataset.example){state.prompt={anime:initialPrompt,product:'一只白色香水瓶置于石台上，镜头缓慢环绕，柔和侧光突出玻璃质感，背景保持简洁。',scene:'清晨海边，薄雾缓缓散开，低机位镜头向前移动，阳光落在轻柔的浪花上。'}[target.dataset.example];state.scenario='draft';invalidate();render();return;}
    if(['estimate','reestimate'].includes(target.id)){state.estimated=true;state.scenario='estimate';render();view('shot');return;}
    if(target.id==='generate'&&!state.busy&&state.estimated){state.busy=true;setScenario('running');view('shot');return;}
    if(target.id==='prepare-materials'){materialsDialog();return;}
    if(target.id==='restart'){restoreTaskParameters();return;}
    if(target.id==='confirm-new'){try{localStorage.removeItem('s2v-chat-design-draft-v1');}catch{}state.saved=false;byId('help-dialog').close();setScenario('empty');view('chat');return;}
    if(target.dataset.action){
      event.preventDefault();document.querySelector('.sidebar').classList.remove('open');byId('nav-toggle').setAttribute('aria-expanded','false');
      if(target.dataset.action==='creation'){view('chat');return;}
      if(target.dataset.action==='help')help('帮助与工单','<p>正式工单应附带当前模式、参数与错误状态；不附带供应商或凭据。</p><p>本设计稿不创建工单。服务未就绪时，保留描述与参数，待恢复后重新估价。</p>');
      else if(target.dataset.action==='task')help('任务详情 · 状态示意',`<p>当前演示：${escape(state.scenario)}。正式页面展示真实任务 ID、状态、退款与交付记录。</p><p>提交不确定时先核对原任务，不重复提交，也不预先声称退款。</p>`);
      else help(target.dataset.destination||'首页概览','<p>本轮设计稿聚焦创作页。正式产品继续使用现有页面与导航，不新增第二套资产或任务系统。</p>');
    }
  });
  document.addEventListener('change',event=>{
    if(event.target.dataset.file===undefined)return;
    const file=event.target.files[0];if(!file)return;
    if(!['image/png','image/jpeg','image/webp'].includes(file.type)){feedback('请选择 PNG、JPEG 或 WebP 图片。');return;}
    const index=Number(event.target.dataset.file);const previous=state.materials[index];if(previous?.local)URL.revokeObjectURL(previous.src);
    state.materials[index]={src:URL.createObjectURL(file),name:file.name,local:true};invalidate();byId('materials-dialog').close();render();feedback('已在本地预览图片，没有上传或调用生成服务。');
  });
  document.addEventListener('keydown',event=>{if(event.key==='Escape'){document.querySelector('.sidebar').classList.remove('open');byId('nav-toggle').setAttribute('aria-expanded','false');}});
  const initial=new URLSearchParams(location.search).get('state');
  setScenario(['empty','draft','estimate','running','delivered','unavailable','refunded','review'].includes(initial)?initial:'draft');
  if(!initial){
    try{
      const saved=JSON.parse(localStorage.getItem('s2v-chat-design-draft-v1')||'null');
      if(saved&&typeof saved.prompt==='string'&&Object.hasOwn(modes,saved.mode)&&['16:9','9:16'].includes(saved.ratio)&&saved.duration===5){
        state.prompt=saved.prompt;state.mode=saved.mode;state.ratio=saved.ratio;state.materials=[{},{}];state.saved=true;render();
        feedback('已恢复原型描述与参数。尚未上传的图片需要重新选择。');
      }
    }catch{}
  }
})();
