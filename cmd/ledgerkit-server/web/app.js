"use strict";

const STORAGE_KEY_URL = "ledgerkit.serverUrl";
const STORAGE_KEY_TOKEN = "ledgerkit.token";
const STORAGE_KEY_THEME = "ledgerkit.theme";

function getTheme() {
  try {
    return localStorage.getItem(STORAGE_KEY_THEME) || "light";
  } catch {
    return "light";
  }
}

function setTheme(theme) {
  try {
    localStorage.setItem(STORAGE_KEY_THEME, theme);
  } catch {
    // localStorage unavailable — theme just won't persist across reloads.
  }
  document.documentElement.dataset.theme = theme;
}

// Apply immediately (not waiting for DOMContentLoaded) so there's no flash
// of the wrong theme before the rest of the page has parsed.
setTheme(getTheme());

function getServerUrl() {
  try {
    return localStorage.getItem(STORAGE_KEY_URL) || "";
  } catch {
    return "";
  }
}

function getToken() {
  try {
    return localStorage.getItem(STORAGE_KEY_TOKEN) || "";
  } catch {
    return "";
  }
}

function saveSettings(url, token) {
  try {
    localStorage.setItem(STORAGE_KEY_URL, url);
    localStorage.setItem(STORAGE_KEY_TOKEN, token);
  } catch {
    // localStorage unavailable (e.g. private browsing) — settings just won't persist.
  }
}

async function api(path, options = {}) {
  const base = getServerUrl().replace(/\/+$/, "");
  const res = await fetch(base + path, {
    ...options,
    headers: {
      ...(options.headers || {}),
      Authorization: "Bearer " + getToken(),
    },
  });
  if (!res.ok) {
    let msg = res.statusText;
    try {
      const body = await res.json();
      if (body.error) msg = body.error;
    } catch {
      // response wasn't JSON; fall back to statusText
    }
    throw new Error(msg);
  }
  if (res.status === 204) return null;
  return res.json();
}

function signClass(sign) {
  if (sign > 0) return "positive";
  if (sign < 0) return "negative";
  return "zero";
}

function el(tag, attrs = {}, children = []) {
  const node = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs)) {
    if (k === "class") node.className = v;
    else node.setAttribute(k, v);
  }
  for (const child of children) {
    node.appendChild(typeof child === "string" ? document.createTextNode(child) : child);
  }
  return node;
}

// showError renders an error message via textContent (never innerHTML) since
// the message can come from an arbitrary configured server's response body
// (Settings lets a user point at any URL) — treat it as untrusted text, not
// markup, so a crafted error string can't inject script into this origin.
function showError(container, message) {
  container.innerHTML = "";
  container.appendChild(el("div", { class: "empty-state" }, ["錯誤: " + message]));
}

function cssVar(name) {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim();
}

// cssVars reads several custom properties from a single getComputedStyle()
// call, for chart renderers that need multiple colors per render — calling
// cssVar() once per color each forces its own style recalculation.
function cssVars(names) {
  const styles = getComputedStyle(document.documentElement);
  const out = {};
  for (const name of names) out[name] = styles.getPropertyValue(name).trim();
  return out;
}

function formatMoney(n, commodity = "$") {
  const rounded = Math.round(n);
  const sign = rounded < 0 ? "-" : "";
  const abs = Math.abs(rounded).toLocaleString();
  // Alphabetic commodities (TWD, USD) go after the number; symbols ($, €)
  // go before it — matches internal/journal's Amount.String() convention.
  return /^[A-Za-z]+$/.test(commodity) ? sign + abs + " " + commodity : sign + commodity + abs;
}

// dominantCommodity picks the commodity with the largest total absolute
// value across entries. Reports' hero/trend/donut each show one number, so
// mixing commodities into a single sum would be meaningless (a $5 + a
// 500 TWD entry summed to "505" is neither) — restrict to whichever
// commodity actually represents this journal instead of ignoring the field.
function dominantCommodity(entries) {
  const totals = new Map();
  for (const e of entries) {
    const c = e.amount.commodity;
    totals.set(c, (totals.get(c) || 0) + Math.abs(parseFloat(e.amount.value)));
  }
  let best = "$";
  let bestVal = -1;
  for (const [c, v] of totals) {
    if (v > bestVal) {
      best = c;
      bestVal = v;
    }
  }
  return best;
}

