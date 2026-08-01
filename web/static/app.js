// im-server 前端验收台
// 作者: wym
// 纯原生 JS，方便本地直接打开验收，不引入构建工具

const state = {
  token: localStorage.getItem("im_token") || "",
  userId: Number(localStorage.getItem("im_user_id") || 0),
  username: localStorage.getItem("im_username") || "",
  nickname: localStorage.getItem("im_nickname") || "",
  ws: null,
  mode: null, // 'peer' | 'group'
  peerId: 0,
  groupId: 0,
  authMode: "login",
};

const $ = (id) => document.getElementById(id);

function toast(text) {
  const el = $("toast");
  el.textContent = text;
  el.classList.remove("hidden");
  clearTimeout(toast._t);
  toast._t = setTimeout(() => el.classList.add("hidden"), 2200);
}

async function api(path, options = {}) {
  const headers = { "Content-Type": "application/json", ...(options.headers || {}) };
  if (state.token) headers.Authorization = `Bearer ${state.token}`;
  const res = await fetch(path, { ...options, headers });
  const data = await res.json().catch(() => ({}));
  if (!res.ok || (data.code !== undefined && data.code !== 0)) {
    throw new Error(data.message || `请求失败 HTTP ${res.status}`);
  }
  return data.data;
}

function saveSession(data) {
  state.token = data.token;
  state.userId = data.user_id;
  state.username = data.username;
  state.nickname = data.nickname || data.username;
  localStorage.setItem("im_token", state.token);
  localStorage.setItem("im_user_id", String(state.userId));
  localStorage.setItem("im_username", state.username);
  localStorage.setItem("im_nickname", state.nickname);
}

function clearSession() {
  state.token = "";
  state.userId = 0;
  localStorage.removeItem("im_token");
  localStorage.removeItem("im_user_id");
  localStorage.removeItem("im_username");
  localStorage.removeItem("im_nickname");
  if (state.ws) {
    state.ws.close();
    state.ws = null;
  }
}

function setWsStatus(ok, text) {
  $("ws-dot").className = `dot ${ok ? "on" : "off"}`;
  $("ws-text").textContent = text;
}

function showMain() {
  $("auth-view").classList.add("hidden");
  $("main-view").classList.remove("hidden");
  $("me-name").textContent = state.nickname || state.username;
  $("me-id").textContent = state.userId;
  connectWS();
  refreshGroups();
}

function showAuth() {
  $("main-view").classList.add("hidden");
  $("auth-view").classList.remove("hidden");
}

function appendSys(text) {
  const div = document.createElement("div");
  div.className = "sys";
  div.textContent = text;
  $("messages").appendChild(div);
  $("messages").scrollTop = $("messages").scrollHeight;
}

function appendBubble({ from, fromName, content, ts, me }) {
  const div = document.createElement("div");
  div.className = `bubble ${me ? "me" : ""}`;
  const time = ts ? new Date(ts * 1000).toLocaleTimeString() : "";
  div.innerHTML = `<div class="meta">${fromName || from || ""} · ${time}</div><div class="body"></div>`;
  div.querySelector(".body").textContent = content;
  $("messages").appendChild(div);
  $("messages").scrollTop = $("messages").scrollHeight;
}

function clearMessages() {
  $("messages").innerHTML = "";
}

function connectWS() {
  if (!state.token) return;
  if (state.ws && (state.ws.readyState === WebSocket.OPEN || state.ws.readyState === WebSocket.CONNECTING)) {
    return;
  }
  const proto = location.protocol === "https:" ? "wss" : "ws";
  const url = `${proto}://${location.host}/ws?token=${encodeURIComponent(state.token)}`;
  const ws = new WebSocket(url);
  state.ws = ws;
  setWsStatus(false, "连接中...");

  ws.onopen = () => {
    setWsStatus(true, "已连接");
    appendSys("WebSocket 已连接");
  };
  ws.onclose = () => {
    setWsStatus(false, "已断开");
    appendSys("WebSocket 已断开，3 秒后重连...");
    setTimeout(connectWS, 3000);
  };
  ws.onerror = () => setWsStatus(false, "连接异常");
  ws.onmessage = (ev) => {
    let msg;
    try {
      msg = JSON.parse(ev.data);
    } catch {
      return;
    }
    handleWSMessage(msg);
  };
}

