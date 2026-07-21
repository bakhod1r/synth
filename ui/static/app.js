// Synth workbench. No framework, no build step, no external requests.
'use strict';

const state = {
  types: [],          // [{kind, category, localized, locales}]
  byKind: new Map(),
  fields: [],         // [{name, kind, params}]
  lang: 'en',
};

const el = (id) => document.getElementById(id);
const t = (key, ...args) => {
  const v = (I18N[state.lang] || I18N.en)[key];
  return typeof v === 'function' ? v(...args) : (v ?? key);
};

// Kinds that take numeric bounds, and the one that takes a list. Everything
// else has no options, and showing empty boxes for them would be noise.
const NUMERIC_KINDS = new Set([
  'int', 'float', 'amount', 'salary', 'percentage', 'rating',
  'temperature', 'year', 'port', 'latitude', 'longitude',
]);

// ---------------------------------------------------------------- boot

async function boot() {
  state.lang = localStorage.getItem('synth.lang') || navigatorLang();
  el('lang').value = state.lang;

  const [types, locales] = await Promise.all([
    fetch('/api/types').then((r) => r.json()),
    fetch('/api/locales').then((r) => r.json()),
  ]);
  state.types = types;
  for (const ty of types) state.byKind.set(ty.kind, ty);

  const sel = el('locale');
  for (const name of locales) {
    const opt = document.createElement('option');
    opt.value = opt.textContent = name;
    if (name === 'en_US') opt.selected = true;
    sel.appendChild(opt);
  }

  applyLanguage();
  renderTypes('');

  addField('name');
  addField('email');

  el('lang').addEventListener('change', (e) => {
    state.lang = e.target.value;
    localStorage.setItem('synth.lang', state.lang);
    applyLanguage();
    renderTypes(el('search').value);
    renderFields();
  });
  el('search').addEventListener('input', (e) => renderTypes(e.target.value));
  el('add').addEventListener('click', () => addField('name'));
  el('reseed').addEventListener('click', () => {
    el('seed').value = Math.floor(Math.random() * 1e6);
    schedulePreview();
  });
  for (const id of ['name', 'count', 'locale', 'seed']) {
    el(id).addEventListener('input', schedulePreview);
  }
  el('download').addEventListener('click', download);
  el('copyYaml').addEventListener('click', copyYaml);
}

function navigatorLang() {
  const code = (navigator.language || 'en').slice(0, 2);
  return I18N[code] ? code : 'en';
}

// applyLanguage rewrites every marked element. Attributes are set separately
// so placeholders and tooltips are translated too, not just visible text.
function applyLanguage() {
  document.documentElement.lang = state.lang;
  for (const node of document.querySelectorAll('[data-i18n]')) {
    node.textContent = t(node.dataset.i18n);
  }
  for (const node of document.querySelectorAll('[data-i18n-placeholder]')) {
    node.placeholder = t(node.dataset.i18nPlaceholder);
  }
  for (const node of document.querySelectorAll('[data-i18n-title]')) {
    node.title = t(node.dataset.i18nTitle);
  }
}

// ---------------------------------------------------------------- palette

function categoryName(key) {
  const names = CATEGORY_NAMES[state.lang] || CATEGORY_NAMES.en;
  return names[key] || key;
}

function groupByCategory(types) {
  const groups = new Map();
  for (const ty of types) {
    if (!groups.has(ty.category)) groups.set(ty.category, []);
    groups.get(ty.category).push(ty);
  }
  return [...groups].sort((a, b) => categoryName(a[0]).localeCompare(categoryName(b[0])));
}

function renderTypes(query) {
  const q = query.trim().toLowerCase();
  const matching = state.types.filter((ty) => !q || ty.kind.includes(q));

  const box = el('types');
  box.textContent = '';
  for (const [category, items] of groupByCategory(matching)) {
    const h = document.createElement('div');
    h.className = 'group-name';
    h.textContent = `${categoryName(category)} (${items.length})`;
    box.appendChild(h);

    for (const ty of items) {
      const b = document.createElement('button');
      b.type = 'button';
      b.className = ty.localized ? 'type localized' : 'type';
      b.title = ty.locales && ty.locales.length
        ? t('localizedIn', ty.locales.length, ty.locales.join(', '))
        : ty.localized ? t('legendLocalized') : t('legendGlobal');

      const dot = document.createElement('span');
      dot.className = 'dot';
      b.appendChild(dot);
      b.appendChild(document.createTextNode(ty.kind));
      b.addEventListener('click', () => addField(ty.kind));
      box.appendChild(b);
    }
  }
}

