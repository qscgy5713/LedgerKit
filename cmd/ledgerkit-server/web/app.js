"use strict";

const STORAGE_KEY_URL = "ledgerkit.serverUrl";
const STORAGE_KEY_TOKEN = "ledgerkit.token";

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

// ---- Balance view ----

async function loadBalance() {
  const rowsEl = document.getElementById("balance-rows");
  const totalEl = document.getElementById("balance-total");
  rowsEl.textContent = "loading…";
  totalEl.textContent = "";
  try {
    const data = await api("/api/balance");
    rowsEl.innerHTML = "";
    for (const row of data.rows) {
      for (const amt of row.amounts) {
        rowsEl.appendChild(
          el("div", { class: "row" }, [
            el("span", { class: "account" }, ["  ".repeat(row.depth) + row.name]),
            el("span", { class: "amount " + signClass(amt.sign) }, [amt.display]),
          ])
        );
      }
    }
    totalEl.innerHTML = "";
    for (const amt of data.total) {
      totalEl.appendChild(
        el("div", { class: "total-row" }, [
          el("span", {}, ["total"]),
          el("span", { class: "amount " + signClass(amt.sign) }, [amt.display]),
        ])
      );
    }
  } catch (err) {
    rowsEl.textContent = "錯誤: " + err.message;
  }
}

// ---- Register view ----

async function loadRegister() {
  const entriesEl = document.getElementById("register-entries");
  entriesEl.textContent = "loading…";
  try {
    const entries = await api("/api/register");
    entriesEl.innerHTML = "";
    for (const e of entries.slice().reverse()) {
      const running = e.running.map((a) => a.display).join(", ");
      entriesEl.appendChild(
        el("div", { class: "register-entry" }, [
          el("div", { class: "meta" }, [el("span", {}, [e.date]), el("span", {}, [e.account])]),
          el("div", { class: "main" }, [
            el("span", {}, [e.description]),
            el("span", { class: "amount " + signClass(e.amount.sign) }, [e.amount.display]),
          ]),
          el("div", { class: "meta" }, ["balance: " + running]),
        ])
      );
    }
  } catch (err) {
    entriesEl.textContent = "錯誤: " + err.message;
  }
}

// ---- Add view ----

function addPostingRow(account = "", amount = "") {
  const container = document.getElementById("postings");
  const row = el("div", { class: "posting-row" }, [
    el("input", { type: "text", placeholder: "account, e.g. expenses:food:coffee", value: account }),
    el("input", { type: "text", placeholder: "amount (blank = infer)", value: amount }),
  ]);
  container.appendChild(row);
}

function resetAddForm() {
  document.getElementById("add-date").valueAsDate = new Date();
  document.getElementById("add-desc").value = "";
  document.getElementById("postings").innerHTML = "";
  addPostingRow();
  addPostingRow();
  document.getElementById("add-status").textContent = "";
  document.getElementById("add-status").className = "status";
}

async function submitAddForm(ev) {
  ev.preventDefault();
  const status = document.getElementById("add-status");
  status.textContent = "";
  status.className = "status";

  const date = document.getElementById("add-date").value;
  const description = document.getElementById("add-desc").value.trim();
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
    resetAddForm();
    status.textContent = "已新增";
    status.className = "status ok";
    loadBalance();
  } catch (err) {
    status.textContent = "錯誤: " + err.message;
    status.className = "status error";
  }
}

// ---- Settings view ----

function loadSettingsForm() {
  document.getElementById("server-url").value = getServerUrl();
  document.getElementById("api-token").value = getToken();
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
  for (const btn of document.querySelectorAll("nav button")) {
    btn.classList.toggle("active", btn.dataset.view === name);
  }
  for (const view of document.querySelectorAll(".view")) {
    view.classList.toggle("active", view.id === "view-" + name);
  }
  if (name === "balance") loadBalance();
  if (name === "register") loadRegister();
  if (name === "add") resetAddForm();
  if (name === "settings") loadSettingsForm();
}

document.addEventListener("DOMContentLoaded", () => {
  for (const btn of document.querySelectorAll("nav button")) {
    btn.addEventListener("click", () => showView(btn.dataset.view));
  }
  document.getElementById("add-posting").addEventListener("click", () => addPostingRow());
  document.getElementById("add-form").addEventListener("submit", submitAddForm);
  document.getElementById("settings-form").addEventListener("submit", submitSettingsForm);

  showView("balance");
});
