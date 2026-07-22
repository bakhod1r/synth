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
};

const el = (id) => document.getElementById(id);
const t = (key, ...args) => {
  const v = (I18N[state.lang] || I18N.en)[key];
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
  el('copySpec').addEventListener('click', copySpec);
  el('togglePalette').addEventListener('click', () => togglePane('palette'));
  el('toggleTools').addEventListener('click', () => togglePane('tools'));
  el('closeTools').addEventListener('click', () => togglePane('tools', false));
  await bootTools();
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
  password:     [{ key: 'strength', type: 'select', options: ['', 'pin', 'weak', 'medium', 'strong', 'very-strong'] },
                 { key: 'length', type: 'number' }],
  passwordhash: [{ key: 'strength', type: 'select', options: ['', 'pin', 'weak', 'medium', 'strong', 'very-strong'] }],
  passphrase:   [{ key: 'words', type: 'number' }, { key: 'sep', type: 'text' }],
  cardexpiry:   [{ key: 'format', type: 'select', options: ['', 'MM/YYYY', 'YYYY-MM'] },
                 { key: 'expired', type: 'select', options: ['', 'true'] }],
  time:         [{ key: 'min', type: 'date' }, { key: 'max', type: 'date' }],
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

function optionInputs(field, index) {
  const box = document.createElement('div');
  box.className = 'opts';

  for (const spec of specFor(field.kind)) {
    const label = t(spec.key);
    let control;
    if (spec.type === 'select') {
      control = document.createElement('select');
      for (const value of spec.options) {
        const opt = document.createElement('option');
        opt.value = value;
        opt.textContent = value === '' ? `— ${label} —` : value;
        control.appendChild(opt);
      }
      control.value = field.params[spec.key] ?? '';
    } else {
      control = document.createElement('input');
      control.type = spec.type;
      control.placeholder = label;
      if (field.params[spec.key] !== undefined) control.value = field.params[spec.key];
    }
    control.setAttribute('aria-label', `${field.name} ${label}`);
    control.addEventListener('input', () => {
      const raw = String(control.value).trim();
      if (raw === '') delete state.fields[index].params[spec.key];
      else state.fields[index].params[spec.key] = raw;
      schedulePreview();
    });
    control.addEventListener('change', () => control.dispatchEvent(new Event('input')));
    box.appendChild(control);
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
      state.fields[i].auto = false;
      schedulePreview();
    });
    nameCell.appendChild(input);

    const kindCell = document.createElement('td');
    kindCell.appendChild(kindSelect(f, i));

    const optCell = document.createElement('td');
    optCell.appendChild(optionInputs(f, i));

    // Blank share belongs to every column, not to a particular type: any
    // column can have missing values, and a primary key never does.
    const blankCell = document.createElement('td');
    blankCell.className = 'blank';
    const blank = document.createElement('input');
    blank.type = 'number';
    blank.min = '0';
    blank.max = '100';
    blank.value = f.params.blank ?? 0;
    blank.setAttribute('aria-label', `${f.name} ${t('blank')}`);
    blank.addEventListener('input', () => {
      const v = Number(blank.value);
      if (!v || v <= 0) delete state.fields[i].params.blank;
      else state.fields[i].params.blank = String(Math.min(v, 100));
      schedulePreview();
    });
    blankCell.append(blank, document.createTextNode('%'));

    // Masking belongs to every column too. A card number or a national id in
    // a fixture reaches a ticket or a screenshot sooner or later, so the safe
    // form has to be one click away rather than a documentation lookup.
    const maskCell = document.createElement('td');
    maskCell.className = 'mask';
    const mask = document.createElement('select');
    mask.setAttribute('aria-label', `${f.name} ${t('mask')}`);
    for (const mode of ['', 'partial', 'hash', 'redact', 'token']) {
      const o = document.createElement('option');
      o.value = mode;
      o.textContent = mode === '' ? t('maskNone') : t('mask_' + mode);
      mask.appendChild(o);
    }
    mask.value = f.params.mask ?? '';
    mask.addEventListener('change', () => {
      if (!mask.value) delete state.fields[i].params.mask;
      else state.fields[i].params.mask = mask.value;
      schedulePreview();
    });
    maskCell.appendChild(mask);

    const removeCell = document.createElement('td');
    removeCell.appendChild(iconButton(t('remove'), '×', () => {
      state.fields.splice(i, 1);
      renderFields();
      schedulePreview();
    }));

    tr.append(orderCell, nameCell, kindCell, optCell, blankCell, maskCell, removeCell);
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


// ---------------------------------------------------------------- panes

// togglePane shows or hides a side panel. Both rails collapse so the schema
// and its output can use the full width when neither is needed.
function togglePane(id, force) {
  const pane = el(id);
  const show = force === undefined ? pane.hidden : force;
  pane.hidden = !show;
  localStorage.setItem(`synth.pane.${id}`, show ? 'open' : 'closed');
  const btn = id === 'tools' ? el('toggleTools') : el('togglePalette');
  btn.classList.toggle('on', show);
}

function restorePanes() {
  togglePane('palette', localStorage.getItem('synth.pane.palette') !== 'closed');
  togglePane('tools', localStorage.getItem('synth.pane.tools') === 'open');
}

// ---------------------------------------------------------------- tools

const toolState = { catalog: [], decode: false };

async function bootTools() {
  toolState.catalog = await fetch('/api/tools').then((r) => r.json());
  renderToolOptions();

  el('toolOp').addEventListener('change', onToolChange);
  el('dirEncode').addEventListener('click', () => setDirection(false));
  el('dirDecode').addEventListener('click', () => setDirection(true));
  el('toolRun').addEventListener('click', runTool);
  el('toolCopy').addEventListener('click', async () => {
    const text = el('toolOutput').value;
    if (!text) return;
    try {
      await navigator.clipboard.writeText(text);
      toast(t('copiedOutput'));
    } catch { /* clipboard refused; the value is already on screen */ }
  });
  onToolChange();
  restorePanes();
}

function groupLabel(group) {
  return { hash: t('groupHash'), password: t('groupPassword'), encoding: t('groupEncoding') }[group] || group;
}

function renderToolOptions() {
  const sel = el('toolOp');
  const previous = sel.value;
  sel.textContent = '';
  const groups = new Map();
  for (const tool of toolState.catalog) {
    if (!groups.has(tool.group)) groups.set(tool.group, []);
    groups.get(tool.group).push(tool);
  }
  for (const [group, items] of groups) {
    const og = document.createElement('optgroup');
    og.label = groupLabel(group);
    for (const tool of items) {
      const opt = document.createElement('option');
      opt.value = opt.textContent = tool.op;
      og.appendChild(opt);
    }
    sel.appendChild(og);
  }
  if (previous) sel.value = previous;
}

function currentTool() {
  return toolState.catalog.find((tool) => tool.op === el('toolOp').value);
}

// onToolChange shows only the inputs the chosen operation actually uses, and
// hides the encode/decode switch for one-way hashes rather than offering a
// decode that cannot exist.
function onToolChange() {
  const tool = currentTool();
  if (!tool) return;

  el('toolDirection').hidden = !tool.reversible;
  if (!tool.reversible) setDirection(false);
  el('toolKeyRow').hidden = !tool.needsKey;
  el('toolSaltRow').hidden = !tool.needsSalt;

  const warn = el('toolWarn');
  warn.hidden = !tool.warn;
  warn.textContent = tool.warn || '';

  el('toolOutput').value = '';
  el('toolNote').hidden = true;
}

function setDirection(decode) {
  toolState.decode = decode;
  el('dirEncode').classList.toggle('on', !decode);
  el('dirDecode').classList.toggle('on', decode);
}

async function runTool() {
  const tool = currentTool();
  if (!tool) return;
  const input = el('toolInput').value;
  if (!input.trim()) {
    showToolResult('', t('needInput'), true);
    return;
  }
  const op = tool.reversible ? `${tool.op}-${toolState.decode ? 'decode' : 'encode'}` : tool.op;
  const res = await fetch('/api/tools', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      op,
      input,
      key: el('toolKey').value,
      salt: el('toolSalt').value,
      iterations: Number(el('toolIter').value || 0),
    }),
  });
  const text = await res.text();
  if (!res.ok) {
    showToolResult('', text, true);
    return;
  }
  const data = JSON.parse(text);
  showToolResult(data.output, data.note || '', false);
}

function showToolResult(output, note, isError) {
  el('toolOutput').value = output;
  const box = el('toolNote');
  box.hidden = !note;
  box.textContent = note;
  box.classList.toggle('warn', Boolean(isError));
}