// ---------------------------------------------------------------- schema

function addField(kind) {
  state.fields.push({ name: uniqueName(kind), kind, params: {} });
  renderFields();
  schedulePreview();
  // Keep the newly added row in view; with a long schema it would otherwise
  // appear below the fold and look like nothing happened.
  const rows = el('fields').querySelectorAll('tbody tr');
  rows[rows.length - 1]?.scrollIntoView({ block: 'nearest' });
}

// uniqueName avoids a duplicate column, which would silently drop data.
function uniqueName(base) {
  const taken = new Set(state.fields.map((f) => f.name));
  if (!taken.has(base)) return base;
  for (let i = 2; ; i++) {
    const candidate = `${base}_${i}`;
    if (!taken.has(candidate)) return candidate;
  }
}

// kindSelect builds the type dropdown, grouped like the palette, so a column's
// type can be changed after the fact instead of being fixed at creation.
function kindSelect(field, index) {
  const sel = document.createElement('select');
  sel.setAttribute('aria-label', t('colType'));
  for (const [category, items] of groupByCategory(state.types)) {
    const group = document.createElement('optgroup');
    group.label = categoryName(category);
    for (const ty of items) {
      const opt = document.createElement('option');
      opt.value = opt.textContent = ty.kind;
      if (ty.kind === field.kind) opt.selected = true;
      group.appendChild(opt);
    }
    sel.appendChild(group);
  }
  sel.addEventListener('change', () => {
    state.fields[index].kind = sel.value;
    state.fields[index].params = {};   // old bounds rarely fit a new type
    renderFields();
    schedulePreview();
  });
  return sel;
}

function optionInputs(field, index) {
  const box = document.createElement('div');
  box.className = 'opts';

  const bind = (key, placeholder, type) => {
    const input = document.createElement('input');
    input.type = type;
    input.placeholder = placeholder;
    input.setAttribute('aria-label', `${field.name} ${placeholder}`);
    if (field.params[key] !== undefined) input.value = field.params[key];
    input.addEventListener('input', () => {
      const raw = input.value.trim();
      if (raw === '') delete state.fields[index].params[key];
      else state.fields[index].params[key] = raw;
      schedulePreview();
    });
    box.appendChild(input);
  };

  if (NUMERIC_KINDS.has(field.kind)) {
    bind('min', t('min'), 'number');
    bind('max', t('max'), 'number');
  } else if (field.kind === 'enum') {
    bind('choices', t('choices'), 'text');
  }
  return box;
}

function iconButton(label, glyph, onClick) {
  const b = document.createElement('button');
  b.type = 'button';
  b.className = 'icon';
  b.title = label;
  b.setAttribute('aria-label', label);
  b.textContent = glyph;
  b.addEventListener('click', onClick);
  return b;
}

function move(index, delta) {
  const to = index + delta;
  if (to < 0 || to >= state.fields.length) return;
  const [f] = state.fields.splice(index, 1);
  state.fields.splice(to, 0, f);
  renderFields();
  schedulePreview();
}

function renderFields() {
  const body = el('fields').querySelector('tbody');
  body.textContent = '';

  for (const [i, f] of state.fields.entries()) {
    const tr = document.createElement('tr');

    const orderCell = document.createElement('td');
    orderCell.className = 'order';
    orderCell.append(
      iconButton(t('moveUp'), '↑', () => move(i, -1)),
      iconButton(t('moveDown'), '↓', () => move(i, 1)),
    );

    const nameCell = document.createElement('td');
    const input = document.createElement('input');
    input.value = f.name;
    input.setAttribute('aria-label', t('colName'));
    input.addEventListener('input', () => {
      state.fields[i].name = input.value;
      schedulePreview();
    });
    nameCell.appendChild(input);

    const kindCell = document.createElement('td');
    kindCell.appendChild(kindSelect(f, i));

    const optCell = document.createElement('td');
    optCell.appendChild(optionInputs(f, i));

    const removeCell = document.createElement('td');
    removeCell.appendChild(iconButton(t('remove'), '×', () => {
      state.fields.splice(i, 1);
      renderFields();
      schedulePreview();
    }));

    tr.append(orderCell, nameCell, kindCell, optCell, removeCell);
    body.appendChild(tr);
  }
  el('empty').hidden = state.fields.length > 0;
  el('fields').hidden = state.fields.length === 0;
}

