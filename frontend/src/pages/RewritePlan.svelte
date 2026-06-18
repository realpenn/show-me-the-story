<script>
  import { onMount } from 'svelte';
  import { api } from '../lib/api.js';
  import { addToast, progress, rewriteState, taskRunning } from '../lib/stores.js';
  import { t } from '../lib/i18n/index.js';

  let loading = false;
  let editingJSON = false;
  let jsonText = '';
  let savingFullTextNum = 0;

  $: rewrite = $rewriteState || {};
  $: requests = rewrite.requests || [];
  $: plan = rewrite.plan || {};
  $: chapters = plan.chapters || [];
  $: mappings = plan.mappings || [];
  $: impacts = plan.request_impacts || [];

  onMount(loadRewrite);

  async function loadRewrite() {
    loading = true;
    try {
      rewriteState.set(await api('GET', '/api/rewrite'));
    } catch (e) {
      addToast(e.message, 'error');
    } finally {
      loading = false;
    }
  }

  async function generatePlan() {
    if (requests.length === 0) {
      addToast($t('rewritePlan.toast.needRequests'), 'error');
      return;
    }
    try {
      await api('POST', '/api/rewrite/plan/generate');
      addToast($t('rewritePlan.toast.generateStarted'), 'info');
    } catch (e) {
      addToast(e.message, 'error');
    }
  }

  function startJSONEdit() {
    jsonText = JSON.stringify(plan || {}, null, 2);
    editingJSON = true;
  }

  async function saveJSONEdit() {
    try {
      const parsed = JSON.parse(jsonText);
      const saved = await api('PUT', '/api/rewrite/plan', parsed);
      rewriteState.update(r => ({ ...(r || {}), plan: saved }));
      editingJSON = false;
      addToast($t('rewritePlan.toast.saved'), 'success');
    } catch (e) {
      addToast(e.message, 'error');
    }
  }

  async function confirmPlan() {
    try {
      const data = await api('POST', '/api/rewrite/plan/confirm');
      rewriteState.set(data);
      progress.set(await api('GET', '/api/progress'));
      addToast($t('rewritePlan.toast.confirmed'), 'success');
    } catch (e) {
      addToast(e.message, 'error');
    }
  }

  function mappingLabel(m) {
    return `${m.target_chapter_num} ← ${(m.source_chapter_nums || []).join(', ')}`;
  }

  async function saveChapterFullText(ch) {
    if (ch.use_original_full_text && !String(ch.full_text_reason || '').trim()) {
      addToast($t('rewritePlan.fullText.needReason'), 'error');
      return;
    }
    savingFullTextNum = ch.num;
    try {
      const next = JSON.parse(JSON.stringify(plan || {}));
      const target = (next.chapters || []).find(item => item.num === ch.num);
      if (!target) throw new Error($t('rewritePlan.fullText.notFound'));
      target.use_original_full_text = !!ch.use_original_full_text;
      target.full_text_reason = target.use_original_full_text ? String(ch.full_text_reason || '').trim() : '';
      const saved = await api('PUT', '/api/rewrite/plan', next);
      rewriteState.update(r => ({ ...(r || {}), plan: saved }));
      addToast($t('rewritePlan.fullText.saved'), 'success');
    } catch (e) {
      addToast(e.message, 'error');
    } finally {
      savingFullTextNum = 0;
    }
  }
</script>

