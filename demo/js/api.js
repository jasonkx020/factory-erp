(function (global) {
  const DEFAULT_BASE = "http://127.0.0.1:18080/api/v1";
  const KEY = "erp_access_token";

  function base() {
    return localStorage.getItem("erp_api_base") || DEFAULT_BASE;
  }

  function token() {
    return localStorage.getItem(KEY) || "";
  }

  function setToken(t) {
    if (t) localStorage.setItem(KEY, t);
    else localStorage.removeItem(KEY);
  }

  async function request(method, path, body) {
    const headers = { "Content-Type": "application/json" };
    const t = token();
    if (t) headers.Authorization = "Bearer " + t;
    const res = await fetch(base() + path, {
      method,
      headers,
      body: body == null ? undefined : JSON.stringify(body)
    });
    const data = await res.json();
    if (data && data.code === 0 && data.msg === "UNAUTHORIZED") {
      setToken("");
    }
    return data;
  }

  async function login(loginName, password) {
    const r = await request("POST", "/auth/login", {
      login_name: loginName,
      password: password,
      client_type: "web"
    });
    if (r.code === 1 && r.data && r.data.access_token) {
      setToken(r.data.access_token);
    }
    return r;
  }

  async function me() {
    return request("GET", "/auth/me");
  }

  /** Map UI domain to API list path */
  const DOMAIN_LIST = {
    产品管理: "/product/products",
    库存管理: "/inventory/balances",
    生产管理: "/production/processes",
    工资管理: "/payroll/wage-rates",
    人事管理: "/hr/employees",
    审批管理: "/approval/tasks",
    系统管理: "/system/settings",
    客户管理: "/crm/customers",
    销售管理: "/sales/orders",
    采购管理: "/purchase/suppliers",
    统计报表: "/report/dashboards/boss",
    财务管理: "/finance/vouchers",
    固定资产管理: "/asset/fixed-assets"
  };

  async function listByDomain(domain) {
    const path = DOMAIN_LIST[domain] || "/system/settings";
    return request("GET", path);
  }

  global.ERP_API = {
    base,
    token,
    setToken,
    request,
    login,
    me,
    listByDomain,
    DOMAIN_LIST
  };
})(window);
