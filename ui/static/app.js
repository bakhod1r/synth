// Synth workbench. No framework, no build step, no external requests.
'use strict';

// Params the spec expects as numbers. Everything else is passed through as
// written, because "very-strong" is not a number and Number() would send NaN.
const NUMBER_PARAMS = new Set(['length', 'words', 'sentences', 'negative', 'iterations', 'blank']);

// Kinds whose min/max are dates rather than numbers, so the value must be sent
// as written instead of run through Number().
const DATE_BOUND_KINDS = new Set(['time', 'unixtime']);

const state = {
  types: [],          // [{kind, category, localized, locales}]
  byKind: new Map(),
  fields: [],         // [{name, kind, params, auto}]
  lang: 'en',
  view: 'table',      // 'table' | 'stacked'
  lastRows: [],
  lastOrder: [],
  // The generated file, held so Download saves exactly what Generate produced.
  // Regenerating on the download click would be wasted work and, worse, would
  // make it impossible to tell whether you are saving what you just looked at.
  generated: null,
  generating: false,
};

const el = (id) => document.getElementById(id);
// t falls back to English rather than to the key name. A screen showing
// "copySpecHint" is broken; a screen showing one English line among translated
// ones is merely incomplete, and still usable.
const t = (key, ...args) => {
  const dict = I18N[state.lang] || I18N.en;
  const v = dict[key] ?? I18N.en[key];
  return typeof v === 'function' ? v(...args) : (v ?? key);
};

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

  // A shared link carries the whole schema in the URL fragment. If one is
  // present, load it instead of the starter fields — that is the point of the
  // link. A malformed fragment falls back to the default rather than throwing.
  const shared = readSharedSpec();
  if (shared) {
    loadSharedSpec(shared);
  } else {
    addField('name');
    addField('email');
  }

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
  el('generate').addEventListener('click', generate);
  el('download').addEventListener('click', saveGenerated);
  el('copySpec').addEventListener('click', copySpec);
  el('share').addEventListener('click', share);
  el('togglePalette').addEventListener('click', () => togglePane('palette'));
  el('viewStacked').addEventListener('click', () => setView('stacked'));
  el('viewTable').addEventListener('click', () => setView('table'));
  setView(localStorage.getItem('synth.view') || 'table');
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

// typeButton renders one palette entry.
function typeButton(ty) {
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
  return b;
}

// renderTypes shows a shortlist first and keeps the full catalog one search
// away. Two hundred types listed at once is a reference manual, not a
// starting point — it buries the dozen fields most schemas actually begin
// with.
function renderTypes(query) {
  const q = query.trim().toLowerCase();
  const box = el('types');
  box.textContent = '';

  if (q) {
    // Searching means the user knows what they want: show every match, flat.
    const matching = state.types.filter((ty) => ty.kind.includes(q));
    for (const [category, items] of groupByCategory(matching)) {
      box.appendChild(groupHeading(`${categoryName(category)} (${items.length})`));
      for (const ty of items) box.appendChild(typeButton(ty));
    }
    if (matching.length === 0) {
      const p = document.createElement('p');
      p.className = 'hint';
      p.textContent = '—';
      box.appendChild(p);
    }
    return;
  }

  // Default view: the common shortlist, then every category collapsed.
  box.appendChild(groupHeading(t('common')));
  for (const kind of COMMON) {
    const ty = state.byKind.get(kind);
    if (ty) box.appendChild(typeButton(ty));
  }

  const hint = document.createElement('p');
  hint.className = 'hint palette-hint';
  hint.textContent = t('paletteHint', state.types.length);
  box.appendChild(hint);

  for (const [category, items] of groupByCategory(state.types)) {
    const details = document.createElement('details');
    const summary = document.createElement('summary');
    summary.textContent = `${categoryName(category)} (${items.length})`;
    details.appendChild(summary);
    for (const ty of items) details.appendChild(typeButton(ty));
    box.appendChild(details);
  }
}

function groupHeading(text) {
  const h = document.createElement('div');
  h.className = 'group-name';
  h.textContent = text;
  return h;
}

// ---------------------------------------------------------------- schema

