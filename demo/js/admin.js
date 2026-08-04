(function () {
  const nav = document.getElementById("domainNav");
  const content = document.getElementById("contentArea");
  const crumb = document.getElementById("crumb");
  let iamState = window.ERP_IAM.defaultState();
  let current = { domain: null, module: null };

  function statusTag(text) {
    const t = String(text);
    let cls = "tag";
    if (/已|合格|生效|正常|通过|启用|合作/.test(t)) cls += " ok";
    else if (/待|预警|预发|进行/.test(t)) cls += " warn";
    else if (/驳回|冻结|亏料/.test(t)) cls += " danger";
    else cls += " muted";
    return `<span class="${cls}">${t}</span>`;
  }

  function renderPanel(domain, module, liveRows) {
    const meta = window.ERP_MODULE_META[module] || {
      ...window.ERP_DEFAULT_META,
      desc: `${window.ERP_DEFAULT_META.desc}（${domain} · ${module}）`
    };
    const columns = liveRows && liveRows.columns ? liveRows.columns : meta.columns;
    const rows = liveRows && liveRows.rows ? liveRows.rows : meta.rows;
    const thead = columns.map((c) => `<th>${c}</th>`).join("");
    const tbody = rows
      .map((row) => {
        const cells = row
          .map((cell, i) => {
            const isStatus = /状态|预警|质检/.test(columns[i] || "");
            return `<td>${isStatus ? statusTag(cell) : cell}</td>`;
          })
          .join("");
        return `<tr>${cells}<td><button class="btn sm ghost">查看</button></td></tr>`;
      })
      .join("");

    const apiHint = liveRows
      ? `<div class="hint" style="border-color:#9fd4cb">已对接 API：${liveRows.path || ""} · ${rows.length} 条</div>`
      : `<div class="hint">演示数据（API 未返回列表时回退静态样例）</div>`;

    return `
      <div class="content-panel active" data-key="${domain}|${module}">
        <h2 class="page-title">${module}</h2>
        <p class="page-desc">${domain} → ${module}</p>
        <div class="hint">后台能力：${meta.desc}</div>
        ${apiHint}
        <div class="toolbar">
          <button class="btn">新建</button>
          <button class="btn ghost">导出</button>
          <button class="btn ghost">打印</button>
          <span class="spacer"></span>
          <input placeholder="多条件检索…" style="width:200px" />
          <button class="btn ghost">查询</button>
        </div>
        <div class="card" style="overflow:auto">
          <table class="data">
            <thead><tr>${thead}<th>操作</th></tr></thead>
            <tbody>${tbody}</tbody>
          </table>
        </div>
      </div>`;
  }

  function rowsFromApiList(list) {
    if (!Array.isArray(list) || !list.length) return { columns: ["提示"], rows: [["暂无数据"]] };
    const keys = Object.keys(list[0]).slice(0, 8);
    return {
      columns: keys,
      rows: list.map((item) => keys.map((k) => (item[k] == null ? "" : String(item[k]))))
    };
  }

  async function openModule(domain, module) {
    current = { domain, module };
    crumb.textContent = `${domain} / ${module}`;
    if (window.ERP_IAM.isIamModule(module)) {
      paintIam();
      document.querySelectorAll(".nav-domain .mods a").forEach((a) => {
        a.classList.toggle("active", a.dataset.domain === domain && a.dataset.module === module);
      });
      document.querySelectorAll(".nav-domain").forEach((el) => {
        el.classList.toggle("open", el.dataset.domain === domain);
      });
      return;
    }
    content.innerHTML = `<div class="content-panel active"><p class="page-desc">加载中…</p></div>`;
    let live = null;
    try {
      if (window.ERP_API && window.ERP_API.token()) {
        const path = window.ERP_API.DOMAIN_LIST[domain];
        const r = await window.ERP_API.listByDomain(domain);
        if (r && r.code === 1) {
          const list = (r.data && (r.data.list || r.data.widgets || r.data)) || [];
          const arr = Array.isArray(list) ? list : [];
          live = { ...rowsFromApiList(arr), path: path };
        }
      }
    } catch (e) {
      live = null;
    }
    content.innerHTML = renderPanel(domain, module, live);
    document.querySelectorAll(".nav-domain .mods a").forEach((a) => {
      a.classList.toggle("active", a.dataset.domain === domain && a.dataset.module === module);
    });
    document.querySelectorAll(".nav-domain").forEach((el) => {
      el.classList.toggle("open", el.dataset.domain === domain);
    });
  }

  async function ensureLogin() {
    const userEl = document.querySelector(".topbar .user");
    if (!window.ERP_API) return;
    if (!window.ERP_API.token()) {
      const r = await window.ERP_API.login("admin", "admin123");
      if (r.code !== 1) {
        if (userEl) userEl.textContent = "API 登录失败 · 使用静态演示";
        return;
      }
    }
    const me = await window.ERP_API.me();
    if (me.code === 1 && userEl) {
      const name = (me.data.user && (me.data.user.name || me.data.user.login_name)) || "admin";
      userEl.textContent = `已登录 API · ${name}`;
    }
  }

  function paintIam() {
    const result = window.ERP_IAM.renderByModule(current.module, iamState);
    if (!result) return false;
    iamState = result.state;
    content.innerHTML = result.html;
    window.ERP_IAM.bind(content, current.module, iamState, paintIam);
    return true;
  }

  function showHome() {
    current = { domain: null, module: null };
    crumb.textContent = "工作台";
    const domainCards = window.ERP_MENUS.map(
      (d) => `
      <div class="stat" style="cursor:pointer" data-open-domain="${d.domain}">
        <div class="label">${d.domain}</div>
        <div class="value" style="font-size:18px">${d.modules.length} 个模块</div>
      </div>`
    ).join("");

    content.innerHTML = `
      <div class="content-panel active">
        <h2 class="page-title">管理端工作台</h2>
        <p class="page-desc">菜单与第 3 章核心功能表一一对应，共 13 大域；二级菜单 = 功能模块。</p>
        <div class="stats">
          <div class="stat"><div class="label">今日生产任务</div><div class="value">12</div></div>
          <div class="stat"><div class="label">待办审批</div><div class="value">7</div></div>
          <div class="stat"><div class="label">亏料预警</div><div class="value">2</div></div>
          <div class="stat"><div class="label">待发货订单</div><div class="value">5</div></div>
        </div>
        <div class="card" style="padding:14px;margin-bottom:16px">
          <h3 style="margin:0 0 8px;font-size:14px">权限中心（ERP 能力）</h3>
          <p class="page-desc" style="margin-bottom:10px">
            管理员管理 / 管理员分组 / 角色管理 / 权限码 / 菜单裁剪 / 登录冻结 ——
            落在「人事管理·权限分配」与「系统管理·自定义权限」等表内模块，不另立平台模块名。
          </p>
          <div class="toolbar" style="margin:0">
            <button class="btn" data-go-iam="权限分配">打开权限分配</button>
            <button class="btn ghost" data-go-iam="自定义权限">自定义权限</button>
            <button class="btn ghost" data-go-iam="自定义菜单">自定义菜单</button>
            <button class="btn ghost" data-go-iam="登录控制">登录控制</button>
            <button class="btn ghost" data-go-iam="账户冻结">账户冻结</button>
          </div>
        </div>
        <div class="hint">产线映射：原料入库 → 清洗 → 去皮(计件领料) → 收货 → 切断 → 去芯 → 半成品入库 → 切块 → 过筛装袋 → 成品入库销售。功能落在生产管理 + 库存管理，不另立流程模块。</div>
        <h3 style="font-size:14px;margin:0 0 10px">十三大核心功能</h3>
        <div class="stats">${domainCards}</div>
      </div>`;

    content.querySelectorAll("[data-open-domain]").forEach((el) => {
      el.addEventListener("click", () => {
        const domain = el.getAttribute("data-open-domain");
        const first = window.ERP_MENUS.find((d) => d.domain === domain).modules[0];
        openModule(domain, first);
      });
    });
    content.querySelectorAll("[data-go-iam]").forEach((btn) => {
      btn.addEventListener("click", () => {
        const mod = btn.getAttribute("data-go-iam");
        const domain = mod === "权限分配" || mod === "成本隐藏" ? (mod === "成本隐藏" ? "生产管理" : "人事管理") : "系统管理";
        if (mod === "权限分配") iamState.tab = "users";
        openModule(domain, mod);
      });
    });
  }

  function buildNav() {
    nav.innerHTML = window.ERP_MENUS.map((d) => {
      const links = d.modules
        .map((m) => {
          const mark = window.ERP_IAM.isIamModule(m) ? ' style="color:#9fd4cb"' : "";
          return `<a href="#" data-domain="${d.domain}" data-module="${m}"${mark}>${m}</a>`;
        })
        .join("");
      return `<div class="nav-domain" data-domain="${d.domain}">
        <button type="button"><span>${d.domain}</span><span>▾</span></button>
        <div class="mods">${links}</div>
      </div>`;
    }).join("");

    nav.querySelectorAll(".nav-domain > button").forEach((btn) => {
      btn.addEventListener("click", () => {
        btn.parentElement.classList.toggle("open");
      });
    });

    nav.querySelectorAll(".mods a").forEach((a) => {
      a.addEventListener("click", (e) => {
        e.preventDefault();
        openModule(a.dataset.domain, a.dataset.module);
      });
    });
  }

  document.getElementById("btnHome").addEventListener("click", showHome);
  buildNav();
  showHome();
  const hr = nav.querySelector('.nav-domain[data-domain="人事管理"]');
  if (hr) hr.classList.add("open");
  ensureLogin();
})();
