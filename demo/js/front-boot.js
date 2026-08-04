(function () {
  // Shared front-end API bootstrap for workshop/worker/sales/boss
  async function boot(label) {
    const el = document.querySelector(".pad-top h1") || document.querySelector("h1");
    if (!window.ERP_API) return;
    try {
      if (!window.ERP_API.token()) {
        await window.ERP_API.login("admin", "admin123");
      }
      const me = await window.ERP_API.me();
      if (me.code === 1 && el) {
        el.setAttribute("title", "API 已连接 · " + ((me.data.user && me.data.user.name) || "admin"));
      }
      const box = document.getElementById("apiLive");
      if (box) {
        const r = await window.ERP_API.request("GET", label || "/production/processes");
        const list = (r.data && r.data.list) || [];
        box.textContent = r.code === 1 ? `API 实时 ${list.length} 条` : `API: ${r.msg}`;
      }
    } catch (e) {
      /* keep static demo */
    }
  }
  window.ERP_FRONT_BOOT = boot;
})();