// ---- Category color/avatar (a small hashed categorical palette, so each
// account/category gets a stable, distinct color without a lookup table) ----

const CATEGORICAL_LIGHT = ["#2a78d6", "#eb6834", "#1baf7a", "#eda100", "#e87ba4", "#008300", "#4a3aa7", "#e34948"];
const CATEGORICAL_DARK = ["#3987e5", "#d95926", "#199e70", "#c98500", "#d55181", "#008300", "#9085e9", "#e66767"];

// isDarkMode mirrors style.css's cascade: an explicit data-theme attribute
// (a manual theme toggle, if one is ever added) wins over the OS preference,
// same as the CSS rules it's paired with — so avatar/pill colors (computed
// here in JS) never end up mismatched against chart colors (read via
// cssVar(), which already follows that same cascade automatically).
function isDarkMode() {
  const theme = document.documentElement.dataset.theme;
  if (theme === "dark") return true;
  if (theme === "light") return false;
  return window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches;
}

function categoryColor(name) {
  let h = 0;
  for (let i = 0; i < name.length; i++) h = (h * 31 + name.charCodeAt(i)) >>> 0;
  const palette = isDarkMode() ? CATEGORICAL_DARK : CATEGORICAL_LIGHT;
  return palette[h % palette.length];
}

// categoryOf picks the segment right under the top-level account
// (expenses:food:coffee -> food) as the "category" identity for
// avatars/pills/breakdowns; falls back to the top-level segment for
// two-part accounts (income:salary -> salary).
function categoryOf(account) {
  const parts = account.split(":");
  return parts[1] || parts[0];
}

function avatar(name, size) {
  return el("div", { class: "avatar" + (size === "small" ? " small" : ""), style: `background:${categoryColor(name)}` }, [
    [...name][0]?.toUpperCase() ?? "?",
  ]);
}

// ---- Balance view ----

async function loadBalance() {
  const rowsEl = document.getElementById("balance-rows");
  const totalEl = document.getElementById("balance-total");
  rowsEl.innerHTML = '<div class="empty-state">loading…</div>';
  totalEl.textContent = "";
  try {
    const data = await api("/api/balance");
    rowsEl.innerHTML = "";
    if (data.rows.length === 0) {
      rowsEl.innerHTML = '<div class="empty-state">還沒有任何交易</div>';
      totalEl.innerHTML = "";
      return;
    }
    let card = null;
    for (const row of data.rows) {
      if (row.depth === 0) {
        rowsEl.appendChild(el("div", { class: "balance-group-header" }, [avatar(row.name, "small"), el("span", {}, [row.name])]));
        card = el("div", { class: "card" }, []);
        rowsEl.appendChild(card);
        // The group header above already shows this account's name; only
        // its total is still useful here, so render that alone rather than
        // repeating "assets" as the card's first row.
        for (const amt of row.amounts) {
          card.appendChild(
            el("div", { class: "row depth-0" }, [
              el("span", { class: "account" }, ["小計"]),
              el("span", { class: "amount " + signClass(amt.sign) }, [amt.display]),
            ])
          );
        }
        continue;
      }
      for (const amt of row.amounts) {
        // depth 1 gets the avatar (it's the "category" level); deeper
        // levels are indented text only, scaling with depth so a 4th+
        // level (e.g. expenses:food:coffee:starbucks) still reads as
        // nested instead of silently landing flush-left.
        const cell =
          row.depth === 1
            ? el("div", { class: "account-cell" }, [avatar(row.name), el("span", { class: "account" }, [row.name])])
            : el("span", { class: "account indented" }, [row.name]);
        if (row.depth > 1) cell.style.paddingLeft = 44 + (row.depth - 2) * 14 + "px";
        card.appendChild(
          el("div", { class: "row depth-" + row.depth }, [cell, el("span", { class: "amount " + signClass(amt.sign) }, [amt.display])])
        );
      }
    }
    totalEl.innerHTML = "";
    for (const amt of data.total) {
      totalEl.appendChild(
        el("div", { class: "total-row" }, [
          el("span", {}, ["總計"]),
          el("span", { class: "amount " + signClass(amt.sign) }, [amt.display]),
        ])
      );
    }
  } catch (err) {
    showError(rowsEl, err.message);
  }
}

