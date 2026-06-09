<script>
  import { progress, streamingContent, streamingChapterIdx } from '../lib/stores.js';

  $: p = $progress;
  $: hasOutline = p?.chapters?.length > 0;

  const statusIcons = { pending: '', writing: '⏳', review: '👀', accepted: '✅' };
</script>

<div class="space-y-3">
  {#if !hasOutline}
    <div class="text-center py-20 text-base-content/40">
      <div class="text-5xl mb-3">📝</div>
      <p class="text-base mb-4">尚未生成大纲。请在右侧聊天中输入「请生成大纲」开始。</p>
    </div>
  {:else}
    <div class="card bg-base-200 shadow-sm">
      <div class="card-body p-4 gap-2">
        <h3 class="text-base font-semibold">📖 {p.title || ''}</h3>
        {#if p.core_prompt}
          <div>
            <span class="text-xs text-base-content/50">核心写作提示词</span>
            <div class="bg-base-300 rounded p-2 text-sm mt-0.5">{p.core_prompt}</div>
          </div>
        {/if}
        {#if p.story_synopsis}
          <div>
            <span class="text-xs text-base-content/50">故事梗概</span>
            <div class="bg-base-300 rounded p-2 text-sm mt-0.5">{p.story_synopsis}</div>
          </div>
        {/if}
        <h4 class="text-sm font-semibold mt-1 text-base-content/60">章节大纲</h4>
        <div class="overflow-x-auto">
          <table class="table table-sm">
            <thead><tr><th>#</th><th>标题</th><th>大纲</th><th>状态</th></tr></thead>
            <tbody>
              {#each p.chapters as ch}
                <tr>
                  <td>{ch.num}</td>
                  <td>{ch.title}</td>
                  <td class="max-w-md">{ch.outline}</td>
                  <td>{statusIcons[ch.status] || ''}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>

        {#if $streamingChapterIdx >= 0 && $streamingContent}
          <div class="bg-base-300 rounded p-3 mt-1 text-sm max-h-48 overflow-y-auto chapter-content">
            <div class="text-xs text-base-content/40 mb-1">正在生成中...</div>
            {$streamingContent}
          </div>
        {/if}
      </div>
    </div>
  {/if}
</div>
