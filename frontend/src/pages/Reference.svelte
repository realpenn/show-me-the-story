<script>
  import { onMount } from 'svelte';
  import { api } from '../lib/api.js';
  import { addToast, referenceState, settings, taskRunning } from '../lib/stores.js';
  import { t } from '../lib/i18n/index.js';

  let sourceName = '';
  let importContent = '';
  let loading = false;
  let editing = false;
  let editableChapters = [];

  $: ref = $referenceState;
  $: book = ref?.book || {};
  $: analysis = ref?.analysis || {};
  $: chapters = book?.chapters || [];
  $: analysisChapters = analysis?.chapters || [];
  $: settingsCandidate = analysis?.settings || {};
  $: candidateCount =
    (settingsCandidate.characters?.length || 0) +
    (settingsCandidate.worldview?.length || 0) +
    (settingsCandidate.organizations?.length || 0) +
    (settingsCandidate.relations?.length || 0);

  onMount(loadReference);

  async function loadReference(includeContent = false) {
    loading = true;
    try {
      const data = await api('GET', '/api/reference' + (includeContent ? '?include_content=1' : ''));
      referenceState.set(data);
      if (includeContent) {
        editableChapters = (data.book?.chapters || []).map(ch => ({
          num: ch.num,
          title: ch.title || '',
          content: ch.content || '',
        }));
      }
    } catch (e) {
      addToast(e.message, 'error');
    } finally {
      loading = false;
    }
  }

  async function handleFile(e) {
    const file = e.target.files?.[0];
    if (!file) return;
    sourceName = file.name;
    importContent = await file.text();
  }

  async function importReference() {
    const content = importContent.trim();
    if (!content) {
      addToast($t('reference.toast.needContent'), 'error');
      return;
    }
    try {
      await api('POST', '/api/reference/import', { content, source_name: sourceName });
      addToast($t('reference.toast.importStarted'), 'info');
      importContent = '';
    } catch (e) {
      addToast(e.message, 'error');
    }
  }

  async function startEditing() {
    editing = true;
    await loadReference(true);
  }

  function cancelEditing() {
    editing = false;
    editableChapters = [];
  }

  function addChapterAfter(idx) {
    editableChapters.splice(idx + 1, 0, {
      num: idx + 2,
      title: $t('reference.chapter.newTitle', { num: idx + 2 }),
      content: '',
    });
    editableChapters = editableChapters.map((ch, i) => ({ ...ch, num: i + 1 }));
  }

  function removeChapter(idx) {
    editableChapters.splice(idx, 1);
    editableChapters = editableChapters.map((ch, i) => ({ ...ch, num: i + 1 }));
  }

  async function saveChapters() {
    if (editableChapters.some(ch => !ch.title.trim() || !ch.content.trim())) {
      addToast($t('reference.toast.chapterRequired'), 'error');
      return;
    }
    try {
      const data = await api('PUT', '/api/reference/chapters', { chapters: editableChapters });
      referenceState.set(data);
      editing = false;
      editableChapters = [];
      addToast($t('reference.toast.chaptersSaved'), 'success');
    } catch (e) {
      addToast(e.message, 'error');
    }
  }

  async function analyzeReference() {
    if (!chapters.length) {
      addToast($t('reference.toast.needImport'), 'error');
      return;
    }
    try {
      await api('POST', '/api/reference/analyze');
      addToast($t('reference.toast.analyzeStarted'), 'info');
    } catch (e) {
      addToast(e.message, 'error');
    }
  }

  async function importSettings() {
    try {
      const data = await api('POST', '/api/reference/settings/import');
      settings.set(data.settings);
      referenceState.update(r => r ? { ...r, analysis: data.analysis } : r);
      addToast($t('reference.toast.settingsImported', { n: data.imported }), 'success');
    } catch (e) {
      addToast(e.message, 'error');
    }
  }

  function statusLabel(status) {
    if (status === 'auto_applied') return $t('reference.settings.status.auto');
    if (status === 'preview_required') return $t('reference.settings.status.preview');
    if (status === 'applied') return $t('reference.settings.status.applied');
    return $t('reference.settings.status.none');
  }
</script>

