<script>
  import { onMount } from 'svelte';
  import { api } from '../lib/api.js';
  import { addToast, rewriteState, taskRunning } from '../lib/stores.js';
  import { t } from '../lib/i18n/index.js';

  const types = ['global', 'chapter', 'range', 'character', 'setting', 'relationship', 'ending', 'forbidden'];
  const priorities = ['P0', 'P1', 'P2'];
  const intensities = ['light', 'medium', 'heavy'];

  let loading = false;
  let editingId = '';
  let form = emptyForm();

  $: requests = $rewriteState?.requests || [];
  $: plan = $rewriteState?.plan || {};

  onMount(loadRewrite);

  function emptyForm() {
    return {
      type: 'global',
      scope: '',
      chapter_num: '',
      chapter_start: '',
      chapter_end: '',
      object: '',
      instruction: '',
      intensity: 'medium',
      affects_following: true,
      priority: 'P1',
    };
  }

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

  function editRequest(req) {
    editingId = req.id;
    form = {
      type: req.type || 'global',
      scope: req.scope || '',
      chapter_num: req.chapter_num || '',
      chapter_start: req.chapter_start || '',
      chapter_end: req.chapter_end || '',
      object: req.object || '',
      instruction: req.instruction || '',
      intensity: req.intensity || 'medium',
      affects_following: req.affects_following !== false,
      priority: req.priority || 'P1',
    };
  }

  function cancelEdit() {
    editingId = '';
    form = emptyForm();
  }

  function payload() {
    return {
      ...form,
      chapter_num: Number(form.chapter_num) || 0,
      chapter_start: Number(form.chapter_start) || 0,
      chapter_end: Number(form.chapter_end) || 0,
    };
  }

  async function saveRequest() {
    if (!form.instruction.trim()) {
      addToast($t('rewriteRequests.toast.needInstruction'), 'error');
      return;
    }
    try {
      if (editingId) {
        await api('PUT', '/api/rewrite/requests/' + editingId, payload());
        addToast($t('rewriteRequests.toast.updated'), 'success');
      } else {
        await api('POST', '/api/rewrite/requests', payload());
        addToast($t('rewriteRequests.toast.created'), 'success');
      }
      cancelEdit();
      await loadRewrite();
    } catch (e) {
      addToast(e.message, 'error');
    }
  }

  async function deleteRequest(id) {
    try {
      await api('DELETE', '/api/rewrite/requests/' + id);
      addToast($t('rewriteRequests.toast.deleted'), 'success');
      await loadRewrite();
    } catch (e) {
      addToast(e.message, 'error');
    }
  }

  function scopeLabel(req) {
    if (req.type === 'chapter') return $t('rewriteRequests.scope.chapter', { n: req.chapter_num || req.chapter_start });
    if (req.type === 'range') return $t('rewriteRequests.scope.range', { a: req.chapter_start, b: req.chapter_end });
    return req.object || req.scope || $t('rewriteRequests.scope.global');
  }
</script>