function addField(kind) {
  state.fields.push({ name: uniqueName(kind), kind, params: {}, auto: true });
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

// uniqueNameExcept is uniqueName ignoring one row — the row being renamed
// must not collide with its own old name.
function uniqueNameExcept(base, index) {
  const taken = new Set(state.fields.filter((_, i) => i !== index).map((f) => f.name));
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
    const f = state.fields[index];
    f.kind = sel.value;
    f.params = {};   // old bounds rarely fit a new type
    // A name the user never typed is just the old type's label, so leaving it
    // produces a column called "name_2" holding phone numbers. Once they edit
    // the name themselves it is theirs, and changing the type must not touch it.
    if (f.auto) {
      f.name = uniqueNameExcept(sel.value, index);
    }
    renderFields();
    schedulePreview();
  });
  return sel;
}

// PARAM_SPECS declares the controls each kind offers. Showing a type's real
// knobs beats a generic min/max that means nothing for a password.
const PARAM_SPECS = {
  // A named strength sets the six toggles at once; anything set explicitly
  // still wins, so the policy is a starting point rather than a cage.
  password:     [{ key: 'strength', type: 'select', options: ['', 'pin', 'weak', 'medium', 'strong', 'very-strong'] },
                 { key: 'length', type: 'number' },
                 { key: 'min', type: 'number' }, { key: 'max', type: 'number' },
                 { key: 'lower', type: 'toggle' }, { key: 'upper', type: 'toggle' },
                 { key: 'digits', type: 'toggle' }, { key: 'symbols', type: 'toggle' },
                 { key: 'ambiguous', type: 'toggle' }],
  passwordhash: [{ key: 'strength', type: 'select', options: ['', 'pin', 'weak', 'medium', 'strong', 'very-strong'] },
                 { key: 'length', type: 'number' },
                 { key: 'min', type: 'number' }, { key: 'max', type: 'number' },
                 { key: 'lower', type: 'toggle' }, { key: 'upper', type: 'toggle' },
                 { key: 'digits', type: 'toggle' }, { key: 'symbols', type: 'toggle' },
                 { key: 'ambiguous', type: 'toggle' }],
  passphrase:   [{ key: 'words', type: 'number' }, { key: 'sep', type: 'text' }],
  cardexpiry:   [{ key: 'format', type: 'select', options: ['', 'MM/YYYY', 'YYYY-MM'] },
                 { key: 'expired', type: 'select', options: ['', 'true'] }],
  time:         [{ key: 'min', type: 'date' }, { key: 'max', type: 'date' }],
  // birthdate's bounds are ages in years, not dates: "adults" is what the
  // author means, and a spec written as `min: 1960-01-01` describes a different
  // population every year it is re-run.
  birthdate:    [{ key: 'min', type: 'number' }, { key: 'max', type: 'number' }],
  unixtime:     [{ key: 'min', type: 'date' }, { key: 'max', type: 'date' }],
  lorem:        [{ key: 'words', type: 'number' }],
  sentence:     [{ key: 'words', type: 'number' }],
  paragraph:    [{ key: 'sentences', type: 'number' }],
  card:         [{ key: 'brand', type: 'select', options: ['', 'visa', 'mastercard', 'american express', 'discover', 'jcb', 'diners club'] }],
  balance:      [{ key: 'min', type: 'number' }, { key: 'max', type: 'number' },
                 { key: 'negative', type: 'number' }],
  enum:         [{ key: 'choices', type: 'text' }],
};

// Kinds that take plain numeric bounds and nothing else.
const NUMERIC_KINDS = new Set([
  'int', 'float', 'amount', 'salary', 'percentage', 'rating',
  'temperature', 'year', 'port', 'latitude', 'longitude',
]);

function specFor(kind) {
  if (PARAM_SPECS[kind]) return PARAM_SPECS[kind];
  if (NUMERIC_KINDS.has(kind)) {
    return [{ key: 'min', type: 'number' }, { key: 'max', type: 'number' }];
  }
  return [];
}

