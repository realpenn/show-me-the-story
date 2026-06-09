<script>
  import { progress, taskRunning, streamingContent, streamingChapterIdx, selectedChapter, addToast } from '../lib/stores.js';

  export let sendToChat = async () => {};

  $: p = $progress;
  $: inWriting = p?.phase === 'writing';
  $: chapters = p?.chapters || [];
  $: total = chapters.length;
  $: accepted = chapters.filter(c => c.status === 'accepted').length;
  $: pct = total > 0 ? Math.round(accepted / total * 100) : 0;
  $: ch = $selectedChapter >= 0 && $selectedChapter < chapters.length ? chapters[$selectedChapter] : null;
  $: isCurrent = ch && p?.current_chapter_index === $selectedChapter;

  $: displayContent = ($streamingChapterIdx === $selectedChapter && $streamingContent) ? $streamingContent : (ch?.content || '');

  const statusIcons = { pending: '', writing: '⏳', review: '👀', accepted: '✅' };

  function selectChapter(i) {
    selectedChapter.set(i);
  }

  async function doGenerate() {
    await sendToChat(`请生成第 ${chapters[$selectedChapter]?.num || ''} 章`);
  }

  async function doConfirm() {
    await sendToChat(`请确认第 ${chapters[$selectedChapter]?.num || ''} 章`);
  }
</script>

{#if !inWriting}
  <div class="text-center py-16 text-base-content/50">
    <div class="text-5xl mb-4">✍️</div>
    <p class="text-base">请先完成大纲确认，再进入写作阶段。</p>
    <p class="text-sm mt-2 text-base-content/30">在右侧聊天中输入「请确认大纲」</p>
  </div>
{:else}
  <div class="space-y-3">
    <!-- Progress -->
    <div class="card bg-base-200 shadow-sm">
      <div class="card-body p-4 gap-2">
        <h2 class="card-title text-base">写作进度</h2>
        <progress class="progress progress-primary w-full" value={pct} max="100"></progress>
        <div class="text-sm text-base-content/50">{pct}% ({accepted}/{total})</div>
      </div>
    </div>

    <!-- Chapter viewer -->
    <div class="grid grid-cols-[220px_1fr] gap-3" style="min-height:400px">
      <!-- Chapter list -->
      <div class="card bg-base-200 shadow-sm overflow-y-auto max-h-[500px]">
        <ul class="menu menu-sm p-0">
          {#each chapters as c, i}
            <!-- svelte-ignore a11y-click-events-have-key-events -->
            <!-- svelte-ignore a11y-no-static-element-interactions -->
            <li>
              <button class="flex gap-2 {$selectedChapter === i ? 'active' : ''}" on:click={() => selectChapter(i)}>
                <span class="w-6 text-center">{statusIcons[c.status] || ''}</span>
                <span class="text-base-content/50 w-6">{c.num}</span>
                <span class="flex-1 text-left truncate text-sm">{c.title}</span>
              </button>
            </li>
          {/each}
        </ul>
      </div>

      <!-- Content area -->
      <div class="space-y-3">
        {#if ch}
          <div class="card bg-base-200 shadow-sm">
            <div class="card-body p-4">
              <h2 class="card-title text-base">第 {ch.num} 章: {ch.title}</h2>

              {#if ch.summary}
                <div class="bg-base-300 rounded p-2 text-sm text-base-content/70">{ch.summary}</div>
              {/if}

              {#if displayContent}
                <div class="bg-base-300 rounded p-3 text-sm chapter-content max-h-[500px] overflow-y-auto">{displayContent}</div>
              {/if}

              <div class="flex gap-2 flex-wrap mt-2">
                {#if ch.status === 'pending' && isCurrent}
                  <button class="btn btn-primary btn-sm" on:click={doGenerate} disabled={$taskRunning}>生成本章</button>
                {/if}
                {#if ch.status === 'review'}
                  <button class="btn btn-success btn-sm" on:click={doConfirm} disabled={$taskRunning}>确认本章</button>
                {/if}
              </div>
            </div>
          </div>
        {:else}
          <div class="text-center py-16 text-base-content/50 text-base">选择一个章节查看</div>
        {/if}
      </div>
    </div>
  </div>
{/if}
