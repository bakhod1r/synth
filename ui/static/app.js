// Synth workbench. No framework, no build step, no external requests.
'use strict';

const state = {
  types: [],
  fields: [],   // { name, kind }
};

const el = (id) => document.getElementById(id);

// ---------------------------------------------------------------- boot

async function boot() {
  const [types, locales] = await Promise.all([
    fetch('/api/types').then((r) => r.json()),
    fetch('/api/locales').then((r) => r.json()),
  ]);
  state.types = types;
  renderTypes('');

  const sel = el('locale');
  for (const name of locales) {
    const opt = document.createElement('option');
    opt.value = opt.textContent = name;
    if (name === 'en_US') opt.selected = true;
    sel.appendChild(opt);
  }

  addField('name');
  addField('email');

  el('search').addEventListener('input', (e) => renderTypes(e.target.value));
  for (const id of ['name', 'count', 'locale', 'seed']) {
    el(id).addEventListener('input', schedulePreview);
  }
  el('download').addEventListener('click', download);
}

// ---------------------------------------------------------------- palette

function renderTypes(query) {
  const q = query.trim().toLowerCase();
  const matching = state.types.filter((t) => !q || t.kind.includes(q));

  const groups = new Map();
  for (const t of matching) {
    if (!groups.has(t.category)) groups.set(t.category, []);
    groups.get(t.category).push(t);
  }

  const box = el('types');
  box.textContent = '';
  for (const [category, items] of [...groups].sort((a, b) => a[0].localeCompare(b[0]))) {
    const h = document.createElement('div');
    h.className = 'group-name';
    h.textContent = `${category} (${items.length})`;
    box.appendChild(h);

    for (const t of items) {
      const b = document.createElement('button');
      b.type = 'button';
      b.className = t.localized ? 'type localized' : 'type';
      b.title = t.locales && t.locales.length
        ? `Localized in ${t.locales.length} locales: ${t.locales.join(', ')}`
        : t.localized
          ? 'Values follow the selected locale'
          : 'Same values in every locale';

      const dot = document.createElement('span');
      dot.className = 'dot';
      b.appendChild(dot);
      b.appendChild(document.createTextNode(t.kind));
      b.addEventListener('click', () => addField(t.kind));
      box.appendChild(b);
    }
  }
}

// ---------------------------------------------------------------- schema

function addField(kind) {
  state.fields.push({ name: uniqueName(kind), kind });
  renderFields();
  schedulePreview();
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

function renderFields() {
  const body = el('fields').querySelector('tbody');
  body.textContent = '';

  for (const [i, f] of state.fields.entries()) {
    const tr = document.createElement('tr');

    const nameCell = document.createElement('td');
    const input = document.createElement('input');
    input.value = f.name;
    input.setAttribute('aria-label', 'Column name');
    input.addEventListener('input', () => {
      state.fields[i].name = input.value;
      schedulePreview();
    });
    nameCell.appendChild(input);

    const kindCell = document.createElement('td');
    kindCell.className = 'kind';
    kindCell.textContent = f.kind;

    const removeCell = document.createElement('td');
    const remove = document.createElement('button');
    remove.type = 'button';
    remove.className = 'remove';
    remove.textContent = '×';
    remove.setAttribute('aria-label', `Remove ${f.name}`);
    remove.addEventListener('click', () => {
      state.fields.splice(i, 1);
      renderFields();
      schedulePreview();
    });
    removeCell.appendChild(remove);

    tr.append(nameCell, kindCell, removeCell);
    body.appendChild(tr);
  }
  el('empty').hidden = state.fields.length > 0;
}

function currentSpec(count) {
  const fields = {};
  const order = [];
  for (const f of state.fields) {
    const name = f.name.trim();
    if (!name || fields[name]) continue;
    fields[name] = { kind: f.kind };
    order.push(name);
  }
  return {
    name: el('name').value || 'data',
    count: count ?? Number(el('count').value || 10),
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
    renderRows(JSON.parse(text), spec.order);
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

// ---------------------------------------------------------------- download

async function download() {
  const spec = currentSpec();
  if (spec.order.length === 0) {
    showError('Add at least one field first.');
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