// settingsFor lists everything adjustable on a column: the type's own options,
// then the two that belong to every column — how often the value is missing,
// and whether it is masked.
function settingsFor(field) {
  const list = [
    ...specFor(field.kind),
    { key: 'blank', type: 'percent' },
    { key: 'mask', type: 'select', options: ['', 'partial', 'hash', 'redact', 'token'], labels: {
      '': 'maskNone', partial: 'mask_partial', hash: 'mask_hash', redact: 'mask_redact', token: 'mask_token',
    } },
  ];
  // A field whose values follow the locale can be opted out of it — an English
  // category on an otherwise Uzbek record. Offered only where it does something:
  // on a type the locale never reaches, localize= is a no-op (see gen/localize).
  const meta = state.byKind.get(field.kind);
  if (meta && meta.localized) {
    list.push({ key: 'localize', type: 'select', options: ['', 'false'], labels: {
      '': 'localizeOn', false: 'localizeOff',
    } });
  }
  // The digest options only exist for the two modes that produce a digest.
  // Showing them next to mask=redact would suggest a salt changes something
  // there, and it does not.
  const mode = field.params.mask;
  if (mode === 'hash' || mode === 'token') {
    list.push({ key: 'algo', type: 'select', options: ['', 'sha256', 'sha512'], labels: {
      '': 'algoDefault', sha256: 'SHA-256', sha512: 'SHA-512',
    } });
    list.push({ key: 'salt', type: 'text' });
    list.push({ key: 'secret', type: 'text' });
    list.push({ key: 'digest', type: 'number' });
  }
  return list;
}

// countSet reports how many settings carry a value, so the row can say so
// without being opened.
function countSet(field) {
  return settingsFor(field).filter((sp) => {
    const v = field.params[sp.key];
    return v !== undefined && v !== '' && !(sp.key === 'blank' && Number(v) === 0);
  }).length;
}

function paintSettingsButton(button, field) {
  const n = countSet(field);
  button.textContent = n ? t('optionsSet', n) : t('optionsNone');
  button.classList.toggle('has-options', n > 0);
  button.setAttribute('aria-label', `${field.name}: ${t('colOptions')}`);
}

// control builds one labelled input for a setting.
function control(field, index, spec) {
  const row = document.createElement('label');
  row.className = 'setting';
  const label = document.createElement('span');
  label.textContent = t(spec.key);
  row.appendChild(label);

  let input;
  if (spec.type === 'toggle') {
    // Three states, not a checkbox: unset lets the strength policy decide,
    // which is different from explicitly turning the class off.
    input = document.createElement('select');
    for (const [value, key] of [['', 'toggleAuto'], ['true', 'toggleOn'], ['false', 'toggleOff']]) {
      const o = document.createElement('option');
      o.value = value;
      o.textContent = t(key);
      input.appendChild(o);
    }
    input.value = field.params[spec.key] ?? '';
  } else if (spec.type === 'select') {
    input = document.createElement('select');
    for (const value of spec.options) {
      const o = document.createElement('option');
      o.value = value;
      o.textContent = spec.labels ? t(spec.labels[value]) : (value === '' ? '—' : value);
      input.appendChild(o);
    }
    input.value = field.params[spec.key] ?? '';
  } else if (spec.type === 'percent') {
    input = document.createElement('input');
    input.type = 'number';
    input.min = '0';
    input.max = '100';
    input.value = field.params.blank ?? 0;
  } else {
    input = document.createElement('input');
    input.type = spec.type;
    if (field.params[spec.key] !== undefined) input.value = field.params[spec.key];
  }

  input.addEventListener('input', () => {
    const raw = String(input.value).trim();
    const params = state.fields[index].params;
    if (spec.type === 'percent') {
      const v = Number(raw);
      if (!v || v <= 0) delete params.blank;
      else params.blank = String(Math.min(v, 100));
    } else if (raw === '') {
      delete params[spec.key];
    } else {
      params[spec.key] = raw;
    }
    schedulePreview();
    // Changing the mask changes which settings exist, so rebuild the dialog.
    if (spec.key === 'mask') fillSettings(index);
  });
  input.addEventListener('change', () => input.dispatchEvent(new Event('input')));
  row.appendChild(input);
  return row;
}

// openSettings fills the dialog for one column. Edits apply as they are made —
// there is nothing to cancel, and the preview updates behind the dialog, which
// is the fastest way to see what a setting actually does.
function openSettings(index) {
  const field = state.fields[index];
  const dialog = el('colSettings');
  el('colSettingsTitle').textContent = `${field.name} — ${field.kind}`;

  fillSettings(index);
  dialog.addEventListener('close', () => renderFields(), { once: true });
  dialog.showModal();
}