function currentSpec() {
  const fields = Object.create(null);
  const order = [];
  for (const f of state.fields) {
    const name = f.name.trim();
    if (!name || fields[name]) continue;
    const def = { kind: f.kind };
    for (const [k, v] of Object.entries(f.params)) {
      if (k === 'choices') def.choices = v.split(',').map((s) => s.trim()).filter(Boolean);
      else def[k] = Number(v);
    }
    fields[name] = def;
    order.push(name);
  }
  return {
    name: el('name').value || 'data',
    count: Number(el('count').value || 10),
    locale: el('locale').value,
    seed: Number(el('seed').value || 0),
    format: el('format').value,
    fields,
    order,
  };
}

// ---------------------------------------------------------------- preview

let pending;

function schedulePreview() {
  clearTimeout(pending);
  pending = setTimeout(preview, 150);
}

async function preview() {
  const spec = currentSpec();
  if (spec.order.length === 0) {
    el('preview').textContent = '';
    el('previewNote').textContent = '';
    showError(null);
    return;
  }
  try {
    const res = await fetch('/api/preview', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(spec),
    });
    const text = await res.text();
    if (!res.ok) {
      showError(text);
      return;
    }
    showError(null);
    const rows = JSON.parse(text);
    renderRows(rows, spec.order);
    el('previewNote').textContent = t('previewNote', rows.length);
  } catch (err) {
    showError(String(err));
  }
}

function renderRows(rows, order) {
  const table = document.createElement('table');
  table.className = 'rows';

  const head = document.createElement('tr');
  for (const c of order) {
    const th = document.createElement('th');
    th.textContent = c;
    head.appendChild(th);
  }
  table.appendChild(head);

  for (const row of rows) {
    const tr = document.createElement('tr');
    for (const c of order) {
      const td = document.createElement('td');
      const v = row[c];
      td.textContent = v === null || v === undefined ? '' : String(v);
      td.title = td.textContent;
      tr.appendChild(td);
    }
    table.appendChild(tr);
  }

  const box = el('preview');
  box.textContent = '';
  box.appendChild(table);
}

function showError(message) {
  const box = el('error');
  box.hidden = !message;
  box.textContent = message || '';
}

let toastTimer;
function toast(message) {
  const box = el('toast');
  box.textContent = message;
  box.hidden = false;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => { box.hidden = true; }, 2500);
}

// ---------------------------------------------------------------- output

// yamlOf renders the schema as a spec file, so what the workbench builds can
// be committed and run from the CLI. Teaching that path is the point.
function yamlOf(spec) {
  const lines = [
    `name: ${spec.name}`,
    `count: ${spec.count}`,
    `locale: ${spec.locale}`,
    `seed: ${spec.seed}`,
    'fields:',
  ];
  for (const name of spec.order) {
    const def = spec.fields[name];
    const parts = [`kind: ${def.kind}`];
    for (const [k, v] of Object.entries(def)) {
      if (k === 'kind') continue;
      parts.push(Array.isArray(v) ? `${k}: [${v.join(', ')}]` : `${k}: ${v}`);
    }
    lines.push(`  ${name}: { ${parts.join(', ')} }`);
  }
  return lines.join('\n') + '\n';
}

async function copyYaml() {
  const spec = currentSpec();
  if (spec.order.length === 0) {
    showError(t('addFieldFirst'));
    return;
  }
  const doc = yamlOf(spec);
  try {
    await navigator.clipboard.writeText(doc);
    toast(t('copied'));
  } catch {
    // Clipboard access can be refused; showing the document is still useful.
    showError(t('copyFailed'));
    const pre = document.createElement('pre');
    pre.className = 'yaml';
    pre.textContent = doc;
    const box = el('preview');
    box.textContent = '';
    box.appendChild(pre);
  }
}

async function download() {
  const spec = currentSpec();
  if (spec.order.length === 0) {
    showError(t('addFieldFirst'));
    return;
  }
  const res = await fetch('/api/generate', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(spec),
  });
  if (!res.ok) {
    showError(await res.text());
    return;
  }
  showError(null);

  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `${spec.name}.${spec.format}`;
  a.click();
  URL.revokeObjectURL(url);
}

boot();
