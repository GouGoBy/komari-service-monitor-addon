import { useState, type FormEvent } from "react";
import { Save, X } from "lucide-react";
import type { ServiceConfig } from "@/services/serviceMonitor";

export function ServiceEditor({ config, busy, onCancel, onSave }: { config: ServiceConfig; busy: boolean; onCancel: () => void; onSave: (body: ServiceConfig & { token?: string }) => Promise<void> }) {
  const [draft, setDraft] = useState({ ...config });
  const [token, setToken] = useState("");
  function field<K extends keyof ServiceConfig>(key: K, value: ServiceConfig[K]) { setDraft(old => ({ ...old, [key]: value })); }
  function submit(e: FormEvent) { e.preventDefault(); void onSave({ ...draft, ...(token ? { token } : {}) }); }
  return <form className="surface-card availability-editor" onSubmit={submit} key={config.id}>
    <header className="availability-toolbar"><div><div className="availability-eyebrow">MONITOR SETTINGS</div><h3>{draft.id ? "编辑服务监测" : "添加服务监测"}</h3></div><button type="button" disabled={busy} className="theme-manage-button" onClick={onCancel}><X size={14} />取消</button></header>
    <fieldset disabled={busy}><div className="availability-fields"><label>名称<input aria-label="监测名称" required maxLength={100} value={draft.name} onChange={e => field("name", e.target.value)} /></label><label>监测类型<select aria-label="监测类型" value={draft.type} onChange={e => field("type", e.target.value as ServiceConfig["type"])}><option value="http">网站 HTTP / HTTPS</option><option value="astrbot">AstrBot</option></select></label></div>
      <label>目标地址<input aria-label="目标地址" type="url" required value={draft.url} placeholder="https://example.com" onChange={e => field("url", e.target.value)} /><small>AstrBot 填写 Dashboard 根地址，不要追加 API 路径。请求由 Komari 后端发起。</small></label>
      <div className="availability-fields"><label>检查间隔 / 秒<input aria-label="检查间隔" type="number" required min={10} max={86400} value={draft.interval} onChange={e => field("interval", Number(e.target.value))} /></label><label>超时 / 秒<input aria-label="超时" type="number" required min={1} max={Math.min(60, draft.interval - 1)} value={draft.timeout} onChange={e => field("timeout", Number(e.target.value))} /></label></div>
      {draft.type === "http" ? <div className="availability-fields"><label>预期 HTTP 状态码<input aria-label="预期状态码" type="number" min={0} max={599} value={draft.expected_status} onChange={e => field("expected_status", Number(e.target.value))} /><small>0 表示接受全部 2xx 状态。</small></label><label>页面关键词 / 可选<input aria-label="页面关键词" maxLength={4096} value={draft.keyword} onChange={e => field("keyword", e.target.value)} /></label></div> : <>
        <div className="availability-fields"><label>认证方式<select aria-label="认证方式" value={draft.auth_type || "apikey"} onChange={e => field("auth_type", e.target.value)}><option value="apikey">API Key</option><option value="bearer">Bearer / Dashboard JWT</option></select></label><label>接口版本<select aria-label="接口版本" value={draft.api_path === "/api/v1/stats" ? "/api/v1/bots" : draft.api_path || "/api/v1/bots"} onChange={e => field("api_path", e.target.value)}><option value="/api/v1/bots">当前 v1 API</option><option value="/api/stat/get">旧版 Dashboard API</option></select></label></div>
        <label>API Key / JWT<input aria-label="API Key / JWT" type="password" autoComplete="new-password" maxLength={8192} value={token} onChange={e => setToken(e.target.value)} /><small>编辑时留空保留原密钥。v1 API Key 需 bot 权限；旧版接口使用 Dashboard JWT。</small></label><label>机器人实例 ID / 可选<input aria-label="机器人实例 ID" maxLength={255} value={draft.bot_id} onChange={e => field("bot_id", e.target.value)} /><small>留空检查服务 API；指定 ID 时检查该实例是否 running，仅支持 v1。</small></label>
      </>}
      <div className="availability-actions"><label className="availability-checkbox"><input type="checkbox" checked={draft.enabled} onChange={e => field("enabled", e.target.checked)} />启用监测</label><label className="availability-checkbox"><input type="checkbox" checked={draft.public} onChange={e => field("public", e.target.checked)} />在主页公开展示</label></div>
      <button type="submit" className="theme-manage-button is-primary"><Save size={14} />{busy ? "正在保存…" : "保存监测"}</button>
    </fieldset>
  </form>;
}
