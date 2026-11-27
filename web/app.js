const filesEl = document.querySelector("#files");
const messagesEl = document.querySelector("#messages");
const uploadForm = document.querySelector("#uploadForm");
const uploadState = document.querySelector("#uploadState");
const fileInput = document.querySelector("#fileInput");
const messageForm = document.querySelector("#messageForm");
const messageText = document.querySelector("#messageText");
const messageState = document.querySelector("#messageState");
const refreshBtn = document.querySelector("#refreshBtn");
const addressLine = document.querySelector("#addressLine");

let itemsETag;
let latestItemsRequest = 0;
let hasLoadedItems = false;
let pollTimer;

async function api(path, options) {
  return readResponse(await fetch(path, options));
}

async function readResponse(res) {
  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || `HTTP ${res.status}`);
  }
  if (res.status === 204) return null;
  return res.json();
}

async function copyText(value) {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(value);
      return true;
    } catch {
      // The browser can deny clipboard access even in a secure context.
    }
  }

  const focused = document.activeElement;
  const field = document.createElement("textarea");
  field.value = value;
  field.readOnly = true;
  field.style.cssText = "position:fixed;top:0;left:0;width:1px;height:1px;min-height:0;padding:0;border:0;opacity:0;pointer-events:none";
  document.body.append(field);
  try {
    field.focus({ preventScroll: true });
    field.select();
    field.setSelectionRange(0, field.value.length);
    return document.execCommand("copy");
  } catch {
    return false;
  } finally {
    field.remove();
    focused?.focus({ preventScroll: true });
  }
}

function formatBytes(bytes) {
  const units = ["B", "KB", "MB", "GB", "TB"];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  return `${value.toFixed(value >= 10 || unit === 0 ? 0 : 1)} ${units[unit]}`;
}

function formatTime(value) {
  return new Date(value).toLocaleString();
}

function empty(text) {
  const div = document.createElement("div");
  div.className = "empty";
  div.textContent = text;
  return div;
}

function renderFiles(files) {
  filesEl.replaceChildren();
  if (!files.length) {
    filesEl.append(empty("最近 24 小时没有文件"));
    return;
  }
  for (const file of files) {
    const item = document.createElement("div");
    item.className = "item";

    const body = document.createElement("div");
    const title = document.createElement("div");
    title.className = "title";
    title.textContent = file.name;
    const meta = document.createElement("div");
    meta.className = "meta";
    meta.textContent = `${formatBytes(file.size)} · ${formatTime(file.createdAt)}`;
    body.append(title, meta);

    const actions = document.createElement("div");
    actions.className = "actions";
    const download = document.createElement("a");
    download.href = `/api/files/${file.id}`;
    download.textContent = "下载";
    download.setAttribute("download", file.name);
    const remove = document.createElement("button");
    remove.type = "button";
    remove.className = "danger";
    remove.textContent = "删除";
    remove.addEventListener("click", async () => {
      await api(`/api/files/${file.id}`, { method: "DELETE" });
      await loadItems();
    });
    actions.append(download, remove);

    item.append(body, actions);
    filesEl.append(item);
  }
}

function renderMessages(messages) {
  messagesEl.replaceChildren();
  if (!messages.length) {
    messagesEl.append(empty("最近 24 小时没有消息"));
    return;
  }
  for (const message of messages) {
    const item = document.createElement("div");
    item.className = "item";

    const body = document.createElement("div");
    const text = document.createElement("div");
    text.className = "message-text";
    text.textContent = message.text;
    const meta = document.createElement("div");
    meta.className = "meta";
    meta.textContent = formatTime(message.createdAt);
    body.append(text, meta);

    const actions = document.createElement("div");
    actions.className = "actions";
    const copy = document.createElement("button");
    copy.type = "button";
    copy.textContent = "复制";
    let copyFeedbackTimer;
    copy.addEventListener("click", async () => {
      const copied = await copyText(message.text);
      copy.textContent = copied ? "已复制" : "复制失败";
      clearTimeout(copyFeedbackTimer);
      copyFeedbackTimer = setTimeout(() => { copy.textContent = "复制"; }, 1800);
      messageState.textContent = copied ? "已复制到剪贴板" : "复制失败，请选中消息文本后手动复制";
    });
    const remove = document.createElement("button");
    remove.type = "button";
    remove.className = "danger";
    remove.textContent = "删除";
    remove.addEventListener("click", async () => {
      await api(`/api/messages/${message.id}`, { method: "DELETE" });
      await loadItems();
    });
    actions.append(copy, remove);

    item.append(body, actions);
    messagesEl.append(item);
  }
}

async function loadHealth() {
  const health = await api("/api/health");
  const urls = health.urls || [];
  addressLine.textContent = urls.length ? `访问地址：${urls.join("  ")}` : "访问地址：当前设备";
}

async function loadItems() {
  const request = ++latestItemsRequest;
  const response = await fetch("/api/items", {
    headers: itemsETag ? { "If-None-Match": itemsETag } : undefined,
  });
  const etag = response.headers.get("ETag");
  if (response.status === 304 || (response.ok && etag && etag === itemsETag)) return;
  const items = await readResponse(response);
  if (request !== latestItemsRequest) return;
  renderFiles(items.files || []);
  renderMessages(items.messages || []);
  itemsETag = etag;
  hasLoadedItems = true;
}

async function refreshItems() {
  clearTimeout(pollTimer);
  try {
    await loadItems();
  } catch (err) {
    if (!hasLoadedItems) {
      filesEl.replaceChildren(empty(`加载失败：${err.message}`));
      messagesEl.replaceChildren(empty(`加载失败：${err.message}`));
    }
  } finally {
    clearTimeout(pollTimer);
    if (!document.hidden) pollTimer = setTimeout(refreshItems, 5000);
  }
}

function updatePolling() {
  if (document.hidden) clearTimeout(pollTimer);
  else refreshItems();
}

uploadForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  if (!fileInput.files.length) return;
  const form = new FormData();
  form.append("file", fileInput.files[0]);
  uploadState.textContent = "上传中...";
  try {
    await api("/api/files", { method: "POST", body: form });
    fileInput.value = "";
    uploadState.textContent = "上传完成";
    await loadItems();
  } catch (err) {
    uploadState.textContent = `上传失败：${err.message}`;
  }
});

messageForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  const text = messageText.value.trim();
  if (!text) return;
  await api("/api/messages", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ text }),
  });
  messageText.value = "";
  await loadItems();
});

refreshBtn.addEventListener("click", refreshItems);
document.addEventListener("visibilitychange", updatePolling);

loadHealth().catch(() => {
  addressLine.textContent = "访问地址加载失败";
});
updatePolling();