// ---- Register view (grouped by day) ----

async function loadRegister() {
  const entriesEl = document.getElementById("register-entries");
  entriesEl.innerHTML = '<div class="empty-state">loading…</div>';
  try {
    const entries = await api("/api/register");
    entriesEl.innerHTML = "";
    if (entries.length === 0) {
      entriesEl.innerHTML = '<div class="empty-state">還沒有任何交易</div>';
      return;
    }
    let lastDate = null;
    let card = null;
    for (const e of entries.slice().reverse()) {
      if (e.date !== lastDate) {
        entriesEl.appendChild(el("div", { class: "day-header" }, [e.date]));
        card = el("div", { class: "card" }, []);
        entriesEl.appendChild(card);
        lastDate = e.date;
      }
      const running = e.running.map((a) => a.display).join(", ");
      const category = categoryOf(e.account);
      const color = categoryColor(category);
      card.appendChild(
        el("div", { class: "register-entry" }, [
          avatar(category),
          el("div", { class: "entry-main" }, [
            el("div", { class: "desc" }, [e.description]),
            el("span", { class: "pill", style: `background:${color}26;color:${color}` }, [category]),
          ]),
          el("div", { class: "right" }, [
            el("div", { class: "amount " + signClass(e.amount.sign) }, [e.amount.display]),
            el("div", { class: "running" }, [running]),
          ]),
        ])
      );
    }
  } catch (err) {
    showError(entriesEl, err.message);
  }
}

// ---- Reports view ----

function monthKey(dateStr) {
  return dateStr.slice(0, 7);
}

// currentMonthKey uses local date fields, not toISOString() (which is UTC) —
// ledger dates are local calendar dates, so a UTC-derived "this month" can be
// off by one for hours around local midnight on the 1st, depending on the
// viewer's timezone offset.
function currentMonthKey() {
  const now = new Date();
  return now.getFullYear() + "-" + String(now.getMonth() + 1).padStart(2, "0");
}

function monthLabel(key) {
  const [, m] = key.split("-");
  return parseInt(m, 10) + "月";
}

// Draws a bar from the zero baseline to tipY, rounded only at the tip end
// (square at the baseline) per the mark spec.
function barPath(x, width, baselineY, tipY, r) {
  const up = tipY < baselineY;
  r = Math.max(0, Math.min(r, width / 2, Math.abs(baselineY - tipY)));
  if (up) {
    return `M${x},${baselineY} L${x},${tipY + r} Q${x},${tipY} ${x + r},${tipY} L${x + width - r},${tipY} Q${x + width},${tipY} ${x + width},${tipY + r} L${x + width},${baselineY} Z`;
  }
  return `M${x},${baselineY} L${x},${tipY - r} Q${x},${tipY} ${x + r},${tipY} L${x + width - r},${tipY} Q${x + width},${tipY} ${x + width},${tipY - r} L${x + width},${baselineY} Z`;
}

function renderTrendChart(container, months) {
  if (months.length === 0) {
    container.innerHTML = '<div class="empty-state">還沒有資料</div>';
    return;
  }
  const width = 320;
  const height = 180;
  const baselineY = height / 2;
  const halfHeight = height / 2 - 20;
  const maxVal = Math.max(1, ...months.flatMap((m) => [m.income, m.expense]));
  const slotWidth = width / months.length;
  const barWidth = Math.min(22, slotWidth * 0.28);
  const gap = 2;
  const theme = cssVars(["--income", "--expense", "--baseline", "--text-muted"]);
  const income = theme["--income"] || "#2a78d6";
  const expense = theme["--expense"] || "#e34948";
  const baseline = theme["--baseline"] || "#c3c2b7";
  const muted = theme["--text-muted"] || "#898781";

  let svg = `<svg viewBox="0 0 ${width} ${height}" xmlns="http://www.w3.org/2000/svg">`;
  svg += `<line x1="0" y1="${baselineY}" x2="${width}" y2="${baselineY}" stroke="${baseline}" stroke-width="1"/>`;
  months.forEach((m, i) => {
    const slotCenter = slotWidth * i + slotWidth / 2;
    // Clamp to a minimum visible sliver so a non-zero value never disappears
    // next to a much larger one on the shared scale (e.g. $60 of expenses
    // beside $52,000 of income) — the point is to still register as "present".
    const incomeH = m.income > 0 ? Math.max((m.income / maxVal) * halfHeight, 3) : 0;
    const expenseH = m.expense > 0 ? Math.max((m.expense / maxVal) * halfHeight, 3) : 0;
    if (incomeH > 0.5) {
      svg += `<path d="${barPath(slotCenter - barWidth - gap / 2, barWidth, baselineY - gap, baselineY - gap - incomeH, 4)}" fill="${income}"/>`;
    }
    if (expenseH > 0.5) {
      svg += `<path d="${barPath(slotCenter + gap / 2, barWidth, baselineY + gap, baselineY + gap + expenseH, 4)}" fill="${expense}"/>`;
    }
    svg += `<text x="${slotCenter}" y="${height - 4}" text-anchor="middle" font-size="10" fill="${muted}">${monthLabel(m.month)}</text>`;
  });
  svg += `</svg>`;
  container.innerHTML = svg;
}