<div class="space-y-3">
  <div class="card bg-base-200 shadow-sm">
    <div class="card-body p-4 gap-3">
      <div class="flex items-center gap-2">
        <h2 class="card-title text-base flex-1">{$t('reference.title')}</h2>
        {#if chapters.length}
          <button class="btn btn-ghost btn-xs" on:click={() => loadReference(false)} disabled={$taskRunning || loading}>{$t('common.refresh')}</button>
          <button class="btn btn-ghost btn-xs" on:click={startEditing} disabled={$taskRunning || loading}>{$t('reference.btn.editChapters')}</button>
          <button class="btn btn-primary btn-xs" on:click={analyzeReference} disabled={$taskRunning || loading}>{$t('reference.btn.analyze')}</button>
        {/if}
      </div>

      <div class="grid grid-cols-3 gap-2 text-sm">
        <div class="bg-base-300 rounded p-3">
          <div class="text-xs text-base-content/50">{$t('reference.stats.chapters')}</div>
          <div class="font-semibold">{chapters.length}</div>
        </div>
        <div class="bg-base-300 rounded p-3">
          <div class="text-xs text-base-content/50">{$t('reference.stats.runes')}</div>
          <div class="font-semibold">{book.total_runes || 0}</div>
        </div>
        <div class="bg-base-300 rounded p-3">
          <div class="text-xs text-base-content/50">{$t('reference.stats.analysis')}</div>
          <div class="font-semibold">{analysisChapters.length ? $t('common.yes') : $t('common.no')}</div>
        </div>
      </div>
    </div>
  </div>

  {#if !chapters.length}
    <div class="card bg-base-200 shadow-sm">
      <div class="card-body p-4 gap-3">
        <h3 class="card-title text-base">{$t('reference.import.title')}</h3>
        <p class="text-xs text-base-content/50">{$t('reference.import.hint')}</p>
        <div class="flex gap-2 flex-wrap">
          <input class="file-input file-input-sm flex-1 min-w-52" type="file" accept=".txt,text/plain" on:change={handleFile} disabled={$taskRunning} />
          <input class="input input-sm flex-1 min-w-52" bind:value={sourceName} placeholder={$t('reference.import.sourceName')} disabled={$taskRunning} />
        </div>
        <textarea class="textarea w-full h-64 text-sm font-serif" bind:value={importContent} placeholder={$t('reference.import.placeholder')} disabled={$taskRunning}></textarea>
        {#if importContent.length > 800000}
          <p class="text-xs text-warning">{$t('reference.import.largeHint')}</p>
        {/if}
        <div class="flex justify-end">
          <button class="btn btn-primary btn-sm" on:click={importReference} disabled={$taskRunning || !importContent.trim()}>{$t('reference.import.submit')}</button>
        </div>
      </div>
    </div>
  {/if}

  {#if chapters.length && !editing}
    <div class="card bg-base-200 shadow-sm">
      <div class="card-body p-4 gap-2">
        <div class="flex items-center gap-2">
          <h3 class="card-title text-base flex-1">{$t('reference.chapters.title')}</h3>
          <span class="badge badge-sm badge-outline">{book.title || $t('common.untitled')}</span>
        </div>
        <div class="max-h-80 overflow-y-auto divide-y divide-base-content/10">
          {#each chapters as ch}
            <div class="py-2 flex items-start gap-3">
              <span class="badge badge-ghost badge-sm shrink-0">#{ch.num}</span>
              <div class="min-w-0 flex-1">
                <div class="text-sm font-medium truncate">{ch.title}</div>
                <div class="text-xs text-base-content/45">{ch.rune_count || 0} {$t('reference.chapters.runes')}</div>
              </div>
            </div>
          {/each}
        </div>
      </div>
    </div>
  {/if}

  {#if editing}
    <div class="card bg-base-200 shadow-sm border border-primary/30">
      <div class="card-body p-4 gap-3">
        <div class="flex items-center gap-2">
          <h3 class="card-title text-base flex-1">{$t('reference.edit.title')}</h3>
          <button class="btn btn-ghost btn-xs" on:click={cancelEditing}>{$t('common.cancel')}</button>
          <button class="btn btn-success btn-xs" on:click={saveChapters} disabled={$taskRunning}>{$t('common.save')}</button>
        </div>
        <p class="text-xs text-base-content/50">{$t('reference.edit.hint')}</p>
        <div class="space-y-3 max-h-[62vh] overflow-y-auto pr-1">
          {#each editableChapters as ch, i}
            <div class="bg-base-300 rounded-lg p-3 space-y-2">
              <div class="flex items-center gap-2">
                <span class="badge badge-primary badge-sm">#{ch.num}</span>
                <input class="input input-sm flex-1" bind:value={ch.title} disabled={$taskRunning} />
                <button class="btn btn-ghost btn-xs" on:click={() => addChapterAfter(i)} disabled={$taskRunning}>{$t('reference.edit.split')}</button>
                <button class="btn btn-ghost btn-xs text-error" on:click={() => removeChapter(i)} disabled={$taskRunning || editableChapters.length <= 1}>{$t('common.delete')}</button>
              </div>
              <textarea class="textarea textarea-sm w-full h-40 font-serif text-sm" bind:value={ch.content} disabled={$taskRunning}></textarea>
            </div>
          {/each}
        </div>
      </div>
    </div>
  {/if}

  {#if analysisChapters.length}
    <div class="card bg-base-200 shadow-sm">
      <div class="card-body p-4 gap-3">
        <div class="flex items-center gap-2">
          <h3 class="card-title text-base flex-1">{$t('reference.analysis.title')}</h3>
          <span class="badge badge-sm">{analysis.mode || 'per_chapter'}</span>
        </div>
        <div class="grid grid-cols-2 gap-2 text-sm">
          <div>
            <div class="text-xs text-base-content/50">{$t('reference.analysis.storyType')}</div>
            <div>{analysis.story_type || $t('common.empty')}</div>
          </div>
          <div>
            <div class="text-xs text-base-content/50">{$t('reference.analysis.style')}</div>
            <div>{analysis.writing_style || $t('common.empty')}</div>
          </div>
        </div>
        <div>
          <div class="text-xs text-base-content/50 mb-1">{$t('reference.analysis.synopsis')}</div>
          <p class="text-sm whitespace-pre-wrap">{analysis.synopsis}</p>
        </div>
        <div>
          <div class="text-xs text-base-content/50 mb-1">{$t('reference.analysis.coreSetting')}</div>
          <p class="text-sm whitespace-pre-wrap">{analysis.core_setting}</p>
        </div>
      </div>
    </div>

    <div class="card bg-base-200 shadow-sm">
      <div class="card-body p-4 gap-2">
        <h3 class="card-title text-base">{$t('reference.analysis.chapters')}</h3>
        <div class="space-y-2 max-h-96 overflow-y-auto">
          {#each analysisChapters as ch}
            <details class="bg-base-300 rounded-lg p-3">
              <summary class="cursor-pointer text-sm font-medium"># {ch.num} {ch.title}</summary>
              <div class="mt-2 text-sm space-y-2">
                <p class="whitespace-pre-wrap">{ch.summary}</p>
                {#if ch.key_events?.length}
                  <div class="text-xs text-base-content/60">{$t('reference.analysis.keyEvents')}: {ch.key_events.join(' / ')}</div>
                {/if}
                {#if ch.scene_function}
                  <div class="text-xs text-base-content/60">{$t('reference.analysis.sceneFunction')}: {ch.scene_function}</div>
                {/if}
              </div>
            </details>
          {/each}
        </div>
      </div>
    </div>
  {/if}

  {#if candidateCount > 0}
    <div class="card bg-base-200 shadow-sm">
      <div class="card-body p-4 gap-3">
        <div class="flex items-center gap-2">
          <h3 class="card-title text-base flex-1">{$t('reference.settings.title')}</h3>
          <span class="badge badge-sm badge-outline">{statusLabel(analysis.settings_import_status)}</span>
          {#if analysis.settings_import_status === 'preview_required'}
            <button class="btn btn-success btn-xs" on:click={importSettings} disabled={$taskRunning}>{$t('reference.settings.import')}</button>
          {/if}
        </div>
        <div class="grid grid-cols-4 gap-2 text-xs">
          <div class="bg-base-300 rounded p-2">{$t('reference.settings.characters')}: {settingsCandidate.characters?.length || 0}</div>
          <div class="bg-base-300 rounded p-2">{$t('reference.settings.worldview')}: {settingsCandidate.worldview?.length || 0}</div>
          <div class="bg-base-300 rounded p-2">{$t('reference.settings.organizations')}: {settingsCandidate.organizations?.length || 0}</div>
          <div class="bg-base-300 rounded p-2">{$t('reference.settings.relations')}: {settingsCandidate.relations?.length || 0}</div>
        </div>
        <div class="max-h-48 overflow-y-auto text-xs space-y-1">
          {#each (settingsCandidate.characters || []) as c}
            <div class="bg-base-300 rounded p-2"><span class="font-medium">{c.name}</span> {c.personality || c.notes || ''}</div>
          {/each}
          {#each (settingsCandidate.worldview || []) as w}
            <div class="bg-base-300 rounded p-2"><span class="font-medium">{w.name}</span> {w.description}</div>
          {/each}
        </div>
      </div>
    </div>
  {/if}
</div>