<div class="space-y-3">
  <div class="card bg-base-200 shadow-sm">
    <div class="card-body p-4 gap-3">
      <div class="flex items-center gap-2">
        <h2 class="card-title text-base flex-1">{$t('rewritePlan.title')}</h2>
        <span class="badge badge-sm badge-outline">{plan.status || 'draft'}</span>
        <button class="btn btn-ghost btn-xs" on:click={loadRewrite} disabled={$taskRunning || loading}>{$t('common.refresh')}</button>
        <button class="btn btn-primary btn-xs" on:click={generatePlan} disabled={$taskRunning || requests.length === 0}>{$t('rewritePlan.btn.generate')}</button>
        {#if chapters.length}
          <button class="btn btn-ghost btn-xs" on:click={startJSONEdit} disabled={$taskRunning}>{$t('rewritePlan.btn.editJSON')}</button>
          <button class="btn btn-success btn-xs" on:click={confirmPlan} disabled={$taskRunning}>{$t('rewritePlan.btn.confirm')}</button>
        {/if}
      </div>
      <div class="grid grid-cols-4 gap-2 text-sm">
        <div class="bg-base-300 rounded p-3">
          <div class="text-xs text-base-content/50">{$t('rewritePlan.stats.requests')}</div>
          <div class="font-semibold">{requests.length}</div>
        </div>
        <div class="bg-base-300 rounded p-3">
          <div class="text-xs text-base-content/50">{$t('rewritePlan.stats.chapters')}</div>
          <div class="font-semibold">{chapters.length}</div>
        </div>
        <div class="bg-base-300 rounded p-3">
          <div class="text-xs text-base-content/50">{$t('rewritePlan.stats.mappings')}</div>
          <div class="font-semibold">{mappings.length}</div>
        </div>
        <div class="bg-base-300 rounded p-3">
          <div class="text-xs text-base-content/50">{$t('rewritePlan.stats.impacts')}</div>
          <div class="font-semibold">{impacts.length}</div>
        </div>
      </div>
    </div>
  </div>

  {#if editingJSON}
    <div class="card bg-base-200 shadow-sm border border-primary/30">
      <div class="card-body p-4 gap-3">
        <div class="flex items-center gap-2">
          <h3 class="card-title text-base flex-1">{$t('rewritePlan.json.title')}</h3>
          <button class="btn btn-ghost btn-xs" on:click={() => editingJSON = false}>{$t('common.cancel')}</button>
          <button class="btn btn-success btn-xs" on:click={saveJSONEdit} disabled={$taskRunning}>{$t('common.save')}</button>
        </div>
        <textarea class="textarea textarea-sm w-full h-[56vh] font-mono text-xs" bind:value={jsonText} disabled={$taskRunning}></textarea>
      </div>
    </div>
  {/if}

  {#if !chapters.length && !editingJSON}
    <div class="card bg-base-200 shadow-sm">
      <div class="card-body p-8 text-center text-base-content/50">
        <div class="text-4xl mb-2">🧱</div>
        <p>{$t('rewritePlan.empty')}</p>
      </div>
    </div>
  {/if}

  {#if chapters.length && !editingJSON}
    <div class="card bg-base-200 shadow-sm">
      <div class="card-body p-4 gap-3">
        <h3 class="card-title text-base">{plan.title || $t('common.untitled')}</h3>
        <div class="grid grid-cols-2 gap-3 text-sm">
          <div>
            <div class="text-xs text-base-content/50 mb-1">{$t('rewritePlan.globalDirection')}</div>
            <p class="whitespace-pre-wrap">{plan.global_direction}</p>
          </div>
          <div>
            <div class="text-xs text-base-content/50 mb-1">{$t('rewritePlan.corePremise')}</div>
            <p class="whitespace-pre-wrap">{plan.core_premise}</p>
          </div>
        </div>
        {#if plan.style_guide}
          <div>
            <div class="text-xs text-base-content/50 mb-1">{$t('rewritePlan.styleGuide')}</div>
            <p class="text-sm whitespace-pre-wrap">{plan.style_guide}</p>
          </div>
        {/if}
      </div>
    </div>

    <div class="grid grid-cols-2 gap-3">
      <div class="card bg-base-200 shadow-sm">
        <div class="card-body p-4 gap-2">
          <h3 class="card-title text-base">{$t('rewritePlan.impacts.title')}</h3>
          <div class="space-y-2 max-h-72 overflow-y-auto">
            {#each impacts as impact}
              <div class="bg-base-300 rounded p-2 text-xs">
                <div class="font-medium">{impact.request_id}</div>
                <div>{impact.summary}</div>
                {#if impact.affected_chapters?.length}
                  <div class="text-base-content/50">{$t('rewritePlan.impacts.chapters')}: {impact.affected_chapters.join(', ')}</div>
                {/if}
              </div>
            {/each}
          </div>
        </div>
      </div>

      <div class="card bg-base-200 shadow-sm">
        <div class="card-body p-4 gap-2">
          <h3 class="card-title text-base">{$t('rewritePlan.mappings.title')}</h3>
          <div class="flex flex-wrap gap-2 max-h-72 overflow-y-auto">
            {#each mappings as m}
              <span class="badge badge-outline gap-1">{mappingLabel(m)} <span class="opacity-60">{m.mapping_type}</span></span>
            {/each}
          </div>
        </div>
      </div>
    </div>

    <div class="card bg-base-200 shadow-sm">
      <div class="card-body p-4 gap-2">
        <h3 class="card-title text-base">{$t('rewritePlan.chapters.title')}</h3>
        <div class="space-y-2 max-h-[62vh] overflow-y-auto">
          {#each chapters as ch}
            <details class="bg-base-300 rounded-lg p-3">
              <summary class="cursor-pointer text-sm font-medium">
                # {ch.num} {ch.title}
                <span class="badge badge-ghost badge-xs ml-2">{(ch.source_chapter_nums || []).join(', ')}</span>
                <span class="badge badge-outline badge-xs">{ch.mapping_type}</span>
                {#if ch.use_original_full_text}
                  <span class="badge badge-warning badge-xs">{$t('rewritePlan.fullText.badge')}</span>
                {/if}
              </summary>
              <div class="mt-2 text-sm space-y-2">
                <p class="whitespace-pre-wrap">{ch.outline}</p>
                {#if ch.preserved_events?.length}
                  <div class="text-xs text-base-content/60">{$t('rewritePlan.chapters.preserved')}: {ch.preserved_events.join(' / ')}</div>
                {/if}
                {#if ch.changed_events?.length}
                  <div class="text-xs text-base-content/60">{$t('rewritePlan.chapters.changed')}: {ch.changed_events.join(' / ')}</div>
                {/if}
                {#if ch.forbidden_close_points?.length}
                  <div class="text-xs text-warning">{$t('rewritePlan.chapters.forbidden')}: {ch.forbidden_close_points.join(' / ')}</div>
                {/if}
                <div class="bg-base-200 rounded p-2 space-y-2">
                  <label class="flex items-center gap-2 cursor-pointer text-xs">
                    <input
                      type="checkbox"
                      class="toggle toggle-warning toggle-xs"
                      bind:checked={ch.use_original_full_text}
                      disabled={$taskRunning || savingFullTextNum === ch.num}
                    />
                    <span>{$t('rewritePlan.fullText.toggle')}</span>
                  </label>
                  {#if ch.use_original_full_text}
                    <textarea
                      class="textarea textarea-sm w-full h-16 text-xs"
                      bind:value={ch.full_text_reason}
                      placeholder={$t('rewritePlan.fullText.reason')}
                      disabled={$taskRunning || savingFullTextNum === ch.num}
                    ></textarea>
                    <div class="flex items-center gap-2">
                      <span class="text-xs text-warning flex-1">{$t('rewritePlan.fullText.strict')}</span>
                      <button class="btn btn-warning btn-xs" on:click={() => saveChapterFullText(ch)} disabled={$taskRunning || savingFullTextNum === ch.num || !String(ch.full_text_reason || '').trim()}>
                        {savingFullTextNum === ch.num ? $t('common.saving') : $t('common.save')}
                      </button>
                    </div>
                  {:else}
                    <div class="flex justify-end">
                      <button class="btn btn-ghost btn-xs" on:click={() => saveChapterFullText(ch)} disabled={$taskRunning || savingFullTextNum === ch.num}>{savingFullTextNum === ch.num ? $t('common.saving') : $t('common.save')}</button>
                    </div>
                  {/if}
                </div>
              </div>
            </details>
          {/each}
        </div>
      </div>
    </div>
  {/if}
</div>