// renderCategoryDonut draws a donut chart (segments sized by share of
// total) with the total in the center, plus a legend list below — this is
// the "簡單記帳"-style category breakdown (a ranked bar list reads more like
// accounting software than a consumer expense tracker).
function renderCategoryDonut(donutEl, legendEl, categories, commodity) {
  const total = categories.reduce((sum, c) => sum + c.value, 0);
  if (categories.length === 0 || total <= 0) {
    donutEl.innerHTML = "";
    legendEl.innerHTML = '<div class="empty-state">本月還沒有支出</div>';
    return;
  }

  const size = 200;
  const radius = 76;
  const stroke = 28;
  const circumference = 2 * Math.PI * radius;
  const gapLen = (2.5 / 360) * circumference; // small surface gap between segments

  let offset = 0;
  let segments = "";
  for (const c of categories) {
    const rawLen = (c.value / total) * circumference;
    const dash = Math.max(rawLen - gapLen, 0);
    segments += `<circle cx="${size / 2}" cy="${size / 2}" r="${radius}" fill="none" stroke="${categoryColor(c.name)}" stroke-width="${stroke}" stroke-linecap="round" stroke-dasharray="${dash} ${circumference - dash}" stroke-dashoffset="${-offset}" transform="rotate(-90 ${size / 2} ${size / 2})"/>`;
    offset += rawLen;
  }
  const track = cssVar("--gridline") || "#e1e0d9";
  donutEl.innerHTML = `<svg viewBox="0 0 ${size} ${size}">
    <circle cx="${size / 2}" cy="${size / 2}" r="${radius}" fill="none" stroke="${track}" stroke-width="${stroke}"/>
    ${segments}
    <text x="${size / 2}" y="${size / 2 - 6}" text-anchor="middle" class="donut-center-label">本月支出</text>
    <text x="${size / 2}" y="${size / 2 + 18}" text-anchor="middle" class="donut-center-value">${formatMoney(total, commodity)}</text>
  </svg>`;

  legendEl.innerHTML = "";
  for (const c of categories) {
    const pct = Math.round((c.value / total) * 100);
    legendEl.appendChild(
      el("div", { class: "legend-row" }, [
        el("span", { class: "swatch-dot", style: `background:${categoryColor(c.name)}` }),
        el("div", { class: "legend-main" }, [
          el("span", { class: "legend-name" }, [c.name]),
          el("span", {}, [formatMoney(c.value, commodity) + "（" + pct + "%）"]),
        ]),
      ])
    );
  }
}