function handleWSMessage(msg) {
  if (msg.type === "pong") return;
  if (msg.type === "ack") {
    toast(`ACK: ${msg.msg || "ok"}`);
    return;
  }
  if (msg.type === "error") {
    toast(msg.msg || "错误");
    appendSys(`错误: ${msg.msg}`);
    return;
  }
  if (msg.type === "chat") {
    // 只展示当前单聊会话
    if (state.mode === "peer") {
      const related =
        (msg.from === state.peerId && msg.to === state.userId) ||
        (msg.from === state.userId && msg.to === state.peerId);
      if (related || msg.from === state.peerId) {
        appendBubble({
          from: msg.from,
          fromName: msg.from_name,
          content: msg.content,
          ts: msg.ts,
          me: msg.from === state.userId,
        });
      } else {
        appendSys(`收到来自 ${msg.from_name || msg.from} 的单聊（非当前会话）`);
      }
    } else {
      appendSys(`收到单聊 ${msg.from_name || msg.from}: ${msg.content}`);
    }
    return;
  }
  if (msg.type === "group_chat") {
    if (state.mode === "group" && msg.group_id === state.groupId) {
      appendBubble({
        from: msg.from,
        fromName: msg.from_name,
        content: msg.content,
        ts: msg.ts,
        me: msg.from === state.userId,
      });
    } else {
      appendSys(`群 ${msg.group_id} · ${msg.from_name}: ${msg.content}`);
    }
  }
}

function sendWS(payload) {
  if (!state.ws || state.ws.readyState !== WebSocket.OPEN) {
    toast("WebSocket 未连接");
    return false;
  }
  state.ws.send(JSON.stringify(payload));
  return true;
}

async function openPeer(peerId) {
  state.mode = "peer";
  state.peerId = peerId;
  state.groupId = 0;
  $("session-label").textContent = `单聊 · 对方 ID ${peerId}`;
  clearMessages();
  appendSys(`加载与 ${peerId} 的历史...`);
  try {
    const data = await api(`/api/messages?peer_id=${peerId}&page=1&limit=50`);
    const list = (data.list || []).slice().reverse();
    list.forEach((m) => {
      appendBubble({
        from: m.from_user_id,
        fromName: m.from_username,
        content: m.content,
        ts: Math.floor(new Date(m.created_at).getTime() / 1000),
        me: m.from_user_id === state.userId,
      });
    });
    if (!list.length) appendSys("暂无历史消息，发一条试试");
  } catch (e) {
    appendSys(e.message);
  }
}

async function openGroup(groupId, name) {
  state.mode = "group";
  state.groupId = groupId;
  state.peerId = 0;
  $("session-label").textContent = `群聊 · ${name || ""} (#${groupId})`;
  clearMessages();
  document.querySelectorAll("#group-list li").forEach((li) => {
    li.classList.toggle("active", Number(li.dataset.id) === groupId);
  });
  try {
    const data = await api(`/api/groups/messages?group_id=${groupId}&page=1&limit=50`);
    const list = (data.list || []).slice().reverse();
    list.forEach((m) => {
      appendBubble({
        from: m.from_user_id,
        fromName: m.from_username,
        content: m.content,
        ts: Math.floor(new Date(m.created_at).getTime() / 1000),
        me: m.from_user_id === state.userId,
      });
    });
    if (!list.length) appendSys("暂无群消息");
  } catch (e) {
    appendSys(e.message);
  }
}

