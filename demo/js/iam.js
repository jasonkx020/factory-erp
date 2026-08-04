/**
 * 权限中心展示 —— 严格落在文档第 7 章：
 * 人事管理·权限分配 / 系统管理·自定义权限·自定义菜单·登录控制·账户冻结
 * 「管理员管理 / 管理员分组 / 角色管理」作为权限分配内能力展开，不另立核心功能名
 */
window.ERP_IAM = (function () {
  const ROLES = [
    { code: "sys_admin", name: "系统管理员", group: "平台管理组", scope: "全部", menus: "系统管理全模块 + 权限分配", users: 2 },
    { code: "boss", name: "老板", group: "经营决策组", scope: "全部", menus: "统计报表/财务只读/审批", users: 1 },
    { code: "sales", name: "销售员", group: "业务作业组", scope: "本人", menus: "销售+客户(受保护/隐藏)", users: 8 },
    { code: "purchase", name: "采购员", group: "业务作业组", scope: "全部", menus: "采购+采购类审批", users: 3 },
    { code: "warehouse", name: "仓管员", group: "仓储生产组", scope: "本仓库", menus: "库存(按仓)+采购入库协同", users: 4 },
    { code: "planner", name: "生产计划", group: "仓储生产组", scope: "全部", menus: "任务/派工/工艺/MRP/BOM", users: 2 },
    { code: "foreman", name: "车间主任", group: "仓储生产组", scope: "本车间", menus: "工作台/派工/进度/质检/灵活派发", users: 3 },
    { code: "piece", name: "计件工", group: "一线作业组", scope: "本人", menus: "报工/领料/本人工资只读", users: 46 },
    { code: "fixed", name: "固定工", group: "一线作业组", scope: "本班组", menus: "授权工序报工/收货质检", users: 18 },
    { code: "qc", name: "质检员", group: "仓储生产组", scope: "全部", menus: "来料/入库/过程质检+报表", users: 3 },
    { code: "hr", name: "人事", group: "职能管理组", scope: "全部", menus: "人事+考勤审批", users: 2 },
    { code: "payroll", name: "薪资员", group: "职能管理组", scope: "全部", menus: "工资+批量核算工资", users: 1 },
    { code: "finance", name: "财务", group: "职能管理组", scope: "全部", menus: "财务+财审管控范围", users: 3 }
  ];

  const GROUPS = [
    { code: "g_platform", name: "平台管理组", desc: "系统管理员等高权限后台账号", roles: ["系统管理员"], admins: 2 },
    { code: "g_biz", name: "经营决策组", desc: "老板驾驶舱与经营只读", roles: ["老板"], admins: 1 },
    { code: "g_ops", name: "业务作业组", desc: "销售/采购等业务内勤与外勤", roles: ["销售员", "采购员"], admins: 11 },
    { code: "g_plant", name: "仓储生产组", desc: "仓管/计划/车间/质检", roles: ["仓管员", "生产计划", "车间主任", "质检员"], admins: 12 },
    { code: "g_line", name: "一线作业组", desc: "计件工/固定工（前台为主）", roles: ["计件工", "固定工"], admins: 64 },
    { code: "g_func", name: "职能管理组", desc: "人事/薪资/财务", roles: ["人事", "薪资员", "财务"], admins: 6 }
  ];

  const USERS = [
    { login: "admin", name: "系统管理员", emp: "E0001", roles: ["系统管理员"], group: "平台管理组", scope: "全部", warehouse: "全部仓", process: "—", status: "正常", type: "管理员" },
    { login: "boss01", name: "林总", emp: "E0002", roles: ["老板"], group: "经营决策组", scope: "全部", warehouse: "—", process: "—", status: "正常", type: "管理员" },
    { login: "hr_li", name: "李人事", emp: "E0011", roles: ["人事"], group: "职能管理组", scope: "全部", warehouse: "—", process: "—", status: "正常", type: "管理员" },
    { login: "fin_zhao", name: "赵财务", emp: "E0012", roles: ["财务"], group: "职能管理组", scope: "全部", warehouse: "—", process: "—", status: "正常", type: "管理员" },
    { login: "pay_sun", name: "孙薪资", emp: "E0013", roles: ["薪资员"], group: "职能管理组", scope: "全部", warehouse: "—", process: "—", status: "正常", type: "管理员" },
    { login: "sales_zhang", name: "张敏", emp: "E0101", roles: ["销售员"], group: "业务作业组", scope: "本人", warehouse: "—", process: "—", status: "正常", type: "业务用户" },
    { login: "pur_zhou", name: "周采购", emp: "E0102", roles: ["采购员"], group: "业务作业组", scope: "全部", warehouse: "原料仓", process: "—", status: "正常", type: "业务用户" },
    { login: "wh_wu", name: "吴仓管", emp: "E0201", roles: ["仓管员"], group: "仓储生产组", scope: "本仓库", warehouse: "原料仓,半成品仓", process: "—", status: "正常", type: "业务用户" },
    { login: "plan_chen", name: "陈计划", emp: "E0202", roles: ["生产计划"], group: "仓储生产组", scope: "全部", warehouse: "全部仓", process: "全部·可派工", status: "正常", type: "管理员" },
    { login: "wang_zr", name: "王主任", emp: "E0203", roles: ["车间主任"], group: "仓储生产组", scope: "本车间", warehouse: "原料仓,半成品仓", process: "全部·可派工", status: "正常", type: "管理员" },
    { login: "qc_he", name: "何质检", emp: "E0204", roles: ["质检员"], group: "仓储生产组", scope: "全部", warehouse: "原料仓,成品仓", process: "—", status: "正常", type: "业务用户" },
    { login: "chen_jp", name: "陈某", emp: "E0301", roles: ["计件工"], group: "一线作业组", scope: "本人", warehouse: "原料仓", process: "去皮/去芯/切块·可报工", status: "正常", type: "一线用户" },
    { login: "liu_gd", name: "刘某", emp: "E0302", roles: ["固定工"], group: "一线作业组", scope: "本班组", warehouse: "半成品仓", process: "收货/切断·可报工", status: "正常", type: "一线用户" },
    { login: "old_user", name: "已离职试号", emp: "E0999", roles: [], group: "—", scope: "—", warehouse: "—", process: "—", status: "冻结", type: "已收回" }
  ];

  const ACTIONS = ["查看", "新增", "编辑", "删除", "审批", "导出", "打印"];

  const IAM_MODULES = new Set([
    "权限分配", "自定义权限", "自定义菜单", "登录控制", "账户冻结", "成本隐藏"
  ]);

  function tag(text) {
    const t = String(text);
    let cls = "tag";
    if (/正常|启用|已授权|可见/.test(t)) cls += " ok";
    else if (/冻结|禁用|隐藏|已收回/.test(t)) cls += " danger";
    else if (/待|部分/.test(t)) cls += " warn";
    else cls += " muted";
    return `<span class="${cls}">${t}</span>`;
  }

  function shell(title, domain, hint, tabsHtml, bodyHtml) {
    return `
      <div class="content-panel active">
        <h2 class="page-title">${title}</h2>
        <p class="page-desc">${domain} → 权限能力（文档第 7 章展开，不新增核心功能名）</p>
        <div class="hint">${hint}</div>
        <div class="iam-tabs">${tabsHtml}</div>
        <div id="iamBody">${bodyHtml}</div>
      </div>`;
  }

  function tabs(items, active) {
    return items
      .map(
        (t) =>
          `<button type="button" data-iam-tab="${t.id}" class="${t.id === active ? "active" : ""}">${t.label}</button>`
      )
      .join("");
  }

  /* —— 管理员 / 用户管理 —— */
  function viewUsers(filterType) {
    const rows = USERS.filter((u) => !filterType || u.type === filterType || (filterType === "管理员" && u.type === "管理员"));
    return `
      <div class="toolbar">
        <button class="btn">新建用户/管理员</button>
        <button class="btn ghost">重置密码</button>
        <button class="btn ghost">批量赋角色</button>
        <button class="btn warn">冻结账号</button>
        <span class="spacer"></span>
        <select id="userTypeFilter">
          <option value="">全部类型</option>
          <option value="管理员" ${filterType === "管理员" ? "selected" : ""}>管理员</option>
          <option value="业务用户">业务用户</option>
          <option value="一线用户">一线用户</option>
        </select>
        <input placeholder="登录名/姓名/工号" style="width:180px" />
        <button class="btn ghost">查询</button>
      </div>
      <div class="stats">
        <div class="stat"><div class="label">管理员</div><div class="value">${USERS.filter((u) => u.type === "管理员").length}</div></div>
        <div class="stat"><div class="label">业务用户</div><div class="value">${USERS.filter((u) => u.type === "业务用户").length}</div></div>
        <div class="stat"><div class="label">一线用户</div><div class="value">${USERS.filter((u) => u.type === "一线用户").length}</div></div>
        <div class="stat"><div class="label">冻结</div><div class="value">${USERS.filter((u) => u.status === "冻结").length}</div></div>
      </div>
      <div class="card" style="overflow:auto">
        <table class="data">
          <thead>
            <tr>
              <th>登录名</th><th>姓名</th><th>工号</th><th>类型</th><th>管理员分组</th>
              <th>角色(并集)</th><th>数据范围</th><th>仓范围</th><th>工序范围</th><th>状态</th><th>操作</th>
            </tr>
          </thead>
          <tbody>
            ${rows
              .map(
                (u) => `<tr>
                  <td>${u.login}</td>
                  <td>${u.name}</td>
                  <td>${u.emp}</td>
                  <td>${u.type}</td>
                  <td>${u.group}</td>
                  <td>${u.roles.join("、") || "—"}</td>
                  <td>${u.scope}</td>
                  <td>${u.warehouse}</td>
                  <td>${u.process}</td>
                  <td>${tag(u.status)}</td>
                  <td>
                    <button class="btn sm ghost" data-iam-action="edit-user" data-login="${u.login}">授权</button>
                    <button class="btn sm ghost">编辑</button>
                  </td>
                </tr>`
              )
              .join("")}
          </tbody>
        </table>
      </div>
      <p class="page-desc" style="margin-top:10px">对应实体：User / UserRole / RoleWarehouseScope / RoleProcessScope；入职赋权、离职收回联动 Onboard / Offboard。</p>`;
  }

  /* —— 管理员分组 —— */
  function viewGroups(selected) {
    const g = GROUPS.find((x) => x.code === selected) || GROUPS[0];
    return `
      <div class="iam-layout">
        <div class="iam-side">
          <h4>管理员分组</h4>
          ${GROUPS.map(
            (x) =>
              `<button type="button" class="iam-item ${x.code === g.code ? "active" : ""}" data-iam-group="${x.code}">
                ${x.name}<small>${x.admins} 人 · ${x.roles.length} 角色</small>
              </button>`
          ).join("")}
          <div class="split-actions">
            <button class="btn sm">新建分组</button>
          </div>
        </div>
        <div class="iam-main">
          <h3 style="margin:0 0 6px;font-size:16px">${g.name}</h3>
          <p class="page-desc">${g.desc}</p>
          <div class="iam-badge-row">
            ${g.roles.map((r) => `<span class="tag">${r}</span>`).join("")}
          </div>
          <div class="toolbar">
            <button class="btn">绑定角色</button>
            <button class="btn ghost">添加管理员</button>
            <button class="btn ghost">调整排序</button>
          </div>
          <table class="data">
            <thead><tr><th>登录名</th><th>姓名</th><th>角色</th><th>状态</th><th>操作</th></tr></thead>
            <tbody>
              ${USERS.filter((u) => u.group === g.name)
                .map(
                  (u) => `<tr>
                    <td>${u.login}</td><td>${u.name}</td>
                    <td>${u.roles.join("、")}</td><td>${tag(u.status)}</td>
                    <td><button class="btn sm ghost">移出分组</button></td>
                  </tr>`
                )
                .join("") || `<tr><td colspan="5" style="color:#5c6b75">暂无成员</td></tr>`}
            </tbody>
          </table>
          <div class="hint" style="margin-top:12px">
            管理员分组用于后台账号归类与批量授权，逻辑归属「权限分配」，实体可落 AdminGroup / AdminGroupRole / AdminGroupUser（见逻辑模型补充），不新增第 3 章模块名。
          </div>
        </div>
      </div>`;
  }

  /* —— 角色管理 —— */
  function viewRoles(selected) {
    const role = ROLES.find((r) => r.code === selected) || ROLES[0];
    return `
      <div class="iam-layout three">
        <div class="iam-side">
          <h4>预置角色（文档 7.3）</h4>
          ${ROLES.map(
            (r) =>
              `<button type="button" class="iam-item ${r.code === role.code ? "active" : ""}" data-iam-role="${r.code}">
                ${r.name}<small>${r.group} · ${r.users} 人</small>
              </button>`
          ).join("")}
          <div class="split-actions"><button class="btn sm">新建角色</button></div>
        </div>
        <div class="iam-main">
          <div class="toolbar">
            <strong>${role.name}</strong>
            <span class="tag">${role.code}</span>
            <span class="spacer"></span>
            <button class="btn">保存角色</button>
            <button class="btn ghost">复制角色</button>
          </div>
          <div class="scope-grid">
            <div class="scope-box">
              <h5>基本信息</h5>
              <div style="font-size:12px;line-height:1.8">
                名称：${role.name}<br/>
                分组：${role.group}<br/>
                默认数据范围：<strong>${role.scope}</strong><br/>
                授权摘要：${role.menus}
              </div>
            </div>
            <div class="scope-box">
              <h5>数据范围 DataScope</h5>
              ${["本人", "本班组", "本车间", "本仓库", "全部"]
                .map(
                  (s) =>
                    `<label><input type="radio" name="scope" ${s === role.scope ? "checked" : ""}/> ${s}</label>`
                )
                .join("")}
            </div>
            <div class="scope-box">
              <h5>仓范围</h5>
              ${["原料仓", "半成品仓", "成品仓", "废料仓"]
                .map((w) => `<label><input type="checkbox" checked/> ${w}</label>`)
                .join("")}
            </div>
            <div class="scope-box">
              <h5>工序范围</h5>
              ${["去皮", "收货卡点", "切断", "去芯", "切块", "过筛装袋"]
                .map(
                  (p, i) =>
                    `<label><input type="checkbox" ${i < 4 ? "checked" : ""}/> ${p}
                    <span style="color:#8a9aa3">可报工</span>
                    <input type="checkbox" ${role.code === "foreman" || role.code === "planner" ? "checked" : ""}/> 可派工</label>`
                )
                .join("")}
            </div>
          </div>
          <h4 style="margin:16px 0 8px;font-size:13px;color:#5c6b75">功能模块授权（勾选第 3 章模块 = 菜单+按钮入口）</h4>
          <div class="perm-tree" style="max-height:280px;overflow:auto">
            ${window.ERP_MENUS.map(
              (d) => `
              <details class="perm-domain" ${["系统管理", "人事管理", "生产管理", "库存管理", "工资管理"].includes(d.domain) ? "open" : ""}>
                <summary>${d.domain}
                  <span class="tag muted" style="margin-left:8px;font-weight:400">${d.modules.length}</span>
                </summary>
                <div class="perm-mods">
                  ${d.modules
                    .slice(0, 8)
                    .map(
                      (m) => `
                    <div class="perm-mod">
                      <input type="checkbox" ${shouldCheckModule(role, d.domain, m) ? "checked" : ""}/>
                      <div>
                        <div>${m}</div>
                        <div class="perm-actions">
                          ${ACTIONS.map((a) => `<label><input type="checkbox" ${shouldCheckModule(role, d.domain, m) ? "checked" : ""}/> ${a}</label>`).join("")}
                        </div>
                      </div>
                    </div>`
                    )
                    .join("")}
                  ${d.modules.length > 8 ? `<div style="font-size:12px;color:#5c6b75">… 共 ${d.modules.length} 个模块，演示折叠显示</div>` : ""}
                </div>
              </details>`
            ).join("")}
          </div>
        </div>
        <div class="iam-detail">
          <h4>已绑定用户</h4>
          ${USERS.filter((u) => u.roles.includes(role.name))
            .map((u) => `<div class="iam-item" style="cursor:default">${u.name}<small>${u.login} · ${tag(u.status)}</small></div>`)
            .join("") || "<p style='font-size:12px;color:#5c6b75'>无</p>"}
          <h4 style="margin-top:16px">产线默认映射（7.4）</h4>
          <p style="font-size:12px;color:#5c6b75;line-height:1.7">${lineMapping(role.name)}</p>
        </div>
      </div>`;
  }

  function shouldCheckModule(role, domain, module) {
    const map = {
      系统管理员: () => domain === "系统管理" || module === "权限分配",
      老板: () => domain === "统计报表" || domain === "审批管理" || domain === "财务管理",
      销售员: () => domain === "销售管理" || domain === "客户管理",
      采购员: () => domain === "采购管理" || module.includes("采购"),
      仓管员: () => domain === "库存管理" || module === "采购入库" || module === "采购退货",
      生产计划: () => domain === "生产管理",
      车间主任: () =>
        ["车间工作台", "生产派工", "灵活派发工单", "进度跟踪", "质检管理", "返修单", "废料管理"].includes(module),
      计件工: () => ["扫码报工", "联动式领料", "工人信息管理"].includes(module) || module === "工序工资",
      固定工: () => ["扫码报工", "质检管理"].includes(module),
      质检员: () => /质检|来料/.test(module),
      人事: () => domain === "人事管理" || module === "考勤审批",
      薪资员: () => domain === "工资管理" || module === "批量核算工资",
      财务: () => domain === "财务管理" || module === "财审管控"
    };
    const fn = map[role.name];
    return fn ? !!fn() : false;
  }

  function lineMapping(name) {
    const m = {
      仓管员: "原料/成品入出库 → 库存管理相关模块",
      计件工: "计件报工/领料 → 扫码报工、联动式领料、计件工资",
      固定工: "收货/切断 → 扫码报工、质检管理",
      车间主任: "派工调度 + 领料协同",
      财务: "看成本 → 成本预算/成本核算（成本隐藏对其关闭）",
      薪资员: "算薪 → 工序工资、薪酬核算、工资批量管理、批量核算工资",
      老板: "看成本（成本隐藏关闭）+ 驾驶舱只读"
    };
    return m[name] || "按角色勾选模块授权；前台仅开放表内作业子集。";
  }

  /* —— 用户授权明细 —— */
  function viewAssign(login) {
    const u = USERS.find((x) => x.login === login) || USERS[0];
    return `
      <div class="iam-layout">
        <div class="iam-side">
          <h4>选择用户</h4>
          ${USERS.map(
            (x) =>
              `<button type="button" class="iam-item ${x.login === u.login ? "active" : ""}" data-iam-assign="${x.login}">
                ${x.name}<small>${x.login} · ${x.type}</small>
              </button>`
          ).join("")}
        </div>
        <div class="iam-main">
          <h3 style="margin:0 0 8px;font-size:16px">授权：${u.name}</h3>
          <div class="hint">一人多角色，权限取并集。数据范围 / 仓范围 / 工序范围可在角色默认值上再收紧。</div>
          <div class="scope-grid">
            <div class="scope-box" style="grid-column:1/-1">
              <h5>绑定角色（多选）</h5>
              <div style="display:flex;flex-wrap:wrap;gap:8px">
                ${ROLES.map(
                  (r) =>
                    `<label style="min-width:120px"><input type="checkbox" ${u.roles.includes(r.name) ? "checked" : ""}/> ${r.name}</label>`
                ).join("")}
              </div>
            </div>
            <div class="scope-box">
              <h5>数据范围</h5>
              ${["本人", "本班组", "本车间", "本仓库", "全部"]
                .map((s) => `<label><input type="radio" name="uscope" ${u.scope === s ? "checked" : ""}/> ${s}</label>`)
                .join("")}
            </div>
            <div class="scope-box">
              <h5>仓范围</h5>
              ${["原料仓", "半成品仓", "成品仓"]
                .map(
                  (w) =>
                    `<label><input type="checkbox" ${String(u.warehouse).includes(w) || u.warehouse === "全部仓" ? "checked" : ""}/> ${w}</label>`
                )
                .join("")}
            </div>
            <div class="scope-box">
              <h5>工序范围</h5>
              ${["去皮", "去芯", "切块", "收货卡点", "切断"]
                .map(
                  (p) =>
                    `<label><input type="checkbox" ${String(u.process).includes(p.split("卡")[0]) || String(u.process).includes(p) ? "checked" : ""}/> ${p}</label>`
                )
                .join("")}
            </div>
          </div>
          <div class="split-actions">
            <button class="btn">保存授权</button>
            <button class="btn ghost">同步入职赋权模板</button>
            <button class="btn warn">按离职登记收回</button>
          </div>
        </div>
      </div>`;
  }

  /* —— 自定义权限 · 权限码 + 字段策略 —— */
  function viewPermCodes() {
    const samples = [];
    ["生产管理", "工资管理", "财务管理", "库存管理", "人事管理"].forEach((domain) => {
      const d = window.ERP_MENUS.find((x) => x.domain === domain);
      (d?.modules || []).slice(0, 3).forEach((m) => {
        ACTIONS.slice(0, 4).forEach((a) => {
          samples.push({ code: `${domain}:${m}:${a}`, domain, module: m, action: a });
        });
      });
    });
    return `
      <div class="toolbar">
        <button class="btn">同步生成权限码</button>
        <button class="btn ghost">按角色批量授权</button>
        <span class="spacer"></span>
        <input placeholder="搜索 核心功能:功能模块:动作" style="width:260px" />
      </div>
      <div class="hint">权限码格式严格为「核心功能:功能模块:动作」，例：生产管理:扫码报工:新增。与审批流程设定、财审管控、单据锁定联动拦截。</div>
      <div class="card" style="overflow:auto;max-height:360px">
        <table class="data">
          <thead><tr><th>权限码</th><th>域</th><th>模块</th><th>动作</th><th>系统管理员</th><th>车间主任</th><th>计件工</th><th>财务</th></tr></thead>
          <tbody>
            ${samples
              .slice(0, 24)
              .map((s) => {
                const piece = s.module === "扫码报工" || s.module === "联动式领料";
                const finance = s.domain === "财务管理";
                const foreman = s.domain === "生产管理";
                return `<tr>
                  <td><code style="font-size:12px">${s.code}</code></td>
                  <td>${s.domain}</td><td>${s.module}</td><td>${s.action}</td>
                  <td class="check-on">✓</td>
                  <td class="${foreman ? "check-on" : "check-off"}">${foreman ? "✓" : "—"}</td>
                  <td class="${piece && s.action !== "删除" ? "check-on" : "check-off"}">${piece && s.action !== "删除" ? "✓" : "—"}</td>
                  <td class="${finance ? "check-on" : "check-off"}">${finance ? "✓" : "—"}</td>
                </tr>`;
              })
              .join("")}
          </tbody>
        </table>
      </div>
      <h3 style="font-size:14px;margin:18px 0 8px">字段策略 · 成本隐藏协同</h3>
      <div class="card" style="overflow:auto">
        <table class="data field-matrix">
          <thead><tr><th>字段</th><th>计件工</th><th>固定工</th><th>车间主任</th><th>财务</th><th>老板</th><th>说明</th></tr></thead>
          <tbody>
            <tr><td>成本价 / 成本预算</td><td>${tag("隐藏")}</td><td>${tag("隐藏")}</td><td>${tag("隐藏")}</td><td>${tag("可见")}</td><td>${tag("可见")}</td><td>成本隐藏策略</td></tr>
            <tr><td>毛利</td><td>${tag("隐藏")}</td><td>${tag("隐藏")}</td><td>${tag("隐藏")}</td><td>${tag("可见")}</td><td>${tag("可见")}</td><td>字段策略 FieldPolicy</td></tr>
            <tr><td>他人工资</td><td>${tag("隐藏")}</td><td>${tag("隐藏")}</td><td>${tag("隐藏")}</td><td>${tag("可见")}</td><td>${tag("可见")}</td><td>仅本人计件可见</td></tr>
            <tr><td>本人工资/计件</td><td>${tag("可见")}</td><td>${tag("可见")}</td><td>${tag("可见")}</td><td>${tag("可见")}</td><td>${tag("可见")}</td><td>工人端只读</td></tr>
            <tr><td>供应商采购价</td><td>${tag("隐藏")}</td><td>${tag("隐藏")}</td><td>${tag("隐藏")}</td><td>${tag("可见")}</td><td>${tag("可见")}</td><td>采购/财务</td></tr>
          </tbody>
        </table>
      </div>`;
  }

  /* —— 自定义菜单 —— */
  function viewMenus(roleCode) {
    const role = ROLES.find((r) => r.code === roleCode) || ROLES.find((r) => r.code === "piece");
    return `
      <div class="iam-layout">
        <div class="iam-side">
          <h4>按角色裁剪菜单</h4>
          ${ROLES.map(
            (r) =>
              `<button type="button" class="iam-item ${r.code === role.code ? "active" : ""}" data-iam-menu-role="${r.code}">
                ${r.name}<small>${r.menus}</small>
              </button>`
          ).join("")}
        </div>
        <div class="iam-main">
          <div class="toolbar">
            <strong>自定义菜单 · ${role.name}</strong>
            <span class="spacer"></span>
            <button class="btn">保存菜单裁剪</button>
          </div>
          <div class="hint">按角色裁剪 13 大域菜单树；前台小程序/车间台另按端形态过滤表内子集。</div>
          ${window.ERP_MENUS.map(
            (d) => `
            <details class="perm-domain" open>
              <summary>
                <label style="display:inline-flex;align-items:center;gap:6px" onclick="event.stopPropagation()">
                  <input type="checkbox" ${shouldCheckModule(role, d.domain, d.modules[0]) || shouldCheckModule(role, d.domain, "") ? "checked" : ""}/>
                  ${d.domain}
                </label>
              </summary>
              <div class="perm-mods">
                ${d.modules
                  .map(
                    (m) =>
                      `<label style="display:flex;align-items:center;gap:8px;padding:4px 0;font-size:13px">
                        <input type="checkbox" ${shouldCheckModule(role, d.domain, m) ? "checked" : ""}/>
                        <span style="flex:1">${m}</span>
                        <input type="number" value="${shouldCheckModule(role, d.domain, m) ? 10 : 99}" style="width:56px" title="排序"/>
                      </label>`
                  )
                  .join("")}
              </div>
            </details>`
          ).join("")}
        </div>
      </div>`;
  }

  /* —— 登录控制 / 账户冻结 —— */
  function viewLogin() {
    return `
      <div class="scope-grid">
        <div class="scope-box">
          <h5>登录控制 LoginPolicy</h5>
          <label>失败锁定阈值 <input type="number" value="5" style="width:64px"/> 次</label>
          <label>锁定时长 <input type="number" value="30" style="width:64px"/> 分钟</label>
          <label>会话超时 <input type="number" value="120" style="width:64px"/> 分钟</label>
          <label><input type="checkbox" checked/> 强制定期改密</label>
          <label><input type="checkbox" checked/> 同账号单端登录（可配置）</label>
          <div class="split-actions"><button class="btn">保存策略</button></div>
        </div>
        <div class="scope-box">
          <h5>密码规则</h5>
          <label><input type="checkbox" checked/> 最少 8 位</label>
          <label><input type="checkbox" checked/> 含字母与数字</label>
          <label><input type="checkbox"/> 含特殊字符</label>
          <label><input type="checkbox" checked/> 不可与近 5 次重复</label>
        </div>
        <div class="scope-box">
          <h5>与权限闭环</h5>
          <p style="font-size:12px;line-height:1.7;color:#5c6b75;margin:0">
            自定义权限 + 自定义菜单 → 权限分配赋权 → 登录控制/账户冻结管控访问 → 操作日志审计 → 离职登记收回。
          </p>
        </div>
      </div>`;
  }

  function viewFreeze() {
    return `
      <div class="toolbar">
        <button class="btn warn">冻结选中</button>
        <button class="btn ghost">解冻</button>
        <button class="btn ghost">强制下线</button>
      </div>
      <div class="card" style="overflow:auto">
        <table class="data">
          <thead><tr><th>登录名</th><th>姓名</th><th>状态</th><th>原因</th><th>操作时间</th><th>操作</th></tr></thead>
          <tbody>
            <tr>
              <td>old_user</td><td>已离职试号</td><td>${tag("冻结")}</td>
              <td>离职登记收回权限</td><td>2026-07-30 18:02</td>
              <td><button class="btn sm ghost">解冻</button></td>
            </tr>
            <tr>
              <td>chen_jp</td><td>陈某</td><td>${tag("正常")}</td>
              <td>—</td><td>—</td>
              <td><button class="btn sm warn">冻结</button></td>
            </tr>
          </tbody>
        </table>
      </div>`;
  }

  function viewCostHide() {
    return `
      <div class="hint">生产管理 · 成本隐藏：角色级成本字段隐藏策略；与系统管理·自定义权限·字段策略、财务可见范围协同。</div>
      <h3 style="font-size:14px;margin:0 0 8px">字段策略 · 成本隐藏</h3>
      <div class="card" style="overflow:auto">
        <table class="data field-matrix">
          <thead><tr><th>字段</th><th>计件工</th><th>固定工</th><th>车间主任</th><th>财务</th><th>老板</th></tr></thead>
          <tbody>
            <tr><td>成本价 / 成本预算</td><td>${tag("隐藏")}</td><td>${tag("隐藏")}</td><td>${tag("隐藏")}</td><td>${tag("可见")}</td><td>${tag("可见")}</td></tr>
            <tr><td>毛利</td><td>${tag("隐藏")}</td><td>${tag("隐藏")}</td><td>${tag("隐藏")}</td><td>${tag("可见")}</td><td>${tag("可见")}</td></tr>
            <tr><td>他人工资</td><td>${tag("隐藏")}</td><td>${tag("隐藏")}</td><td>${tag("隐藏")}</td><td>${tag("可见")}</td><td>${tag("可见")}</td></tr>
            <tr><td>本人工资/计件</td><td>${tag("可见")}</td><td>${tag("可见")}</td><td>${tag("可见")}</td><td>${tag("可见")}</td><td>${tag("可见")}</td></tr>
          </tbody>
        </table>
      </div>
      <div class="split-actions"><button class="btn">保存成本隐藏策略</button></div>`;
  }

  const assignTabs = [
    { id: "users", label: "管理员/用户管理" },
    { id: "groups", label: "管理员分组" },
    { id: "roles", label: "角色管理" },
    { id: "assign", label: "用户授权" },
    { id: "overview", label: "权限闭环" }
  ];

  function viewOverview() {
    return `
      <div class="stats">
        <div class="stat"><div class="label">角色数</div><div class="value">${ROLES.length}</div></div>
        <div class="stat"><div class="label">管理员分组</div><div class="value">${GROUPS.length}</div></div>
        <div class="stat"><div class="label">用户账号</div><div class="value">${USERS.length}</div></div>
        <div class="stat"><div class="label">预置权限模型</div><div class="value" style="font-size:16px">RBAC+</div></div>
      </div>
      <div class="card" style="padding:16px">
        <h3 style="margin:0 0 10px;font-size:14px">权限闭环（文档 9.4）</h3>
        <p style="font-size:13px;line-height:1.8;color:#5c6b75;margin:0">
          <strong>自定义权限</strong> + <strong>自定义菜单</strong> 定义能力 →
          <strong>权限分配</strong>（用户/分组/角色/数据·仓·工序范围）赋给用户 →
          <strong>登录控制</strong> / <strong>账户冻结</strong> 管控访问 →
          <strong>操作日志</strong> 审计 →
          <strong>离职登记</strong> 收回。
        </p>
        <div class="iam-badge-row" style="margin-top:12px">
          <span class="tag">User</span><span class="tag">AdminGroup</span><span class="tag">Role</span>
          <span class="tag">PermissionCode</span><span class="tag">MenuCustom</span>
          <span class="tag">FieldPolicy</span><span class="tag">LoginPolicy</span>
        </div>
      </div>
      <div class="hint" style="margin-top:12px">能力入口：人事管理·权限分配；系统管理·自定义权限/菜单/登录控制/账户冻结；生产管理·成本隐藏。不另立「平台权限」等新模块名。</div>`;
  }

  function renderAssign(tab, state) {
    const body =
      tab === "users"
        ? viewUsers(state.userType)
        : tab === "groups"
          ? viewGroups(state.group)
          : tab === "roles"
            ? viewRoles(state.role)
            : tab === "assign"
              ? viewAssign(state.login)
              : viewOverview();
    return shell(
      "权限分配",
      "人事管理",
      "含管理员管理、管理员分组、角色管理、用户↔角色、数据/仓/工序范围；一人多角色取并集。",
      tabs(assignTabs, tab),
      body
    );
  }

  function renderByModule(module, state) {
    state = state || { tab: "users", group: GROUPS[0].code, role: "sys_admin", login: "wang_zr", menuRole: "piece", userType: "" };

    if (module === "权限分配") {
      return { html: renderAssign(state.tab || "users", state), state };
    }
    if (module === "自定义权限") {
      return {
        html: shell(
          "自定义权限",
          "系统管理",
          "权限码 + 字段策略；与审批、财审管控、单据锁定、成本隐藏联动。",
          tabs(
            [
              { id: "codes", label: "权限码与角色矩阵" },
              { id: "fields", label: "字段策略" }
            ],
            state.permTab || "codes"
          ),
          state.permTab === "fields"
            ? viewCostHide()
            : viewPermCodes()
        ),
        state
      };
    }
    if (module === "自定义菜单") {
      return {
        html: shell(
          "自定义菜单",
          "系统管理",
          "按角色裁剪 13 大域菜单树（显隐/排序）。",
          "",
          viewMenus(state.menuRole)
        ),
        state
      };
    }
    if (module === "登录控制") {
      return {
        html: shell("登录控制", "系统管理", "登录策略、会话与密码规则。", "", viewLogin()),
        state
      };
    }
    if (module === "账户冻结") {
      return {
        html: shell("账户冻结", "系统管理", "冻结/解冻/强制下线；与离职收回联动。", "", viewFreeze()),
        state
      };
    }
    if (module === "成本隐藏") {
      return {
        html: shell("成本隐藏", "生产管理", "角色级成本字段隐藏；与自定义权限字段策略一致。", "", viewCostHide()),
        state
      };
    }
    return null;
  }

  function bind(container, module, state, rerender) {
    container.querySelectorAll("[data-iam-tab]").forEach((btn) => {
      btn.addEventListener("click", () => {
        if (module === "权限分配") state.tab = btn.dataset.iamTab;
        if (module === "自定义权限") state.permTab = btn.dataset.iamTab === "fields" ? "fields" : "codes";
        rerender();
      });
    });
    container.querySelectorAll("[data-iam-group]").forEach((btn) => {
      btn.addEventListener("click", () => {
        state.group = btn.dataset.iamGroup;
        state.tab = "groups";
        rerender();
      });
    });
    container.querySelectorAll("[data-iam-role]").forEach((btn) => {
      btn.addEventListener("click", () => {
        state.role = btn.dataset.iamRole;
        state.tab = "roles";
        rerender();
      });
    });
    container.querySelectorAll("[data-iam-assign]").forEach((btn) => {
      btn.addEventListener("click", () => {
        state.login = btn.dataset.iamAssign;
        state.tab = "assign";
        rerender();
      });
    });
    container.querySelectorAll("[data-iam-menu-role]").forEach((btn) => {
      btn.addEventListener("click", () => {
        state.menuRole = btn.dataset.iamMenuRole;
        rerender();
      });
    });
    container.querySelectorAll("[data-iam-action=edit-user]").forEach((btn) => {
      btn.addEventListener("click", () => {
        state.login = btn.dataset.login;
        state.tab = "assign";
        rerender();
      });
    });
    const filter = container.querySelector("#userTypeFilter");
    if (filter) {
      filter.addEventListener("change", () => {
        state.userType = filter.value;
        state.tab = "users";
        rerender();
      });
    }
  }

  return {
    isIamModule: (m) => IAM_MODULES.has(m),
    renderByModule,
    bind,
    defaultState: () => ({
      tab: "users",
      group: "g_platform",
      role: "sys_admin",
      login: "wang_zr",
      menuRole: "piece",
      userType: "",
      permTab: "codes"
    })
  };
})();