async function loadReports() {
  const trendEl = document.getElementById("trend-chart");
  const donutEl = document.getElementById("category-donut");
  const legendEl = document.getElementById("category-legend");
  trendEl.innerHTML = '<div class="empty-state">loading…</div>';
  legendEl.innerHTML = '<div class="empty-state">loading…</div>';
  try {
    const allEntries = await api("/api/register");
    const commodity = dominantCommodity(allEntries);
    // Entries in any other commodity are excluded rather than summed in —
    // see dominantCommodity's note. Personal journals are almost always
    // single-commodity, so this only matters for the rare mixed case.
    const entries = allEntries.filter((e) => e.amount.commodity === commodity);
    const monthly = new Map(); // month -> {income, expense}
    const categories = new Map(); // this-month category -> total
    const thisMonth = currentMonthKey();

    for (const e of entries) {
      const val = parseFloat(e.amount.value);
      const mk = monthKey(e.date);
      if (e.account.startsWith("income:")) {
        const bucket = monthly.get(mk) || { income: 0, expense: 0 };
        bucket.income += -val; // income postings are credit-normal (negative); flip for display
        monthly.set(mk, bucket);
      } else if (e.account.startsWith("expenses:")) {
        const bucket = monthly.get(mk) || { income: 0, expense: 0 };
        bucket.expense += val;
        monthly.set(mk, bucket);
        if (mk === thisMonth) {
          const cat = categoryOf(e.account);
          categories.set(cat, (categories.get(cat) || 0) + val);
        }
      }
    }

    const months = [...monthly.entries()]
      .sort(([a], [b]) => a.localeCompare(b))
      .slice(-6)
      .map(([month, v]) => ({ month, ...v }));
    renderTrendChart(trendEl, months);

    const catList = [...categories.entries()]
      .map(([name, value]) => ({ name, value }))
      .sort((a, b) => b.value - a.value);
    const top = catList.slice(0, 7);
    const restTotal = catList.slice(7).reduce((sum, c) => sum + c.value, 0);
    if (restTotal > 0) top.push({ name: "其他", value: restTotal });
    renderCategoryDonut(donutEl, legendEl, top, commodity);

    const thisMonthTotals = monthly.get(thisMonth) || { income: 0, expense: 0 };
    const net = thisMonthTotals.income - thisMonthTotals.expense;
    document.getElementById("hero-income").textContent = formatMoney(thisMonthTotals.income, commodity);
    document.getElementById("hero-expense").textContent = formatMoney(thisMonthTotals.expense, commodity);
    // hero-net sits on a gradient card (see .hero-card), so it stays white
    // rather than sign-colored — formatMoney's leading "-" already carries
    // the sign, and red/green text doesn't reliably contrast every gradient.
    document.getElementById("hero-net").textContent = formatMoney(net, commodity);
  } catch (err) {
    showError(trendEl, err.message);
    donutEl.innerHTML = "";
    legendEl.innerHTML = "";
  }
}

// ---- Add view: quick entry (category grid + numeric keypad) is the
// primary flow — no account paths to type. It composes a normal two-posting
// transaction under the hood (category account + a fixed default asset
// account, with the second leg elided so the server infers it). "進階模式"
// below still exposes the full multi-posting form for anything this
// two-tap flow can't express (transfers, custom accounts, 3+ postings). ----

const DEFAULT_ASSET_ACCOUNT = "assets:cash";
// Quick-entry amounts need an explicit commodity, or they parse with an
// empty one (journal.ParseAmount("780") has commodity "") — a different
// bucket than the rest of a "$"-denominated journal, which would silently
// vanish from Reports' dominant-commodity filter. Change this if your
// journal actually uses a different symbol.
const QUICK_ENTRY_COMMODITY = "$";

const EXPENSE_CATEGORIES = [
  { label: "餐飲", icon: "🍚", account: "expenses:food" },
  { label: "交通", icon: "🚗", account: "expenses:transport" },
  { label: "購物", icon: "🛍️", account: "expenses:shopping" },
  { label: "娛樂", icon: "🎮", account: "expenses:fun" },
  { label: "醫療", icon: "🏥", account: "expenses:medical" },
  { label: "居家", icon: "🏠", account: "expenses:home" },
  { label: "通訊", icon: "📱", account: "expenses:communication" },
  { label: "其他", icon: "📦", account: "expenses:other" },
];
const INCOME_CATEGORIES = [
  { label: "薪資", icon: "💰", account: "income:salary" },
  { label: "獎金", icon: "🎁", account: "income:bonus" },
  { label: "投資", icon: "📈", account: "income:investment" },
  { label: "其他", icon: "📦", account: "income:other" },
];

let quickType = "expense";
let quickCategory = null;
let quickAmount = "0";

function setAddStatus(text, kind) {
  const status = document.getElementById("add-status");
  status.textContent = text;
  status.className = "status" + (kind ? " " + kind : "");
}

