(function () {
  const user = getUser();
  if (!user || user.role !== "admin") {
    location.href = "/login.html";
    return;
  }

  renderNav("admin");

  let propType = "";
  let userRole = "";
  let searchTimer = null;

  const msgEl = document.getElementById("admin-msg");
  const statsEl = document.getElementById("stats");
  const propList = document.getElementById("prop-list");
  const userList = document.getElementById("user-list");

  function showMsg(text, ok) {
    msgEl.className = ok ? "alert ok" : "alert";
    msgEl.textContent = text;
    if (ok) setTimeout(() => { msgEl.textContent = ""; msgEl.className = ""; }, 2500);
  }

  function fmtDate(iso) {
    if (!iso) return "—";
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return String(iso).slice(0, 10);
    return d.toLocaleDateString("en-IN", { day: "numeric", month: "short", year: "numeric" });
  }

  function isRecent(iso, days) {
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return false;
    return Date.now() - d.getTime() < days * 86400000;
  }

  async function loadStats() {
    const s = await api("/api/admin/stats");
    const by = s.properties_by_type || {};
    statsEl.innerHTML = `
      <div class="stat-tile"><div class="label">Properties</div><div class="value">${s.properties_total}</div>
        <div class="sub">R ${by.room || 0} · H ${by.home || 0} · PG ${by.pg || 0} · S ${by.shop || 0}</div></div>
      <div class="stat-tile"><div class="label">New today</div><div class="value">${s.properties_today}</div>
        <div class="sub">${s.properties_week} this week</div></div>
      <div class="stat-tile"><div class="label">Users</div><div class="value">${s.users_total}</div>
        <div class="sub">${s.owners} owners · ${s.seekers} seekers</div></div>
      <div class="stat-tile"><div class="label">Owners</div><div class="value">${s.owners}</div>
        <div class="sub">Can list properties</div></div>
      <div class="stat-tile"><div class="label">Seekers</div><div class="value">${s.seekers}</div>
        <div class="sub">Registered to book</div></div>
    `;
  }

  async function loadProperties() {
    const q = document.getElementById("prop-search").value.trim();
    const params = new URLSearchParams();
    if (propType) params.set("type", propType);
    if (q) params.set("q", q);
    const list = await api("/api/admin/properties?" + params.toString());
    if (!list.length) {
      propList.innerHTML = `<div class="admin-empty">No properties found.</div>`;
      return;
    }
    propList.innerHTML = `
      <table class="admin-table">
        <thead>
          <tr>
            <th>Property</th>
            <th>Type</th>
            <th>Owner</th>
            <th>Rent</th>
            <th>Added</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          ${list
            .map(
              (p) => `
            <tr data-id="${p.id}">
              <td class="title-cell">
                <a href="/property.html?id=${p.id}">${escapeHtml(p.title)}</a>
                ${isRecent(p.created_at, 7) ? `<span class="is-new">NEW</span>` : ""}
                <div class="muted" style="font-weight:400;font-size:0.82rem">${escapeHtml(locationLine(p))}</div>
              </td>
              <td><span class="badge badge-${p.type}">${typeLabel(p.type)}</span></td>
              <td>
                ${escapeHtml(p.owner_name || "—")}
                <div class="muted" style="font-size:0.8rem">${escapeHtml(p.owner_email || "")}</div>
              </td>
              <td>${formatRent(p.rent)}</td>
              <td>${fmtDate(p.created_at)}</td>
              <td class="row-actions">
                <a class="btn btn-ghost" href="/property.html?id=${p.id}">View</a>
                <button type="button" class="btn-danger" data-del-prop="${p.id}">Delete</button>
              </td>
            </tr>`
            )
            .join("")}
        </tbody>
      </table>`;
  }

  async function loadUsers() {
    const params = new URLSearchParams();
    if (userRole) params.set("role", userRole);
    const list = await api("/api/admin/users?" + params.toString());
    if (!list.length) {
      userList.innerHTML = `<div class="admin-empty">No users found.</div>`;
      return;
    }
    userList.innerHTML = `
      <table class="admin-table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Role</th>
            <th>Email</th>
            <th>Listings</th>
            <th>Joined</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          ${list
            .map(
              (u) => `
            <tr>
              <td class="title-cell">
                ${escapeHtml(u.name)}
                ${isRecent(u.created_at, 7) ? `<span class="is-new">NEW</span>` : ""}
              </td>
              <td><span class="badge badge-${u.role}">${u.role}</span></td>
              <td>${escapeHtml(u.email)}
                ${u.phone ? `<div class="muted" style="font-size:0.8rem">${escapeHtml(u.phone)}</div>` : ""}
              </td>
              <td>${u.listings}</td>
              <td>${fmtDate(u.created_at)}</td>
              <td class="row-actions">
                ${
                  u.role === "admin"
                    ? `<span class="muted">—</span>`
                    : `<button type="button" class="btn-danger" data-del-user="${u.id}">Delete</button>`
                }
              </td>
            </tr>`
            )
            .join("")}
        </tbody>
      </table>`;
  }

  function escapeHtml(s) {
    return String(s || "")
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");
  }

  async function refreshAll() {
    try {
      await loadStats();
      await loadProperties();
      await loadUsers();
    } catch (err) {
      if (err.status === 401 || err.status === 403) {
        clearSession();
        location.href = "/login.html";
        return;
      }
      showMsg(err.message || "Failed to load admin data", false);
    }
  }

  document.querySelectorAll(".admin-tab").forEach((btn) => {
    btn.addEventListener("click", () => {
      document.querySelectorAll(".admin-tab").forEach((b) => b.classList.remove("is-active"));
      btn.classList.add("is-active");
      const tab = btn.dataset.tab;
      document.getElementById("tab-properties").hidden = tab !== "properties";
      document.getElementById("tab-users").hidden = tab !== "users";
      document.getElementById("tab-properties").classList.toggle("is-active", tab === "properties");
      document.getElementById("tab-users").classList.toggle("is-active", tab === "users");
    });
  });

  document.querySelectorAll("#tab-properties .chip").forEach((chip) => {
    chip.addEventListener("click", () => {
      document.querySelectorAll("#tab-properties .chip").forEach((c) => c.classList.remove("is-active"));
      chip.classList.add("is-active");
      propType = chip.dataset.type || "";
      loadProperties().catch((e) => showMsg(e.message, false));
    });
  });

  document.querySelectorAll("#tab-users .chip").forEach((chip) => {
    chip.addEventListener("click", () => {
      document.querySelectorAll("#tab-users .chip").forEach((c) => c.classList.remove("is-active"));
      chip.classList.add("is-active");
      userRole = chip.dataset.role || "";
      loadUsers().catch((e) => showMsg(e.message, false));
    });
  });

  document.getElementById("prop-search").addEventListener("input", () => {
    clearTimeout(searchTimer);
    searchTimer = setTimeout(() => {
      loadProperties().catch((e) => showMsg(e.message, false));
    }, 280);
  });

  document.getElementById("refresh-btn").addEventListener("click", () => refreshAll());

  propList.addEventListener("click", async (e) => {
    const btn = e.target.closest("[data-del-prop]");
    if (!btn) return;
    const id = btn.dataset.delProp;
    if (!confirm("Delete this property permanently?")) return;
    try {
      await api(`/api/admin/properties/${id}`, { method: "DELETE" });
      showMsg("Property deleted", true);
      await refreshAll();
    } catch (err) {
      showMsg(err.message, false);
    }
  });

  userList.addEventListener("click", async (e) => {
    const btn = e.target.closest("[data-del-user]");
    if (!btn) return;
    const id = btn.dataset.delUser;
    if (!confirm("Delete this user and all their listings?")) return;
    try {
      await api(`/api/admin/users/${id}`, { method: "DELETE" });
      showMsg("User deleted", true);
      await refreshAll();
    } catch (err) {
      showMsg(err.message, false);
    }
  });

  refreshAll();
})();
