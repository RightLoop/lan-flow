const filesEl = document.querySelector("#files");
const messagesEl = document.querySelector("#messages");
const uploadForm = document.querySelector("#uploadForm");
const uploadState = document.querySelector("#uploadState");
const fileInput = document.querySelector("#fileInput");
const messageForm = document.querySelector("#messageForm");
const messageText = document.querySelector("#messageText");
const refreshBtn = document.querySelector("#refreshBtn");
const addressLine = document.querySelector("#addressLine");

async function api(path, options) {
  const res = await fetch(path, options);
  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || `HTTP ${res.status}`);
  }
  if (res.status === 204) return null;
  return res.json();
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
    copy.addEventListener("click", async () => {
      await navigator.clipboard.writeText(message.text);
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
  const items = await api("/api/items");
  renderFiles(items.files || []);
  renderMessages(items.messages || []);
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

refreshBtn.addEventListener("click", loadItems);

loadHealth().catch(() => {
  addressLine.textContent = "访问地址加载失败";
});
loadItems().catch((err) => {
  filesEl.append(empty(`加载失败：${err.message}`));
  messagesEl.append(empty(`加载失败：${err.message}`));
});
setInterval(loadItems, 5000);