function renderCategoryGrid() {
  const grid = document.getElementById("category-grid");
  grid.innerHTML = "";
  const cats = quickType === "expense" ? EXPENSE_CATEGORIES : INCOME_CATEGORIES;
  for (const cat of cats) {
    const btn = el("button", { type: "button", class: "category-item" }, [
      el("div", { class: "icon-circle", style: `background:${categoryColor(cat.account)}` }, [cat.icon]),
      el("span", {}, [cat.label]),
    ]);
    btn.addEventListener("click", () => openKeypad(cat));
    grid.appendChild(btn);
  }
}

function openKeypad(category) {
  quickCategory = category;
  quickAmount = "0";
  document.getElementById("keypad-category-label").textContent = (quickType === "expense" ? "支出" : "收入") + " · " + category.label;
  document.getElementById("keypad-amount").textContent = quickAmount;
  document.getElementById("quick-note").value = "";
  document.getElementById("quick-date").valueAsDate = new Date();
  document.getElementById("type-toggle").hidden = true;
  document.getElementById("category-grid").hidden = true;
  document.getElementById("toggle-advanced").hidden = true;
  document.getElementById("keypad-panel").hidden = false;
}

function closeKeypad() {
  document.getElementById("keypad-panel").hidden = true;
  document.getElementById("type-toggle").hidden = false;
  document.getElementById("category-grid").hidden = false;
  document.getElementById("toggle-advanced").hidden = false;
}

function pressKey(key) {
  if (key === "del") {
    quickAmount = quickAmount.length > 1 ? quickAmount.slice(0, -1) : "0";
  } else if (key === ".") {
    if (!quickAmount.includes(".")) quickAmount += ".";
  } else if (quickAmount === "0") {
    quickAmount = key;
  } else {
    quickAmount += key;
  }
  document.getElementById("keypad-amount").textContent = quickAmount;
}

async function submitQuickEntry() {
  setAddStatus("", "");
  const amount = parseFloat(quickAmount);
  if (!amount || amount <= 0) {
    setAddStatus("請輸入金額", "error");
    return;
  }
  const date = document.getElementById("quick-date").value;
  const note = document.getElementById("quick-note").value.trim();
  const description = note || quickCategory.label;
  const amountStr = QUICK_ENTRY_COMMODITY + amount;
  const postings =
    quickType === "expense"
      ? [
          { account: quickCategory.account, amount: amountStr },
          { account: DEFAULT_ASSET_ACCOUNT, amount: "" },
        ]
      : [
          { account: DEFAULT_ASSET_ACCOUNT, amount: amountStr },
          { account: quickCategory.account, amount: "" },
        ];

  try {
    await api("/api/transactions", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ date, description, postings }),
    });
    closeKeypad();
    setAddStatus("已新增", "ok");
  } catch (err) {
    setAddStatus("錯誤: " + err.message, "error");
  }
}

// ---- Add view: advanced mode (the original full posting-list form) ----

async function populateAccountSuggestions() {
  const datalist = document.getElementById("account-suggestions");
  try {
    const accounts = await api("/api/accounts");
    datalist.innerHTML = "";
    for (const a of accounts) {
      datalist.appendChild(el("option", { value: a }));
    }
  } catch {
    // Suggestions are a nicety; a failed fetch here shouldn't block the form.
  }
}

function addPostingRow(account = "", amount = "") {
  const container = document.getElementById("postings");
  const row = el("div", { class: "posting-row" }, [
    el("input", { type: "text", list: "account-suggestions", placeholder: "帳戶，例如 expenses:food:coffee", value: account }),
    el("input", { type: "text", placeholder: "金額（留空 = 自動推算）", value: amount }),
  ]);
  container.appendChild(row);
}

function resetAdvancedForm() {
  document.getElementById("adv-date").valueAsDate = new Date();
  document.getElementById("adv-desc").value = "";
  document.getElementById("postings").innerHTML = "";
  addPostingRow();
  addPostingRow();
}