function fillSettings(index) {
  const field = state.fields[index];
  const body = el('colSettingsBody');
  body.textContent = '';
  for (const spec of settingsFor(field)) body.appendChild(control(field, index, spec));
  if (field.params.mask === 'hash' || field.params.mask === 'token') {
    const note = document.createElement('p');
    note.className = 'hint';
    note.textContent = t('digestHint');
    body.appendChild(note);
  }
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
      state.fields[i].auto = false;
      schedulePreview();
    });
    nameCell.appendChild(input);

    const kindCell = document.createElement('td');
    kindCell.appendChild(kindSelect(f, i));

    // One button instead of a row of controls. A type offers between zero and
    // nine options; inline they crush every label to three letters, and the
    // count on the button says what is set without opening anything.
    const optCell = document.createElement('td');
    optCell.className = 'opts-cell';
    const open = document.createElement('button');
    open.type = 'button';
    open.className = 'opts-button';
    open.addEventListener('click', () => openSettings(i));
    optCell.appendChild(open);
    paintSettingsButton(open, f);

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
      else if (NUMBER_PARAMS.has(k)) def[k] = Number(v);
      else if ((k === 'min' || k === 'max') && !DATE_BOUND_KINDS.has(f.kind)) def[k] = Number(v);
      else def[k] = v;   // dates, strength, format and brand are words, not numbers
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
  // Any edit invalidates an earlier run: the file it produced no longer matches
  // the schema on screen.
  clearGenerated();
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

// renderRows draws the preview in the chosen layout.
//
// A wide table runs off the side of the pane and the columns on the right are
// simply unreadable, which is worse than useless when the whole point is to
// check the data. The stacked layout puts one record per block with its fields
// listed down the page, so it stays readable at any width and any column count.
function renderRows(rows, order) {
  state.lastRows = rows;
  state.lastOrder = order;
  const box = el('preview');
  box.textContent = '';
  box.appendChild(state.view === 'table' ? tableView(rows, order) : stackedView(rows, order));
}

function stackedView(rows, order) {
  const frag = document.createDocumentFragment();
  for (const [i, row] of rows.entries()) {
    const card = document.createElement('article');
    card.className = 'record';

    const head = document.createElement('div');
    head.className = 'record-head';
    head.textContent = t('record', i + 1);
    card.appendChild(head);

    const dl = document.createElement('dl');
    for (const c of order) {
      const pair = document.createElement('div');
      const dt = document.createElement('dt');
      dt.textContent = c;
      const dd = document.createElement('dd');
      const v = row[c];
      dd.textContent = v === null || v === undefined ? '' : String(v);
      pair.append(dt, dd);
      dl.appendChild(pair);
    }
    card.appendChild(dl);
    frag.appendChild(card);
  }
  return frag;
}

function tableView(rows, order) {
  const wrap = document.createElement('div');
  wrap.className = 'table-scroll';

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
  wrap.appendChild(table);
  return wrap;
}