<div class="space-y-3">
  <div class="card bg-base-200 shadow-sm">
    <div class="card-body p-4 gap-3">
      <div class="flex items-center gap-2">
        <h2 class="card-title text-base flex-1">{$t('rewriteRequests.title')}</h2>
        <button class="btn btn-ghost btn-xs" on:click={loadRewrite} disabled={$taskRunning || loading}>{$t('common.refresh')}</button>
      </div>
      <div class="grid grid-cols-4 gap-2 text-sm">
        <div class="bg-base-300 rounded p-3">
          <div class="text-xs text-base-content/50">{$t('rewriteRequests.stats.count')}</div>
          <div class="font-semibold">{requests.length}</div>
        </div>
        <div class="bg-base-300 rounded p-3">
          <div class="text-xs text-base-content/50">{$t('rewriteRequests.stats.plan')}</div>
          <div class="font-semibold">{plan.status || 'draft'}</div>
        </div>
        <div class="bg-base-300 rounded p-3 col-span-2">
          <div class="text-xs text-base-content/50">{$t('rewriteRequests.stats.hint')}</div>
          <div class="text-xs text-base-content/70">{$t('rewriteRequests.hint')}</div>
        </div>
      </div>
    </div>
  </div>

  <div class="card bg-base-200 shadow-sm">
    <div class="card-body p-4 gap-3">
      <h3 class="card-title text-base">{editingId ? $t('rewriteRequests.form.edit') : $t('rewriteRequests.form.create')}</h3>
      <div class="grid grid-cols-4 gap-2">
        <select class="select select-sm" bind:value={form.type} disabled={$taskRunning}>
          {#each types as type}
            <option value={type}>{$t('rewriteRequests.type.' + type)}</option>
          {/each}
        </select>
        <select class="select select-sm" bind:value={form.intensity} disabled={$taskRunning}>
          {#each intensities as intensity}
            <option value={intensity}>{$t('rewriteRequests.intensity.' + intensity)}</option>
          {/each}
        </select>
        <select class="select select-sm" bind:value={form.priority} disabled={$taskRunning}>
          {#each priorities as p}
            <option value={p}>{p}</option>
          {/each}
        </select>
        <label class="label cursor-pointer justify-start gap-2 bg-base-300 rounded px-3 py-0 min-h-8">
          <input type="checkbox" class="toggle toggle-primary toggle-xs" bind:checked={form.affects_following} disabled={$taskRunning} />
          <span class="label-text text-xs">{$t('rewriteRequests.form.affectsFollowing')}</span>
        </label>
      </div>
      <div class="grid grid-cols-4 gap-2">
        <input class="input input-sm" type="number" min="1" bind:value={form.chapter_num} placeholder={$t('rewriteRequests.form.chapter')} disabled={$taskRunning || !['chapter'].includes(form.type)} />
        <input class="input input-sm" type="number" min="1" bind:value={form.chapter_start} placeholder={$t('rewriteRequests.form.chapterStart')} disabled={$taskRunning || form.type !== 'range'} />
        <input class="input input-sm" type="number" min="1" bind:value={form.chapter_end} placeholder={$t('rewriteRequests.form.chapterEnd')} disabled={$taskRunning || form.type !== 'range'} />
        <input class="input input-sm" bind:value={form.object} placeholder={$t('rewriteRequests.form.object')} disabled={$taskRunning || ['global', 'chapter', 'range', 'ending', 'forbidden'].includes(form.type)} />
      </div>
      <textarea class="textarea textarea-sm h-28 text-sm" bind:value={form.instruction} placeholder={$t('rewriteRequests.form.instruction')} disabled={$taskRunning}></textarea>
      <div class="flex justify-end gap-2">
        {#if editingId}
          <button class="btn btn-ghost btn-xs" on:click={cancelEdit}>{$t('common.cancel')}</button>
        {/if}
        <button class="btn btn-primary btn-xs" on:click={saveRequest} disabled={$taskRunning || !form.instruction.trim()}>{editingId ? $t('common.save') : $t('rewriteRequests.form.add')}</button>
      </div>
    </div>
  </div>

  <div class="card bg-base-200 shadow-sm">
    <div class="card-body p-4 gap-2">
      <h3 class="card-title text-base">{$t('rewriteRequests.list.title')}</h3>
      {#if requests.length === 0}
        <p class="text-sm text-base-content/45 text-center py-8">{$t('rewriteRequests.empty')}</p>
      {:else}
        <div class="space-y-2">
          {#each requests as req}
            <div class="bg-base-300 rounded-lg p-3">
              <div class="flex items-center gap-2 mb-1">
                <span class="badge badge-info badge-sm">{$t('rewriteRequests.type.' + req.type)}</span>
                <span class="badge badge-ghost badge-sm">{scopeLabel(req)}</span>
                <span class="badge badge-outline badge-sm">{req.priority || 'P1'}</span>
                <span class="badge badge-outline badge-sm">{$t('rewriteRequests.intensity.' + (req.intensity || 'medium'))}</span>
                <span class="flex-1"></span>
                <button class="btn btn-ghost btn-xs" on:click={() => editRequest(req)} disabled={$taskRunning}>{$t('common.edit')}</button>
                <button class="btn btn-ghost btn-xs text-error" on:click={() => deleteRequest(req.id)} disabled={$taskRunning}>{$t('common.delete')}</button>
              </div>
              <p class="text-sm whitespace-pre-wrap">{req.instruction}</p>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  </div>
</div>
