import { useEffect, useRef, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Activity, ArrowLeft, ArrowUpRight, Bot, Clock3, Globe, History, Plus, RefreshCw, Settings2, X } from "lucide-react";
import { useAuth } from "@/hooks/useAuth";
import { newServiceConfig, serviceRequest, type ServiceCheck, type ServiceConfig, type ServiceView } from "@/services/serviceMonitor";
import { ServiceEditor } from "./ServiceEditor";
import "./services.css";

const states = { up: "运行正常", down: "服务异常", unknown: "等待检查", paused: "已暂停" };
const stamp = (value?: string) => value ? new Date(value).toLocaleString() : "尚无记录";

export function ServicePanel({ admin = false }: { admin?: boolean }) {
  const { data: me } = useAuth();
  const client = useQueryClient();
  const base = admin ? "/api/admin/services" : "/api/services";
  const query = useQuery({ queryKey: ["availability", admin], queryFn: ({ signal }) => serviceRequest<ServiceView[]>(base, undefined, signal), refetchInterval: 15000, retry: false });
  const [editing, setEditing] = useState<ServiceConfig | null>(null);
  const [selected, setSelected] = useState<ServiceView | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const modal = useRef<HTMLDialogElement>(null);
  const history = useQuery({ queryKey: ["availability-history", admin, selected?.id], queryFn: ({ signal }) => serviceRequest<ServiceCheck[]>(`${base}/history?id=${selected!.id}`, undefined, signal), enabled: selected !== null, retry: false });
  useEffect(() => { if (selected && !modal.current?.open) modal.current?.showModal(); }, [selected]);
  const rows = query.data ?? [];
  async function change(path: string, body: unknown) {
    setBusy(true); setError("");
    try { await serviceRequest(base + path, body); await client.invalidateQueries({ queryKey: ["availability"] }); return true; }
    catch (e) { setError(e instanceof Error ? e.message : String(e)); return false; }
    finally { setBusy(false); }
  }
  return <section className="lumina-services" aria-label={admin ? "服务监测管理" : "服务可用性"}>
    <header className="availability-toolbar">
      <div><div className="availability-eyebrow"><Activity size={13} /> SERVICE AVAILABILITY</div><h2>{admin ? "服务监测管理" : "服务可用性"}<span>{rows.length.toString().padStart(2, "0")}</span></h2><p>网站与 AstrBot · 15 秒自动刷新</p></div>
      <div className="availability-actions">
        {admin ? <Link to="/" className="theme-manage-button"><ArrowLeft size={14} />返回首页</Link> : me?.logged_in && <a href="/admin/service-monitor" className="theme-manage-button"><Settings2 size={14} />管理监测</a>}
        <button className="theme-manage-button" disabled={query.isFetching} onClick={() => void query.refetch()} aria-label="刷新服务状态"><RefreshCw size={14} className={query.isFetching ? "animate-spin" : ""} />刷新</button>
        {admin && <button className="theme-manage-button is-primary" disabled={busy} onClick={() => { setError(""); setEditing(newServiceConfig()); }}><Plus size={14} />添加监测</button>}
      </div>
    </header>
    {rows.length > 0 && <div className="availability-summary"><span><i className="is-up" />正常 <b>{rows.filter(m => m.status === "up").length}</b></span><span><i className="is-down" />异常 <b>{rows.filter(m => m.status === "down").length}</b></span><span><i />等待 / 暂停 <b>{rows.filter(m => m.status === "unknown" || m.status === "paused").length}</b></span></div>}
    {query.error && <div className="surface-card availability-notice" role="alert">服务状态读取失败：{query.error.message}<button className="theme-manage-button" onClick={() => void query.refetch()}>重试</button></div>}
    {error && <p className="availability-error" role="alert">{error}</p>}
    {admin && editing && <ServiceEditor key={editing.id} config={editing} busy={busy} onCancel={() => setEditing(null)} onSave={async body => { if (await change("/save", body)) setEditing(null); }} />}
    {query.isPending && <div className="surface-card availability-notice" role="status"><RefreshCw size={16} className="animate-spin" />正在读取服务状态…</div>}
    {!query.isPending && !query.error && rows.length === 0 && <div className="surface-card availability-notice">{admin ? "还没有服务监测，点击添加监测开始。" : "暂无公开服务监测"}</div>}
    <div className="availability-grid">{rows.map(m => {
      const Icon = m.type === "astrbot" ? Bot : Globe;
      return <article key={m.id} className="surface-card availability-card" data-state={m.status}>
        <header className="availability-card-header"><div className="availability-icon"><Icon size={19} strokeWidth={1.7} /></div><div className="availability-title"><h3 title={m.name}>{m.name}</h3><span>{m.type === "astrbot" ? "ASTRBOT" : "HTTP / HTTPS"}</span></div><span className="availability-badge"><i />{states[m.status]}</span></header>
        <div className="availability-metrics"><div><label>24H 可用率</label><strong>{m.availability_24h === null ? "—" : m.availability_24h.toFixed(2)}{m.availability_24h !== null && <small>%</small>}</strong></div><div><label>响应耗时</label><strong>{m.latest ? Math.round(m.latest.latency) : "—"}{m.latest && <small>ms</small>}</strong></div></div>
        <div className="availability-track" title="最近24小时实际检查成功率"><span style={{ width: `${Math.min(100, Math.max(0, m.availability_24h ?? 0))}%` }} /></div>
        <div className="availability-meta"><span><Clock3 size={12} />{stamp(m.latest?.time)}</span><span>{m.checks_24h} 次检查</span></div>
        {admin && m.latest?.error && <p className="availability-error">{m.latest.error}</p>}
        <footer className="availability-card-footer"><button className="availability-history-button" onClick={() => setSelected(m)}><History size={13} />检查记录<ArrowUpRight size={12} /></button>{admin && m.config && <div className="availability-actions"><button disabled={busy} className="availability-text-button" onClick={() => { setError(""); setEditing(m.config!); }}>编辑</button><button disabled={busy} className="availability-text-button" onClick={() => void change("/save", { ...m.config, enabled: !m.config!.enabled })}>{m.config.enabled ? "暂停" : "启用"}</button><button disabled={busy} className="availability-text-button danger" onClick={() => { if (window.confirm(`删除「${m.name}」及全部检查记录？`)) void change("/delete", { id: m.id }); }}>删除</button></div>}</footer>
      </article>;
    })}</div>
    <p className="availability-footnote">可用率按最近 24 小时的实际检查次数计算。AstrBot 状态代表 API 或指定实例的运行状态，不验证完整消息与模型调用链路。</p>
    <dialog ref={modal} className="availability-dialog surface-card" onClose={() => setSelected(null)}>
      <header className="availability-toolbar"><div><div className="availability-eyebrow">CHECK HISTORY</div><h2>{selected?.name}</h2><p>最近 24 小时 · 最多 1000 条</p></div><button className="theme-manage-button" aria-label="关闭检查记录" onClick={() => modal.current?.close()}><X size={16} /></button></header>
      {history.isPending ? <p role="status">正在加载检查记录…</p> : history.error ? <p className="availability-error" role="alert">{history.error.message}</p> : <div className="availability-table"><table><thead><tr><th>检查时间</th><th>状态</th><th>耗时</th><th>HTTP</th>{admin && <th>诊断</th>}</tr></thead><tbody>{history.data?.map(r => <tr key={r.id}><td>{stamp(r.time)}</td><td className={r.up ? "availability-success" : "availability-error"}>{r.up ? "正常" : "异常"}</td><td>{Math.round(r.latency)} ms</td><td>{r.status_code || "—"}</td>{admin && <td>{r.error || "—"}</td>}</tr>)}</tbody></table>{!history.data?.length && <p>暂无检查记录。</p>}</div>}
    </dialog>
  </section>;
}