async function submitAdvancedForm(ev) {
  ev.preventDefault();
  setAddStatus("", "");

  const date = document.getElementById("adv-date").value;
  const description = document.getElementById("adv-desc").value.trim();
  const rows = [...document.querySelectorAll("#postings .posting-row")];
  const postings = rows
    .map((row) => {
      const [accountInput, amountInput] = row.querySelectorAll("input");
      return { account: accountInput.value.trim(), amount: amountInput.value.trim() };
    })
    .filter((p) => p.account !== "");

  try {
    await api("/api/transactions", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ date, description, postings }),
    });
    resetAdvancedForm();
    setAddStatus("已新增", "ok");
  } catch (err) {
    setAddStatus("錯誤: " + err.message, "error");
  }
}

function setAdvancedMode(showAdvanced) {
  document.getElementById("type-toggle").hidden = showAdvanced;
  document.getElementById("category-grid").hidden = showAdvanced;
  document.getElementById("keypad-panel").hidden = true;
  document.getElementById("advanced-form").hidden = !showAdvanced;
  // Always unhidden here, not just by closeKeypad(): if the user leaves the
  // Add tab while the keypad is open (skipping keypad-back/keypad-ok), this
  // link would otherwise stay hidden until they happened to close a keypad
  // properly again — resetAddView() -> setAdvancedMode(false) runs on every
  // re-entry to the tab, so this is the one place guaranteed to un-stick it.
  document.getElementById("toggle-advanced").hidden = false;
  document.getElementById("toggle-advanced").textContent = showAdvanced ? "‹ 返回簡易模式" : "進階模式（自訂帳戶）";
  if (showAdvanced) {
    resetAdvancedForm();
    populateAccountSuggestions();
  }
}

function resetAddView() {
  quickType = "expense";
  for (const b of document.querySelectorAll("#type-toggle button")) {
    b.classList.toggle("active", b.dataset.type === "expense");
  }
  setAdvancedMode(false);
  renderCategoryGrid();
  setAddStatus("", "");
}

// ---- Settings view ----

function loadSettingsForm() {
  document.getElementById("server-url").value = getServerUrl();
  document.getElementById("api-token").value = getToken();
  const current = getTheme();
  for (const btn of document.querySelectorAll("#theme-toggle button")) {
    btn.classList.toggle("active", btn.dataset.themeChoice === current);
  }
}

function submitSettingsForm(ev) {
  ev.preventDefault();
  const url = document.getElementById("server-url").value.trim();
  const token = document.getElementById("api-token").value.trim();
  saveSettings(url, token);
  const status = document.getElementById("settings-status");
  status.textContent = "已儲存";
  status.className = "status ok";
}

// ---- Navigation ----

function showView(name) {
  for (const btn of document.querySelectorAll("nav.tabbar button")) {
    btn.classList.toggle("active", btn.dataset.view === name);
  }
  for (const view of document.querySelectorAll(".view")) {
    view.classList.toggle("active", view.id === "view-" + name);
  }
  if (name === "reports") loadReports();
  if (name === "balance") loadBalance();
  if (name === "register") loadRegister();
  if (name === "add") resetAddView();
  if (name === "settings") loadSettingsForm();
}

document.addEventListener("DOMContentLoaded", () => {
  for (const btn of document.querySelectorAll("nav.tabbar button")) {
    btn.addEventListener("click", () => showView(btn.dataset.view));
  }

  for (const btn of document.querySelectorAll("#type-toggle button")) {
    btn.addEventListener("click", () => {
      quickType = btn.dataset.type;
      for (const b of document.querySelectorAll("#type-toggle button")) b.classList.toggle("active", b === btn);
      renderCategoryGrid();
    });
  }
  document.getElementById("keypad-back").addEventListener("click", closeKeypad);
  document.getElementById("keypad-ok").addEventListener("click", submitQuickEntry);
  for (const btn of document.querySelectorAll(".keypad-grid button")) {
    btn.addEventListener("click", () => pressKey(btn.dataset.key));
  }
  document.getElementById("toggle-advanced").addEventListener("click", () => {
    setAdvancedMode(document.getElementById("advanced-form").hidden);
  });
  document.getElementById("add-posting").addEventListener("click", () => addPostingRow());
  document.getElementById("advanced-form").addEventListener("submit", submitAdvancedForm);

  document.getElementById("settings-form").addEventListener("submit", submitSettingsForm);
  for (const btn of document.querySelectorAll("#theme-toggle button")) {
    btn.addEventListener("click", () => {
      setTheme(btn.dataset.themeChoice);
      loadSettingsForm();
    });
  }

  showView("reports");
});