function setView(mode) {
  state.view = mode;
  localStorage.setItem('synth.view', mode);
  el('viewStacked').classList.toggle('on', mode === 'stacked');
  el('viewTable').classList.toggle('on', mode === 'table');
  if (state.lastOrder.length) renderRows(state.lastRows, state.lastOrder);
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

async function copySpec() {
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

// ---------------------------------------------------------------- share link

// The schema travels in the URL fragment, base64url-encoded. A fragment is
// never sent to the server, so this keeps the workbench's "loopback only,
// touches nothing" promise: the link is built and read entirely in the browser.

function encodeSpec(spec) {
  const json = JSON.stringify(spec);
  // encodeURIComponent/unescape round-trips UTF-8 through btoa, which only
  // handles Latin-1; then make it URL-safe.
  return btoa(unescape(encodeURIComponent(json)))
    .replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

function decodeSpec(s) {
  const b64 = s.replace(/-/g, '+').replace(/_/g, '/');
  return JSON.parse(decodeURIComponent(escape(atob(b64))));
}

function readSharedSpec() {
  const m = location.hash.match(/[#&]s=([^&]+)/);
  if (!m) return null;
  try {
    return decodeSpec(m[1]);
  } catch {
    return null; // a truncated paste should not break the page
  }
}

// loadSharedSpec restores the editor from a decoded spec object — the same
// shape currentSpec() produces, so it round-trips exactly back into the
// controls.
function loadSharedSpec(spec) {
  if (spec.name != null) el('name').value = spec.name;
  if (spec.count != null) el('count').value = spec.count;
  if (spec.seed != null) el('seed').value = spec.seed;
  if (spec.locale && [...el('locale').options].some((o) => o.value === spec.locale)) {
    el('locale').value = spec.locale;
  }
  if (spec.format) el('format').value = spec.format;
  state.fields = [];
  const order = spec.order && spec.order.length ? spec.order : Object.keys(spec.fields || {});
  for (const name of order) {
    const def = (spec.fields && spec.fields[name]) || {};
    const params = {};
    for (const [k, v] of Object.entries(def)) {
      if (k === 'kind') continue;
      params[k] = Array.isArray(v) ? v.join(',') : String(v);
    }
    state.fields.push({ name, kind: def.kind || 'name', params, auto: false });
  }
  renderFields();
}

async function share() {
  const spec = currentSpec();
  if (spec.order.length === 0) {
    showError(t('addFieldFirst'));
    return;
  }
  const link = location.origin + location.pathname + '#s=' + encodeSpec(spec);
  // Reflect it in the address bar either way, so a refused clipboard still
  // leaves a shareable URL the user can copy by hand.
  history.replaceState(null, '', '#s=' + encodeSpec(spec));
  try {
    await navigator.clipboard.writeText(link);
    toast(t('linkCopied'));
  } catch {
    showError(t('copyFailed'));
  }
}

// generate produces the full dataset the settings ask for.
//
// It is a separate step from the preview on purpose. The preview is a hundred
// rows that redraw as you type; a million-row run is a decision, and running it
// on every keystroke would be absurd. Splitting them also gives the row count
// somewhere honest to appear.
async function generate() {
  if (state.generating) return; // a second click would start a second run

  const spec = currentSpec();
  if (spec.order.length === 0) {
    showError(t('addFieldFirst'));
    return;
  }

  setGenerating(true);
  const started = performance.now();
  try {
    const res = await fetch('/api/generate', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(spec),
    });
    if (!res.ok) {
      showError(await res.text());
      clearGenerated();
      return;
    }
    showError(null);
    const blob = await res.blob();
    state.generated = {
      blob,
      name: `${nonEmpty(spec.name, 'data')}.${spec.format}`,
      rows: spec.count,
      bytes: blob.size,
      ms: Math.round(performance.now() - started),
    };
    el('download').hidden = false;
    setStatus(t('generated', state.generated.rows, humanBytes(blob.size), state.generated.ms));
  } catch (err) {
    showError(String(err));
    clearGenerated();
  } finally {
    setGenerating(false);
  }
}

// saveGenerated writes the blob Generate already produced. No second request:
// the bytes on disk are the bytes that were measured and reported.
function saveGenerated() {
  if (!state.generated) return;
  const url = URL.createObjectURL(state.generated.blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = state.generated.name;
  a.click();
  URL.revokeObjectURL(url);
}

// clearGenerated is called whenever the schema or settings change. A download
// button left over from an earlier schema would hand you a file that does not
// match what is on screen, and nothing about the page would say so.
function clearGenerated() {
  state.generated = null;
  el('download').hidden = true;
  setStatus(null);
}

function setGenerating(on) {
  state.generating = on;
  const button = el('generate');
  button.disabled = on;
  button.classList.toggle('busy', on);
  el('generateLabel').textContent = on ? t('generating') : t('generate');
  if (on) setStatus(t('generatingRows', Number(el('count').value) || 0));
}

function setStatus(text) {
  const box = el('genStatus');
  box.hidden = !text;
  box.textContent = text || '';
}

function nonEmpty(s, fallback) {
  return s && s.trim() ? s.trim() : fallback;
}

function humanBytes(n) {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1024 / 1024).toFixed(1)} MB`;
}

boot();


// ---------------------------------------------------------------- panes

// togglePane shows or hides a side panel. Both rails collapse so the schema
// and its output can use the full width when neither is needed.
function togglePane(id, force) {
  const pane = el(id);
  const show = force === undefined ? pane.hidden : force;
  pane.hidden = !show;
  localStorage.setItem(`synth.pane.${id}`, show ? 'open' : 'closed');
  const btn = el('togglePalette');
  btn.classList.toggle('on', show);
}

function restorePanes() {
  togglePane('palette', localStorage.getItem('synth.pane.palette') !== 'closed');
}


const toolState = { catalog: [], decode: false };
