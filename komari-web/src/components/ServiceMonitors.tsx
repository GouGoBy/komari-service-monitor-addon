import { useCallback, useEffect, useRef, useState, type FormEvent } from "react";
import { Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { newServiceConfig, serviceRequest, type ServiceCheck, type ServiceConfig, type ServiceView } from "@/lib/serviceMonitor";
import "./ServiceMonitors.css";

export default function ServiceMonitors({ admin = false, embedded = false }: { admin?: boolean; embedded?: boolean }) {
  const { t } = useTranslation();
  const [monitors, setMonitors] = useState<ServiceView[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [config, setConfig] = useState<ServiceConfig | null>(null);
  const [token, setToken] = useState("");
  const [saving, setSaving] = useState(false);
  const [formError, setFormError] = useState("");
  const [history, setHistory] = useState<{ name: string; rows: ServiceCheck[] } | null>(null);
  const dialog = useRef<HTMLDialogElement>(null);
  const base = admin ? "/api/admin/services" : "/api/services";
  const refresh = useCallback(async (signal?: AbortSignal) => {
    try {
      const data = await serviceRequest<ServiceView[]>(base, undefined, signal);
      if (!signal?.aborted) { setMonitors(data); setError(""); }
    } catch (e) { if (!signal?.aborted) setError(String(e instanceof Error ? e.message : e)); }
    finally { if (!signal?.aborted) setLoading(false); }
  }, [base]);
  useEffect(() => {
    const controller = new AbortController();
    void refresh(controller.signal);
    const timer = window.setInterval(() => void refresh(controller.signal), 15000);
    return () => { controller.abort(); window.clearInterval(timer); };
  }, [refresh]);
  useEffect(() => { if (history) dialog.current?.showModal(); }, [history]);
  function edit(value: ServiceConfig) {
    setConfig({ ...value }); setToken(""); setFormError("");
  }
  async function save(event: FormEvent) {
    event.preventDefault(); if (!config) return;
    setSaving(true); setFormError("");
    try {
      await serviceRequest(base + "/save", { ...config, ...(token ? { token } : {}) });
      setToken(""); setConfig(null); await refresh();
    } catch (e) { setFormError(e instanceof Error ? e.message : String(e)); }
    finally { setSaving(false); }
  }
  async function mutate(path: string, payload: unknown) {
    setSaving(true);
    try { await serviceRequest(base + path, payload); await refresh(); }
    catch (e) { setError(e instanceof Error ? e.message : String(e)); }
    finally { setSaving(false); }
  }
  async function showHistory(monitor: ServiceView) {
    try { setHistory({ name: monitor.name, rows: await serviceRequest<ServiceCheck[]>(base + "/history?id=" + monitor.id) }); }
    catch (e) { setError(e instanceof Error ? e.message : String(e)); }
  }
  function field<K extends keyof ServiceConfig>(key: K, value: ServiceConfig[K]) {
    setConfig(previous => previous ? { ...previous, [key]: value } : null);
  }
  return <section className={"service-monitors" + (embedded ? " sm-embedded" : "")} aria-label={t("serviceMonitor.title")}>
    <header><div>{embedded ? <h2>{t("serviceMonitor.title")}</h2> : <h1>{t("serviceMonitor.title")}</h1>}<p>由 Komari 服务端检查网站与 AstrBot，每 15 秒刷新状态。</p></div><div className="sm-actions">
      <Link to={admin || embedded ? "/service-monitor" : "/admin/service-monitor"}>{admin ? "公开状态页" : embedded ? "查看全部" : "管理监测"}</Link>
      <button onClick={() => void refresh()}>刷新</button>
      {admin && <button className="sm-primary" onClick={() => edit(newServiceConfig())}>添加监测</button>}
    </div></header>
    {error && <p className="sm-error" role="alert">{error}</p>}
    {config && <form onSubmit={save} className="sm-editor">
      <h2>{config.id ? "编辑监测" : "添加监测"}</h2>
      <div className="sm-fields"><label>名称<input value={config.name} maxLength={100} required onChange={e => field("name", e.target.value)} /></label>
        <label>类型<select aria-label="类型" value={config.type} onChange={e => field("type", e.target.value as ServiceConfig["type"])}><option value="http">网站 HTTP(S)</option><option value="astrbot">AstrBot</option></select></label></div>
      <label>目标 URL<input aria-label="目标 URL" type="url" value={config.url} placeholder="https://example.com 或 http://astrbot:6185" required onChange={e => field("url", e.target.value)} /><small>AstrBot 填 Dashboard 根地址，可含反向代理前缀，不要填 API 路径。</small></label>
      <div className="sm-fields"><label>检查间隔（秒）<input type="number" min={10} max={86400} value={config.interval} required onChange={e => field("interval", Number(e.target.value))} /></label>
        <label>超时（秒，须小于间隔）<input type="number" min={1} max={60} value={config.timeout} required onChange={e => field("timeout", Number(e.target.value))} /></label></div>
      {config.type === "http" ? <div className="sm-fields"><label>预期状态码<input type="number" min={0} max={599} value={config.expected_status} onChange={e => field("expected_status", Number(e.target.value))} /><small>0 接受所有 2xx，自动跟随重定向。</small></label><label>页面关键词（可选）<input value={config.keyword} maxLength={4096} onChange={e => field("keyword", e.target.value)} /></label></div> : <>
        <div className="sm-fields"><label>认证方式<select value={config.auth_type || "apikey"} onChange={e => field("auth_type", e.target.value)}><option value="apikey">API Key</option><option value="bearer">Bearer / Dashboard JWT</option></select></label>
          <label>API 版本<select value={config.api_path === "/api/v1/stats" ? "/api/v1/bots" : config.api_path || "/api/v1/bots"} onChange={e => field("api_path", e.target.value)}><option value="/api/v1/bots">当前 v1 API</option><option value="/api/stat/get">旧版 Dashboard API</option></select></label></div>
        <label>API Key / JWT<input aria-label="API Key / JWT" type="password" value={token} autoComplete="new-password" maxLength={8192} onChange={e => setToken(e.target.value)} /><small>编辑时留空保留原密钥。v1 服务和实例检查均需要 bot 权限；旧版使用 Dashboard JWT，过期需更新。开发者 API Key 不支持 system 权限。</small></label>
        <label>机器人实例 ID（可选）<input value={config.bot_id} maxLength={255} onChange={e => field("bot_id", e.target.value)} /><small>填写后检查实例是否 running，仅支持 v1；留空检查 AstrBot 服务。</small></label>
      </>}
      <div className="sm-actions"><label className="sm-check"><input type="checkbox" checked={config.enabled} onChange={e => field("enabled", e.target.checked)} />启用监测</label><label className="sm-check"><input type="checkbox" checked={config.public} onChange={e => field("public", e.target.checked)} />在公开状态页展示</label></div>
      <div className="sm-actions"><button type="submit" className="sm-primary" disabled={saving}>保存</button><button type="button" disabled={saving} onClick={() => { setConfig(null); setToken(""); }}>取消</button></div>
      {formError && <p className="sm-error" role="alert">{formError}</p>}
    </form>}
    {loading && <p role="status">正在加载…</p>}
    {!loading && !error && !monitors.length && <p>暂无{admin ? "" : "公开"}监测。</p>}
    <div className="sm-grid">{monitors.map(m => <article key={m.id}>
      <span className={"sm-status " + m.status}>{({ up: "正常", down: "不可用", unknown: "等待检查 / 数据过期", paused: "已暂停" })[m.status]}</span>
      {embedded ? <h3>{m.name}</h3> : <h2>{m.name}</h2>}<small>{m.type === "astrbot" ? "AstrBot" : "HTTP(S)"}</small>
      <div className="sm-metrics"><div>24 小时可用率<strong>{m.availability_24h == null ? "—" : m.availability_24h.toFixed(2) + "%"}</strong></div><div>响应耗时<strong>{m.latest ? m.latest.latency.toFixed(0) + " ms" : "—"}</strong></div></div>
      <p>{m.latest ? "最后检查：" + new Date(m.latest.time).toLocaleString() : "尚无检查记录"}</p>
      {admin && m.latest?.error && <p className="sm-error">{m.latest.error}</p>}
      <div className="sm-actions"><button onClick={() => void showHistory(m)}>历史记录</button>{admin && m.config && <>
        <button disabled={saving} onClick={() => edit(m.config!)}>编辑</button>
        <button disabled={saving} onClick={() => void mutate("/save", { ...m.config, enabled: !m.config!.enabled })}>{m.config.enabled ? "暂停" : "启用"}</button>
        <button disabled={saving} onClick={() => { if (window.confirm("删除此监测及全部历史记录？")) void mutate("/delete", { id: m.id }); }}>删除</button>
      </>}</div>
    </article>)}</div>
    <small>可用率按最近 24 小时的实际检查次数计算；历史保留 30 天。AstrBot 状态不代表完整消息或模型调用链路已验证。</small>
    <dialog ref={dialog} onClose={() => setHistory(null)}><h2>{history?.name} · 最近 24 小时记录</h2><p>最多展示最近 1000 条。</p><button onClick={() => dialog.current?.close()}>关闭</button><div className="sm-history"><table><thead><tr><th>时间</th><th>状态</th><th>耗时</th><th>HTTP</th>{admin && <th>原因</th>}</tr></thead><tbody>{history?.rows.map(r => <tr key={r.id}><td>{new Date(r.time).toLocaleString()}</td><td>{r.up ? "正常" : "不可用"}</td><td>{r.latency.toFixed(0)} ms</td><td>{r.status_code || "—"}</td>{admin && <td>{r.error}</td>}</tr>)}</tbody></table>{history?.rows.length === 0 && <p>暂无记录。</p>}</div></dialog>
  </section>;
}