async function refreshGroups() {
  const ul = $("group-list");
  ul.innerHTML = "";
  try {
    const data = await api("/api/groups");
    (data.list || []).forEach((g) => {
      const li = document.createElement("li");
      li.textContent = `${g.name} (#${g.id})`;
      li.dataset.id = g.id;
      li.onclick = () => openGroup(g.id, g.name);
      ul.appendChild(li);
    });
    if (!(data.list || []).length) {
      const li = document.createElement("li");
      li.textContent = "还没有群";
      li.style.cursor = "default";
      ul.appendChild(li);
    }
  } catch (e) {
    toast(e.message);
  }
}

function bindUI() {
  document.querySelectorAll(".tab").forEach((btn) => {
    btn.onclick = () => {
      document.querySelectorAll(".tab").forEach((b) => b.classList.remove("active"));
      btn.classList.add("active");
      state.authMode = btn.dataset.tab;
      $("nickname-wrap").classList.toggle("hidden", state.authMode !== "register");
      $("auth-submit").textContent = state.authMode === "login" ? "登录" : "注册并登录";
      $("auth-msg").textContent = "";
    };
  });

  $("auth-form").onsubmit = async (e) => {
    e.preventDefault();
    const username = $("username").value.trim();
    const password = $("password").value;
    const nickname = $("nickname").value.trim();
    $("auth-msg").textContent = "";
    try {
      if (state.authMode === "register") {
        await api("/api/register", {
          method: "POST",
          body: JSON.stringify({ username, password, nickname }),
        });
      }
      const data = await api("/api/login", {
        method: "POST",
        body: JSON.stringify({ username, password }),
      });
      saveSession(data);
      showMain();
    } catch (err) {
      $("auth-msg").textContent = err.message;
    }
  };

  $("btn-logout").onclick = () => {
    clearSession();
    showAuth();
  };

  $("btn-online").onclick = async () => {
    try {
      const data = await api("/api/online");
      toast(`在线: ${(data.online_user_ids || []).join(", ") || "无"}`);
    } catch (e) {
      toast(e.message);
    }
  };

  $("btn-open-peer").onclick = () => {
    const id = Number($("peer-id").value);
    if (!id || id === state.userId) {
      toast("请输入有效的对方 user_id");
      return;
    }
    openPeer(id);
  };

  $("btn-create-group").onclick = async () => {
    const name = $("group-name").value.trim();
    const members = $("group-members").value
      .split(",")
      .map((s) => Number(s.trim()))
      .filter((n) => n > 0);
    if (!name) {
      toast("请填写群名");
      return;
    }
    try {
      const g = await api("/api/groups", {
        method: "POST",
        body: JSON.stringify({ name, member_ids: members }),
      });
      toast(`已创建群 #${g.id}`);
      $("group-name").value = "";
      $("group-members").value = "";
      await refreshGroups();
      openGroup(g.id, g.name);
    } catch (e) {
      toast(e.message);
    }
  };

  $("send-form").onsubmit = (e) => {
    e.preventDefault();
    const content = $("content").value.trim();
    if (!content) return;
    if (state.mode === "peer" && state.peerId) {
      if (!sendWS({ type: "chat", to: state.peerId, content })) return;
      appendBubble({
        from: state.userId,
        fromName: state.username,
        content,
        ts: Math.floor(Date.now() / 1000),
        me: true,
      });
      $("content").value = "";
      return;
    }
    if (state.mode === "group" && state.groupId) {
      if (!sendWS({ type: "group_chat", group_id: state.groupId, content })) return;
      appendBubble({
        from: state.userId,
        fromName: state.username,
        content,
        ts: Math.floor(Date.now() / 1000),
        me: true,
      });
      $("content").value = "";
      return;
    }
    toast("请先选择单聊对象或群");
  };
}

bindUI();
if (state.token && state.userId) {
  showMain();
} else {
  showAuth();
}
