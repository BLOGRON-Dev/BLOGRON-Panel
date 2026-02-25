import { useState, useEffect, useCallback } from "react";

// ── Config ─────────────────────────────────────────────────────────────────
const API_BASE = import.meta?.env?.VITE_API_URL || "http://localhost:8080";

async function api(path, opts = {}) {
  const token = localStorage.getItem("sp_token");
  const res = await fetch(`${API_BASE}${path}`, {
    headers: { "Content-Type": "application/json", ...(token ? { Authorization: `Bearer ${token}` } : {}) },
    ...opts,
  });
  if (res.status === 401) { localStorage.removeItem("sp_token"); window.location.reload(); }
  return res;
}

// ── Nav ────────────────────────────────────────────────────────────────────
const NAV = [
  { id: "dashboard",  label: "Dashboard",   icon: "⬡" },
  { id: "users",      label: "Users",        icon: "◈" },
  { id: "webserver",  label: "Web Server",   icon: "◉" },
  { id: "wordpress",  label: "WordPress",    icon: "◍" },
  { id: "databases",  label: "Databases",    icon: "◫" },
  { id: "filemanager",label: "File Manager", icon: "◧" },
  { id: "email",      label: "Email",        icon: "◎" },
  { id: "dns",        label: "DNS",          icon: "◆" },
  { id: "cron",       label: "Cron Jobs",    icon: "◷" },
  { id: "ftp",        label: "FTP",          icon: "◰" },
];

// ── Shared UI ──────────────────────────────────────────────────────────────
function StatusBadge({ status }) {
  const c = { active:"bg-emerald-500/20 text-emerald-400 border-emerald-500/30", inactive:"bg-zinc-500/20 text-zinc-400 border-zinc-500/30", suspended:"bg-rose-500/20 text-rose-400 border-rose-500/30", online:"bg-emerald-500/20 text-emerald-400 border-emerald-500/30", offline:"bg-rose-500/20 text-rose-400 border-rose-500/30", enabled:"bg-emerald-500/20 text-emerald-400 border-emerald-500/30", disabled:"bg-zinc-500/20 text-zinc-400 border-zinc-500/30" };
  return <span className={`px-2 py-0.5 rounded text-xs font-mono border ${c[status]||c.inactive}`}>{status}</span>;
}

function Btn({ children, onClick, variant="primary", small, className="" }) {
  const base = `${small?"px-2 py-1 text-xs":"px-3 py-1.5 text-xs"} rounded-lg font-mono font-bold transition-all cursor-pointer`;
  const v = { primary:"bg-cyan-500 hover:bg-cyan-400 text-black", danger:"bg-rose-500/20 hover:bg-rose-500/40 text-rose-400 border border-rose-500/30", ghost:"bg-zinc-800 hover:bg-zinc-700 text-zinc-300 border border-zinc-700", success:"bg-emerald-500/20 hover:bg-emerald-500/40 text-emerald-400 border border-emerald-500/30" };
  return <button onClick={onClick} className={`${base} ${v[variant]||v.primary} ${className}`}>{children}</button>;
}

function Modal({ title, onClose, children, wide }) {
  return (
    <div className="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center z-50 p-4" onClick={onClose}>
      <div className={`bg-zinc-900 border border-zinc-700 rounded-xl ${wide?"w-full max-w-2xl":"w-full max-w-md"} p-6 shadow-2xl`} onClick={e=>e.stopPropagation()}>
        <div className="flex items-center justify-between mb-5">
          <h3 className="text-sm font-bold text-white font-mono">{title}</h3>
          <button onClick={onClose} className="text-zinc-600 hover:text-zinc-300 text-xl leading-none cursor-pointer">×</button>
        </div>
        {children}
      </div>
    </div>
  );
}

function Field({ label, ...p }) {
  return (
    <div className="space-y-1">
      <label className="text-xs text-zinc-500 font-mono">{label}</label>
      <input {...p} className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-200 font-mono focus:outline-none focus:border-cyan-500 transition-colors" />
    </div>
  );
}

function Select({ label, options, ...p }) {
  return (
    <div className="space-y-1">
      <label className="text-xs text-zinc-500 font-mono">{label}</label>
      <select {...p} className="w-full bg-zinc-800 border border-zinc-700 rounded-lg px-3 py-2 text-sm text-zinc-200 font-mono focus:outline-none focus:border-cyan-500 transition-colors">
        {options.map(o=><option key={o.v||o} value={o.v||o}>{o.l||o}</option>)}
      </select>
    </div>
  );
}

function Table({ cols, rows, actions }) {
  return (
    <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-auto">
      <table className="w-full text-xs font-mono min-w-max">
        <thead>
          <tr className="border-b border-zinc-800 text-zinc-600 uppercase tracking-widest">
            {cols.map(c=><th key={c} className="text-left px-4 py-3">{c}</th>)}
            {actions && <th className="text-right px-4 py-3">Actions</th>}
          </tr>
        </thead>
        <tbody>{rows}</tbody>
      </table>
    </div>
  );
}

function TR({ cells, actions }) {
  return (
    <tr className="border-b border-zinc-800/50 hover:bg-zinc-800/30 transition-colors">
      {cells.map((c,i)=><td key={i} className="px-4 py-3 text-zinc-300">{c}</td>)}
      {actions && <td className="px-4 py-3 text-right"><div className="flex items-center justify-end gap-1.5">{actions}</div></td>}
    </tr>
  );
}

function SectionHeader({ title, sub, onAdd, addLabel="+ Add" }) {
  return (
    <div className="flex items-center justify-between">
      <div>
        <h2 className="text-xl font-bold text-white font-mono">{title}</h2>
        {sub && <p className="text-xs text-zinc-600 font-mono mt-0.5">{sub}</p>}
      </div>
      {onAdd && <Btn onClick={onAdd}>{addLabel}</Btn>}
    </div>
  );
}

function EmptyState({ icon, text }) {
  return (
    <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-12 text-center">
      <div className="text-4xl mb-3 opacity-30">{icon}</div>
      <p className="text-zinc-600 font-mono text-sm">{text}</p>
    </div>
  );
}

// ── Login ──────────────────────────────────────────────────────────────────
function Login({ onLogin }) {
  const [u, setU] = useState(""); const [p, setP] = useState(""); const [err, setErr] = useState(""); const [loading, setLoading] = useState(false);
  const submit = async () => {
    setLoading(true); setErr("");
    const r = await fetch(`${API_BASE}/api/auth/login`, { method:"POST", headers:{"Content-Type":"application/json"}, body: JSON.stringify({username:u, password:p}) });
    const d = await r.json();
    setLoading(false);
    if (d.token) { localStorage.setItem("sp_token", d.token); onLogin(); }
    else setErr(d.error || "Login failed");
  };
  return (
    <div className="min-h-screen bg-zinc-950 flex items-center justify-center" style={{fontFamily:"'JetBrains Mono','Fira Code',monospace"}}>
      <div className="w-full max-w-sm">
        <div className="text-center mb-8">
          <div className="w-14 h-14 bg-cyan-500 rounded-2xl flex items-center justify-center text-black text-2xl font-bold mx-auto mb-4">⬡</div>
          <h1 className="text-2xl font-bold text-white">BLOGRON <span className="text-cyan-400">Panel</span></h1>
          <p className="text-zinc-600 text-xs mt-1">Sign in to manage your server</p>
        </div>
        <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-6 space-y-4">
          <Field label="Username" value={u} onChange={e=>setU(e.target.value)} placeholder="admin" />
          <Field label="Password" type="password" value={p} onChange={e=>setP(e.target.value)} placeholder="••••••••" onKeyDown={e=>e.key==="Enter"&&submit()} />
          {err && <p className="text-xs text-rose-400 font-mono">{err}</p>}
          <Btn onClick={submit} className="w-full justify-center">{loading ? "Signing in…" : "Sign In"}</Btn>
        </div>
      </div>
    </div>
  );
}

// ── Dashboard ──────────────────────────────────────────────────────────────
function Dashboard() {
  const [stats, setStats] = useState(null);
  const [services, setServices] = useState([]);

  const load = useCallback(async () => {
    const [sr, svr] = await Promise.all([api("/api/system/stats"), api("/api/system/services")]);
    if (sr.ok) setStats(await sr.json());
    if (svr.ok) setServices(await svr.json());
  }, []);

  useEffect(() => { load(); const t = setInterval(load, 5000); return ()=>clearInterval(t); }, [load]);

  const pct = (v,t) => t ? Math.round(v/t*100) : 0;

  const cards = stats ? [
    { label:"CPU", val:`${Math.round(stats.cpu?.used_pct||0)}%`, sub:`${stats.cpu?.cores||0} cores`, c:"cyan" },
    { label:"RAM", val:`${pct(stats.ram?.used_mb,stats.ram?.total_mb)}%`, sub:`${stats.ram?.used_mb||0} / ${stats.ram?.total_mb||0} MB`, c:"violet" },
    { label:"Disk", val:`${Math.round(stats.disk?.used_pct||0)}%`, sub:`${stats.disk?.used_gb||0} / ${stats.disk?.total_gb||0} GB`, c:"amber" },
    { label:"Uptime", val:stats.uptime||"-", sub:`Load: ${stats.load_avg||"-"}`, c:"rose" },
  ] : [];

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-bold text-white font-mono">Dashboard</h2>
        <p className="text-xs text-zinc-600 font-mono">{stats?.os || "Loading…"}</p>
      </div>
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        {cards.map(card=>(
          <div key={card.label} className={`bg-gradient-to-br from-${card.c}-500/10 to-transparent border border-${card.c}-500/20 rounded-xl p-5`}>
            <p className="text-xs text-zinc-500 font-mono uppercase tracking-widest">{card.label}</p>
            <p className={`text-3xl font-bold font-mono text-${card.c}-400 mt-1`}>{card.val}</p>
            <p className="text-xs text-zinc-600 mt-1">{card.sub}</p>
          </div>
        ))}
      </div>
      <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-5">
        <h3 className="text-sm font-bold text-zinc-300 font-mono mb-4">Services</h3>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
          {services.map(s=>(
            <div key={s.name} className="flex items-center justify-between bg-zinc-800/50 rounded-lg px-4 py-2.5">
              <div className="flex items-center gap-2">
                <div className={`w-1.5 h-1.5 rounded-full ${s.active?"bg-emerald-400 animate-pulse":"bg-zinc-600"}`}/>
                <span className="font-mono text-sm text-zinc-300">{s.name}</span>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-xs text-zinc-600 font-mono">{s.pid ? `PID ${s.pid}` : "—"}</span>
                <StatusBadge status={s.active?"active":"inactive"} />
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

// ── Users ──────────────────────────────────────────────────────────────────
function UsersPanel() {
  const [users, setUsers] = useState([]); const [modal, setModal] = useState(false);
  const [form, setForm] = useState({username:"",email:"",password:"",role:"user"});
  const f = v => ({ ...form, ...v });

  const load = useCallback(async () => { const r = await api("/api/users"); if (r.ok) setUsers(await r.json()); }, []);
  useEffect(()=>{load();},[load]);

  const create = async () => {
    await api("/api/users",{method:"POST",body:JSON.stringify(form)});
    setModal(false); load();
  };
  const del = async u => { await api(`/api/users/${u}`,{method:"DELETE"}); load(); };
  const suspend = async u => { await api(`/api/users/${u}/suspend`,{method:"POST"}); load(); };
  const activate = async u => { await api(`/api/users/${u}/activate`,{method:"POST"}); load(); };

  return (
    <div className="space-y-5">
      <SectionHeader title="Users" sub={`${users.length} accounts`} onAdd={()=>setModal(true)} addLabel="+ Create User" />
      <Table cols={["Username","UID","Home","Shell","Status"]} actions rows={users.map(u=>(
        <TR key={u.username} cells={[
          <span className="text-white font-bold">{u.username}</span>,
          <span className="text-zinc-500">{u.uid}</span>,
          <span className="text-zinc-500">{u.home}</span>,
          <span className="text-zinc-500">{u.shell}</span>,
          <StatusBadge status={u.locked?"suspended":"active"} />
        ]} actions={[
          u.locked ? <Btn key="a" small variant="success" onClick={()=>activate(u.username)}>Activate</Btn>
                   : <Btn key="s" small variant="ghost" onClick={()=>suspend(u.username)}>Suspend</Btn>,
          <Btn key="d" small variant="danger" onClick={()=>del(u.username)}>Delete</Btn>
        ]} />
      ))} />
      {modal && <Modal title="Create User" onClose={()=>setModal(false)}>
        <div className="space-y-4">
          <Field label="Username" value={form.username} onChange={e=>setForm(f({username:e.target.value}))} placeholder="john_doe"/>
          <Field label="Password" type="password" value={form.password} onChange={e=>setForm(f({password:e.target.value}))} placeholder="••••••••"/>
          <Select label="Shell" value={form.shell} onChange={e=>setForm(f({shell:e.target.value}))} options={["/bin/bash","/bin/sh","/usr/sbin/nologin"]}/>
          <div className="flex justify-end gap-2 pt-2">
            <Btn variant="ghost" onClick={()=>setModal(false)}>Cancel</Btn>
            <Btn onClick={create}>Create</Btn>
          </div>
        </div>
      </Modal>}
    </div>
  );
}

// ── Web Server ─────────────────────────────────────────────────────────────
function WebServerPanel() {
  const [vhosts, setVhosts] = useState([]); const [modal, setModal] = useState(false);
  const [form, setForm] = useState({domain:"",docroot:"",php:"8.2",ssl:false});
  const f = v => ({...form,...v});
  const load = useCallback(async()=>{ const r=await api("/api/vhosts"); if(r.ok) setVhosts(await r.json()); },[]);
  useEffect(()=>{load();},[load]);

  const create = async()=>{ await api("/api/vhosts",{method:"POST",body:JSON.stringify(form)}); setModal(false); load(); };
  const del = async d=>{ await api(`/api/vhosts/${d}`,{method:"DELETE"}); load(); };
  const toggle = async(d,enabled)=>{ await api(`/api/vhosts/${d}/${enabled?"disable":"enable"}`,{method:"POST"}); load(); };

  return (
    <div className="space-y-5">
      <SectionHeader title="Web Server" sub="Nginx virtual hosts" onAdd={()=>setModal(true)} addLabel="+ Add Vhost"/>
      <div className="grid gap-3">
        {vhosts.length===0 && <EmptyState icon="◉" text="No virtual hosts configured"/>}
        {vhosts.map(v=>(
          <div key={v.domain} className="bg-zinc-900 border border-zinc-800 hover:border-zinc-700 rounded-xl p-5 transition-colors">
            <div className="flex items-start justify-between">
              <div>
                <div className="flex items-center gap-2 flex-wrap">
                  <span className="font-mono font-bold text-white">{v.domain}</span>
                  {v.ssl && <span className="text-xs font-mono px-1.5 py-0.5 rounded bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">SSL</span>}
                  {v.php && <span className="text-xs font-mono px-1.5 py-0.5 rounded bg-violet-500/10 text-violet-400 border border-violet-500/20">PHP {v.php}</span>}
                </div>
                <p className="text-xs text-zinc-600 font-mono mt-1">{v.docroot||"—"}</p>
              </div>
              <div className="flex items-center gap-2">
                <StatusBadge status={v.enabled?"active":"inactive"}/>
                <Btn small variant="ghost" onClick={()=>toggle(v.domain,v.enabled)}>{v.enabled?"Disable":"Enable"}</Btn>
                <Btn small variant="danger" onClick={()=>del(v.domain)}>Delete</Btn>
              </div>
            </div>
          </div>
        ))}
      </div>
      {modal && <Modal title="Add Virtual Host" onClose={()=>setModal(false)}>
        <div className="space-y-4">
          <Field label="Domain" value={form.domain} onChange={e=>setForm(f({domain:e.target.value}))} placeholder="example.com"/>
          <Field label="Document Root (optional)" value={form.docroot} onChange={e=>setForm(f({docroot:e.target.value}))} placeholder="/var/www/example.com"/>
          <Select label="PHP Version" value={form.php} onChange={e=>setForm(f({php:e.target.value}))} options={["8.3","8.2","8.1","8.0","7.4"]}/>
          <div className="flex items-center gap-2">
            <input type="checkbox" id="ssl" checked={form.ssl} onChange={e=>setForm(f({ssl:e.target.checked}))} className="accent-cyan-500"/>
            <label htmlFor="ssl" className="text-xs text-zinc-400 font-mono">Enable SSL (Let's Encrypt)</label>
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <Btn variant="ghost" onClick={()=>setModal(false)}>Cancel</Btn>
            <Btn onClick={create}>Create</Btn>
          </div>
        </div>
      </Modal>}
    </div>
  );
}

// ── Databases ─────────────────────────────────────────────────────────────
function DatabasePanel() {
  const [dbs, setDbs] = useState([]); const [modal, setModal] = useState(false);
  const [form, setForm] = useState({name:"",db_user:"",password:""});
  const f = v=>({...form,...v});
  const load = useCallback(async()=>{ const r=await api("/api/databases"); if(r.ok) setDbs(await r.json()); },[]);
  useEffect(()=>{load();},[load]);

  const create = async()=>{ await api("/api/databases",{method:"POST",body:JSON.stringify(form)}); setModal(false); load(); };
  const drop = async n=>{ await api(`/api/databases/${n}`,{method:"DELETE"}); load(); };

  return (
    <div className="space-y-5">
      <SectionHeader title="Databases" sub="MySQL" onAdd={()=>setModal(true)} addLabel="+ Create DB"/>
      <Table cols={["Database","Size","Tables"]} actions rows={dbs.map(d=>(
        <TR key={d.name} cells={[
          <span className="text-cyan-400 font-bold">{d.name}</span>,
          <span className="text-zinc-500">{d.size||"—"}</span>,
          <span className="text-zinc-500">{d.tables||0}</span>
        ]} actions={[<Btn key="d" small variant="danger" onClick={()=>drop(d.name)}>Drop</Btn>]}/>
      ))}/>
      {modal && <Modal title="Create Database" onClose={()=>setModal(false)}>
        <div className="space-y-4">
          <Field label="Database Name" value={form.name} onChange={e=>setForm(f({name:e.target.value}))} placeholder="my_database"/>
          <Field label="DB User" value={form.db_user} onChange={e=>setForm(f({db_user:e.target.value}))} placeholder="db_user"/>
          <Field label="DB Password" type="password" value={form.password} onChange={e=>setForm(f({password:e.target.value}))} placeholder="••••••••"/>
          <div className="flex justify-end gap-2 pt-2">
            <Btn variant="ghost" onClick={()=>setModal(false)}>Cancel</Btn>
            <Btn onClick={create}>Create</Btn>
          </div>
        </div>
      </Modal>}
    </div>
  );
}

// ── File Manager ───────────────────────────────────────────────────────────
function FileManagerPanel() {
  const [path, setPath] = useState("/"); const [files, setFiles] = useState([]); const [modal, setModal] = useState(false); const [newName, setNewName] = useState("");
  const load = useCallback(async()=>{ const r=await api(`/api/files?path=${encodeURIComponent(path)}`); if(r.ok){const d=await r.json(); setFiles(d.files||[]);} },[path]);
  useEffect(()=>{load();},[load]);

  const mkdir = async()=>{ await api("/api/files/mkdir",{method:"POST",body:JSON.stringify({path:`${path}/${newName}`})}); setModal(false); setNewName(""); load(); };
  const del = async p=>{ await api(`/api/files?path=${encodeURIComponent(p)}`,{method:"DELETE"}); load(); };
  const nav = name => setPath(`${path==="/"?"":path}/${name}`);
  const breadcrumbs = path.split("/").filter(Boolean);

  return (
    <div className="space-y-5">
      <SectionHeader title="File Manager" sub={`/var/www${path}`} onAdd={()=>setModal(true)} addLabel="+ New Folder"/>
      <div className="flex items-center gap-1 font-mono text-xs bg-zinc-900 border border-zinc-800 rounded-lg px-4 py-2.5">
        <span onClick={()=>setPath("/")} className="text-cyan-400 hover:text-cyan-300 cursor-pointer">/</span>
        {breadcrumbs.map((b,i)=>(
          <span key={i} className="flex items-center gap-1">
            <span className="text-zinc-700">›</span>
            <span className="text-cyan-400 hover:text-cyan-300 cursor-pointer" onClick={()=>setPath("/"+breadcrumbs.slice(0,i+1).join("/"))}>{b}</span>
          </span>
        ))}
      </div>
      <Table cols={["Name","Permissions","Size","Modified"]} actions rows={(files||[]).map((f,i)=>(
        <TR key={i} cells={[
          <span className={`flex items-center gap-2 cursor-pointer ${f.is_dir?"text-cyan-400 font-bold":"text-zinc-300"}`} onClick={()=>f.is_dir&&nav(f.name)}>
            <span>{f.is_dir?"📁":"📄"}</span>{f.name}
          </span>,
          <span className="text-zinc-600">{f.permissions}</span>,
          <span className="text-zinc-500">{f.is_dir?"—":f.size>1048576?(f.size/1048576).toFixed(1)+" MB":f.size>1024?(f.size/1024).toFixed(1)+" KB":f.size+" B"}</span>,
          <span className="text-zinc-600">{new Date(f.modified).toLocaleDateString()}</span>
        ]} actions={[<Btn key="d" small variant="danger" onClick={()=>del(`${path}/${f.name}`)}>Delete</Btn>]}/>
      ))}/>
      {modal && <Modal title="New Folder" onClose={()=>setModal(false)}>
        <div className="space-y-4">
          <Field label="Folder Name" value={newName} onChange={e=>setNewName(e.target.value)} placeholder="new-folder"/>
          <div className="flex justify-end gap-2"><Btn variant="ghost" onClick={()=>setModal(false)}>Cancel</Btn><Btn onClick={mkdir}>Create</Btn></div>
        </div>
      </Modal>}
    </div>
  );
}

// ── Email ──────────────────────────────────────────────────────────────────
function EmailPanel() {
  const [domains, setDomains] = useState([]); const [mailboxes, setMailboxes] = useState([]); const [tab, setTab] = useState("domains");
  const [domainModal, setDomainModal] = useState(false); const [mbModal, setMbModal] = useState(false);
  const [domainForm, setDomainForm] = useState({domain:""});
  const [mbForm, setMbForm] = useState({email:"",password:"",quota:"1G"});

  const loadDomains = useCallback(async()=>{ const r=await api("/api/email/domains"); if(r.ok) setDomains(await r.json()); },[]);
  const loadMailboxes = useCallback(async()=>{ const r=await api("/api/email/mailboxes"); if(r.ok) setMailboxes(await r.json()); },[]);
  useEffect(()=>{ loadDomains(); loadMailboxes(); },[loadDomains,loadMailboxes]);

  const addDomain = async()=>{ await api("/api/email/domains",{method:"POST",body:JSON.stringify(domainForm)}); setDomainModal(false); loadDomains(); };
  const delDomain = async d=>{ await api(`/api/email/domains/${d}`,{method:"DELETE"}); loadDomains(); };
  const addMb = async()=>{ await api("/api/email/mailboxes",{method:"POST",body:JSON.stringify(mbForm)}); setMbModal(false); loadMailboxes(); };
  const delMb = async e=>{ await api(`/api/email/mailboxes/${encodeURIComponent(e)}`,{method:"DELETE"}); loadMailboxes(); };

  return (
    <div className="space-y-5">
      <SectionHeader title="Email" sub="Postfix + Dovecot" onAdd={()=>tab==="domains"?setDomainModal(true):setMbModal(true)} addLabel={tab==="domains"?"+ Add Domain":"+ Add Mailbox"}/>
      <div className="flex gap-1 bg-zinc-900 border border-zinc-800 rounded-lg p-1 w-fit">
        {["domains","mailboxes"].map(t=>(
          <button key={t} onClick={()=>setTab(t)} className={`px-4 py-1.5 rounded-md text-xs font-mono font-bold transition-all cursor-pointer capitalize ${tab===t?"bg-zinc-700 text-white":"text-zinc-500 hover:text-zinc-300"}`}>{t}</button>
        ))}
      </div>
      {tab==="domains" && <>
        <Table cols={["Domain","Mailboxes","Status"]} actions rows={domains.map(d=>(
          <TR key={d.domain} cells={[
            <span className="text-white font-bold">{d.domain}</span>,
            <span className="text-zinc-500">{d.mailboxes}</span>,
            <StatusBadge status="active"/>
          ]} actions={[<Btn key="d" small variant="danger" onClick={()=>delDomain(d.domain)}>Delete</Btn>]}/>
        ))}/>
        {domains.length===0 && <EmptyState icon="◎" text="No mail domains configured"/>}
      </>}
      {tab==="mailboxes" && <>
        <Table cols={["Email","Domain","Quota"]} actions rows={mailboxes.map(m=>(
          <TR key={m.email} cells={[
            <span className="text-cyan-400">{m.email}</span>,
            <span className="text-zinc-500">{m.domain}</span>,
            <span className="text-zinc-500">{m.quota||"1G"}</span>
          ]} actions={[<Btn key="d" small variant="danger" onClick={()=>delMb(m.email)}>Delete</Btn>]}/>
        ))}/>
        {mailboxes.length===0 && <EmptyState icon="◎" text="No mailboxes configured"/>}
      </>}
      {domainModal && <Modal title="Add Mail Domain" onClose={()=>setDomainModal(false)}>
        <div className="space-y-4">
          <Field label="Domain" value={domainForm.domain} onChange={e=>setDomainForm({domain:e.target.value})} placeholder="example.com"/>
          <div className="flex justify-end gap-2"><Btn variant="ghost" onClick={()=>setDomainModal(false)}>Cancel</Btn><Btn onClick={addDomain}>Add</Btn></div>
        </div>
      </Modal>}
      {mbModal && <Modal title="Create Mailbox" onClose={()=>setMbModal(false)}>
        <div className="space-y-4">
          <Field label="Email Address" value={mbForm.email} onChange={e=>setMbForm({...mbForm,email:e.target.value})} placeholder="user@example.com"/>
          <Field label="Password" type="password" value={mbForm.password} onChange={e=>setMbForm({...mbForm,password:e.target.value})} placeholder="••••••••"/>
          <Select label="Quota" value={mbForm.quota} onChange={e=>setMbForm({...mbForm,quota:e.target.value})} options={["256M","512M","1G","2G","5G","Unlimited"]}/>
          <div className="flex justify-end gap-2"><Btn variant="ghost" onClick={()=>setMbModal(false)}>Cancel</Btn><Btn onClick={addMb}>Create</Btn></div>
        </div>
      </Modal>}
    </div>
  );
}

// ── DNS ────────────────────────────────────────────────────────────────────
function DNSPanel() {
  const [zones, setZones] = useState([]); const [selected, setSelected] = useState(null); const [zone, setZone] = useState(null);
  const [zoneModal, setZoneModal] = useState(false); const [recModal, setRecModal] = useState(false);
  const [zoneForm, setZoneForm] = useState({domain:"",ip:""});
  const [recForm, setRecForm] = useState({name:"",type:"A",ttl:"3600",value:""});

  const loadZones = useCallback(async()=>{ const r=await api("/api/dns"); if(r.ok) setZones(await r.json()); },[]);
  const loadZone = useCallback(async d=>{ const r=await api(`/api/dns/${d}`); if(r.ok) setZone(await r.json()); },[]);
  useEffect(()=>{loadZones();},[loadZones]);
  useEffect(()=>{ if(selected) loadZone(selected); },[selected,loadZone]);

  const createZone = async()=>{ await api("/api/dns",{method:"POST",body:JSON.stringify(zoneForm)}); setZoneModal(false); loadZones(); };
  const delZone = async d=>{ await api(`/api/dns/${d}`,{method:"DELETE"}); setSelected(null); setZone(null); loadZones(); };
  const addRecord = async()=>{ await api(`/api/dns/${selected}/records`,{method:"POST",body:JSON.stringify(recForm)}); setRecModal(false); loadZone(selected); };
  const delRecord = async rec=>{ await api(`/api/dns/${selected}/records`,{method:"DELETE",body:JSON.stringify({name:rec.name,type:rec.type,value:rec.value})}); loadZone(selected); };

  return (
    <div className="space-y-5">
      <SectionHeader title="DNS Manager" sub="BIND9 zone editor" onAdd={()=>setZoneModal(true)} addLabel="+ New Zone"/>
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <div className="space-y-2">
          <p className="text-xs text-zinc-600 font-mono uppercase tracking-widest px-1">Zones</p>
          {zones.length===0 && <p className="text-xs text-zinc-700 font-mono px-1">No zones</p>}
          {zones.map(z=>(
            <div key={z.domain} onClick={()=>setSelected(z.domain)} className={`flex items-center justify-between px-4 py-3 rounded-lg cursor-pointer transition-colors font-mono text-sm ${selected===z.domain?"bg-cyan-500/10 border border-cyan-500/20 text-cyan-400":"bg-zinc-900 border border-zinc-800 text-zinc-300 hover:border-zinc-700"}`}>
              <span>{z.domain}</span>
              <Btn small variant="danger" onClick={e=>{e.stopPropagation();delZone(z.domain);}}>×</Btn>
            </div>
          ))}
        </div>
        <div className="lg:col-span-2">
          {!selected ? (
            <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-12 text-center"><p className="text-zinc-600 font-mono text-sm">Select a zone to manage records</p></div>
          ) : (
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <p className="text-sm font-mono font-bold text-white">{selected}</p>
                <Btn small onClick={()=>setRecModal(true)}>+ Add Record</Btn>
              </div>
              <Table cols={["Name","TTL","Type","Value"]} actions rows={(zone?.records||[]).map((rec,i)=>(
                <TR key={i} cells={[
                  <span className="text-zinc-300">{rec.name}</span>,
                  <span className="text-zinc-600">{rec.ttl||"3600"}</span>,
                  <span className={`font-bold ${rec.type==="A"?"text-cyan-400":rec.type==="MX"?"text-amber-400":rec.type==="TXT"?"text-violet-400":"text-zinc-300"}`}>{rec.type}</span>,
                  <span className="text-zinc-400 max-w-xs truncate">{rec.value}</span>
                ]} actions={[<Btn key="d" small variant="danger" onClick={()=>delRecord(rec)}>Delete</Btn>]}/>
              ))}/>
            </div>
          )}
        </div>
      </div>
      {zoneModal && <Modal title="Create DNS Zone" onClose={()=>setZoneModal(false)}>
        <div className="space-y-4">
          <Field label="Domain" value={zoneForm.domain} onChange={e=>setZoneForm({...zoneForm,domain:e.target.value})} placeholder="example.com"/>
          <Field label="Server IP" value={zoneForm.ip} onChange={e=>setZoneForm({...zoneForm,ip:e.target.value})} placeholder="1.2.3.4"/>
          <div className="flex justify-end gap-2"><Btn variant="ghost" onClick={()=>setZoneModal(false)}>Cancel</Btn><Btn onClick={createZone}>Create</Btn></div>
        </div>
      </Modal>}
      {recModal && <Modal title="Add DNS Record" onClose={()=>setRecModal(false)}>
        <div className="space-y-4">
          <Field label="Name" value={recForm.name} onChange={e=>setRecForm({...recForm,name:e.target.value})} placeholder="www"/>
          <Select label="Type" value={recForm.type} onChange={e=>setRecForm({...recForm,type:e.target.value})} options={["A","AAAA","CNAME","MX","TXT","NS","SRV"]}/>
          <Field label="Value" value={recForm.value} onChange={e=>setRecForm({...recForm,value:e.target.value})} placeholder="1.2.3.4"/>
          <Field label="TTL (seconds)" value={recForm.ttl} onChange={e=>setRecForm({...recForm,ttl:e.target.value})} placeholder="3600"/>
          <div className="flex justify-end gap-2"><Btn variant="ghost" onClick={()=>setRecModal(false)}>Cancel</Btn><Btn onClick={addRecord}>Add</Btn></div>
        </div>
      </Modal>}
    </div>
  );
}

// ── Cron ───────────────────────────────────────────────────────────────────
function CronPanel() {
  const [jobs, setJobs] = useState([]); const [modal, setModal] = useState(false);
  const [form, setForm] = useState({minute:"0",hour:"*",day:"*",month:"*",weekday:"*",command:"",user:"root"});
  const f = v=>({...form,...v});
  const load = useCallback(async()=>{ const r=await api("/api/cron"); if(r.ok) setJobs(await r.json()); },[]);
  useEffect(()=>{load();},[load]);

  const create = async()=>{ await api("/api/cron",{method:"POST",body:JSON.stringify(form)}); setModal(false); load(); };
  const del = async(id,user)=>{ await api(`/api/cron/${id}?user=${user}`,{method:"DELETE"}); load(); };
  const run = async(id,user)=>{ await api(`/api/cron/${id}/run?user=${user}`,{method:"POST"}); };

  const presets = [
    {l:"Every minute",v:{minute:"*",hour:"*",day:"*",month:"*",weekday:"*"}},
    {l:"Every hour",v:{minute:"0",hour:"*",day:"*",month:"*",weekday:"*"}},
    {l:"Daily midnight",v:{minute:"0",hour:"0",day:"*",month:"*",weekday:"*"}},
    {l:"Weekly Sunday",v:{minute:"0",hour:"0",day:"*",month:"*",weekday:"0"}},
    {l:"Monthly",v:{minute:"0",hour:"0",day:"1",month:"*",weekday:"*"}},
  ];

  return (
    <div className="space-y-5">
      <SectionHeader title="Cron Jobs" sub={`${jobs.length} scheduled tasks`} onAdd={()=>setModal(true)} addLabel="+ Add Job"/>
      <Table cols={["Schedule","Command","User"]} actions rows={jobs.map(j=>(
        <TR key={j.id} cells={[
          <span className="text-cyan-400">{j.schedule||`${j.minute} ${j.hour} ${j.day} ${j.month} ${j.weekday}`}</span>,
          <span className="text-zinc-300 font-mono max-w-xs truncate">{j.command}</span>,
          <span className="text-zinc-500">{j.user}</span>
        ]} actions={[
          <Btn key="r" small variant="ghost" onClick={()=>run(j.id,j.user)}>▶ Run</Btn>,
          <Btn key="d" small variant="danger" onClick={()=>del(j.id,j.user)}>Delete</Btn>
        ]}/>
      ))}/>
      {jobs.length===0 && <EmptyState icon="◷" text="No cron jobs scheduled"/>}
      {modal && <Modal title="Add Cron Job" onClose={()=>setModal(false)} wide>
        <div className="space-y-4">
          <div className="space-y-1">
            <label className="text-xs text-zinc-500 font-mono">Quick Presets</label>
            <div className="flex flex-wrap gap-2">
              {presets.map(p=><Btn key={p.l} small variant="ghost" onClick={()=>setForm(f(p.v))}>{p.l}</Btn>)}
            </div>
          </div>
          <div className="grid grid-cols-5 gap-2">
            {[["Minute","minute"],["Hour","hour"],["Day","day"],["Month","month"],["Weekday","weekday"]].map(([l,k])=>(
              <Field key={k} label={l} value={form[k]} onChange={e=>setForm(f({[k]:e.target.value}))} placeholder="*"/>
            ))}
          </div>
          <Field label="Command" value={form.command} onChange={e=>setForm(f({command:e.target.value}))} placeholder="/usr/bin/php /var/www/app/artisan schedule:run"/>
          <Select label="Run as User" value={form.user} onChange={e=>setForm(f({user:e.target.value}))} options={["root","www-data"]}/>
          <div className="text-xs text-zinc-600 font-mono bg-zinc-800 rounded px-3 py-2">
            Preview: <span className="text-cyan-400">{form.minute} {form.hour} {form.day} {form.month} {form.weekday}</span> <span className="text-zinc-400">{form.command}</span>
          </div>
          <div className="flex justify-end gap-2"><Btn variant="ghost" onClick={()=>setModal(false)}>Cancel</Btn><Btn onClick={create}>Create</Btn></div>
        </div>
      </Modal>}
    </div>
  );
}

// ── FTP ────────────────────────────────────────────────────────────────────
function FTPPanel() {
  const [users, setUsers] = useState([]); const [modal, setModal] = useState(false);
  const [form, setForm] = useState({username:"",password:"",home_dir:""});
  const f = v=>({...form,...v});
  const load = useCallback(async()=>{ const r=await api("/api/ftp"); if(r.ok) setUsers(await r.json()); },[]);
  useEffect(()=>{load();},[load]);

  const create = async()=>{ await api("/api/ftp",{method:"POST",body:JSON.stringify(form)}); setModal(false); load(); };
  const del = async u=>{ await api(`/api/ftp/${u}`,{method:"DELETE"}); load(); };

  return (
    <div className="space-y-5">
      <SectionHeader title="FTP Accounts" sub="vsftpd" onAdd={()=>setModal(true)} addLabel="+ Add FTP User"/>
      <Table cols={["Username","Home Directory","Status"]} actions rows={users.map(u=>(
        <TR key={u.username} cells={[
          <span className="text-white font-bold">{u.username}</span>,
          <span className="text-zinc-500 font-mono">{u.home_dir}</span>,
          <StatusBadge status="active"/>
        ]} actions={[<Btn key="d" small variant="danger" onClick={()=>del(u.username)}>Delete</Btn>]}/>
      ))}/>
      {users.length===0 && <EmptyState icon="◰" text="No FTP accounts configured"/>}
      {modal && <Modal title="Create FTP Account" onClose={()=>setModal(false)}>
        <div className="space-y-4">
          <Field label="Username" value={form.username} onChange={e=>setForm(f({username:e.target.value}))} placeholder="ftpuser"/>
          <Field label="Password" type="password" value={form.password} onChange={e=>setForm(f({password:e.target.value}))} placeholder="••••••••"/>
          <Field label="Home Directory (optional)" value={form.home_dir} onChange={e=>setForm(f({home_dir:e.target.value}))} placeholder="/var/www/example.com"/>
          <div className="flex justify-end gap-2"><Btn variant="ghost" onClick={()=>setModal(false)}>Cancel</Btn><Btn onClick={create}>Create</Btn></div>
        </div>
      </Modal>}
    </div>
  );
}

// ── WordPress ─────────────────────────────────────────────────────────────
function WordPressPanel() {
  const [sites, setSites] = useState([]);
  const [selected, setSelected] = useState(null);
  const [tab, setTab] = useState("plugins"); // plugins | themes
  const [plugins, setPlugins] = useState([]);
  const [themes, setThemes] = useState([]);
  const [createModal, setCreateModal] = useState(false);
  const [pluginModal, setPluginModal] = useState(false);
  const [themeModal, setThemeModal] = useState(false);
  const [srModal, setSrModal] = useState(false);
  const [loading, setLoading] = useState(false);
  const [msg, setMsg] = useState("");

  const [form, setForm] = useState({
    domain:"", site_title:"", admin_user:"admin", admin_pass:"",
    admin_email:"", db_name:"", db_user:"", db_pass:"", php:"8.2"
  });
  const [pluginForm, setPluginForm] = useState({name:"", activate:true});
  const [themeForm, setThemeForm] = useState({name:"", activate:false});
  const [srForm, setSrForm] = useState({search:"", replace:""});

  const flash = m => { setMsg(m); setTimeout(()=>setMsg(""), 4000); };
  const f = v => ({...form,...v});

  const loadSites = useCallback(async () => {
    const r = await api("/api/wordpress");
    if (r.ok) setSites(await r.json());
  }, []);

  const loadPlugins = useCallback(async (domain) => {
    const r = await api(`/api/wordpress/${domain}/plugins`);
    if (r.ok) setPlugins(await r.json()); else setPlugins([]);
  }, []);

  const loadThemes = useCallback(async (domain) => {
    const r = await api(`/api/wordpress/${domain}/themes`);
    if (r.ok) setThemes(await r.json()); else setThemes([]);
  }, []);

  useEffect(() => { loadSites(); }, [loadSites]);

  useEffect(() => {
    if (!selected) return;
    if (tab === "plugins") loadPlugins(selected.domain);
    if (tab === "themes")  loadThemes(selected.domain);
  }, [selected, tab, loadPlugins, loadThemes]);

  const selectSite = s => { setSelected(s); setTab("plugins"); };

  const createSite = async () => {
    setLoading(true);
    const r = await api("/api/wordpress", {method:"POST", body:JSON.stringify(form)});
    setLoading(false);
    if (r.ok) {
      const data = await r.json();
      flash(`✓ WordPress installed! DB pass: ${data.db_pass}`);
      setCreateModal(false);
      loadSites();
    } else {
      const e = await r.json();
      flash("✗ " + (e.error || "Install failed"));
    }
  };

  const deleteSite = async (domain) => {
    if (!confirm(`Delete WordPress site ${domain}? This cannot be undone.`)) return;
    await api(`/api/wordpress/${domain}`, {method:"DELETE", body:JSON.stringify({delete_db:true})});
    if (selected?.domain === domain) setSelected(null);
    loadSites(); flash("Site deleted");
  };

  const installPlugin = async () => {
    setLoading(true);
    const r = await api(`/api/wordpress/${selected.domain}/plugins`, {method:"POST", body:JSON.stringify(pluginForm)});
    setLoading(false);
    if (r.ok) { setPluginModal(false); loadPlugins(selected.domain); flash("✓ Plugin installed"); }
    else { const e=await r.json(); flash("✗ "+(e.error||"Failed")); }
  };

  const togglePlugin = async (plugin, action) => {
    await api(`/api/wordpress/${selected.domain}/plugins/${plugin}`, {method:"PUT", body:JSON.stringify({action})});
    loadPlugins(selected.domain);
  };

  const installTheme = async () => {
    setLoading(true);
    const r = await api(`/api/wordpress/${selected.domain}/themes`, {method:"POST", body:JSON.stringify(themeForm)});
    setLoading(false);
    if (r.ok) { setThemeModal(false); loadThemes(selected.domain); flash("✓ Theme installed"); }
    else { const e=await r.json(); flash("✗ "+(e.error||"Failed")); }
  };

  const toggleTheme = async (theme, action) => {
    await api(`/api/wordpress/${selected.domain}/themes/${theme}`, {method:"PUT", body:JSON.stringify({action})});
    loadThemes(selected.domain);
  };

  const updateCore = async () => {
    setLoading(true);
    await api(`/api/wordpress/${selected.domain}/update`, {method:"POST"});
    setLoading(false);
    flash("✓ Core update complete");
    loadSites();
  };

  const setMaintenance = async (enable) => {
    await api(`/api/wordpress/${selected.domain}/maintenance`, {method:"POST", body:JSON.stringify({enable})});
    flash(enable ? "✓ Maintenance mode ON" : "✓ Maintenance mode OFF");
  };

  const cacheFlush = async () => {
    await api(`/api/wordpress/${selected.domain}/cache-flush`, {method:"POST"});
    flash("✓ Cache flushed");
  };

  const searchReplace = async () => {
    setLoading(true);
    const r = await api(`/api/wordpress/${selected.domain}/search-replace`, {method:"POST", body:JSON.stringify(srForm)});
    setLoading(false);
    if (r.ok) { setSrModal(false); flash("✓ Search & replace complete"); }
    else { const e=await r.json(); flash("✗ "+(e.error||"Failed")); }
  };

  const popularPlugins = ["woocommerce","yoast-seo","elementor","contact-form-7","wordfence","wpforms-lite","akismet","jetpack","wp-super-cache","w3-total-cache"];
  const popularThemes  = ["astra","generatepress","oceanwp","twentytwentyfour","kadence","blocksy","storefront","divi"];

  return (
    <div className="space-y-5">
      <SectionHeader title="WordPress Sites" sub={`${sites.length} site${sites.length!==1?"s":""}`} onAdd={()=>setCreateModal(true)} addLabel="+ Install WordPress"/>

      {msg && <div className={`px-4 py-2 rounded-lg text-xs font-mono border ${msg.startsWith("✓")?"bg-emerald-500/10 border-emerald-500/30 text-emerald-400":"bg-rose-500/10 border-rose-500/30 text-rose-400"}`}>{msg}</div>}

      <div className="grid lg:grid-cols-3 gap-5">
        {/* Site list */}
        <div className="space-y-2">
          {sites.length === 0 && <EmptyState icon="◍" text="No WordPress sites yet"/>}
          {sites.map(s => (
            <div key={s.domain} onClick={()=>selectSite(s)}
              className={`bg-zinc-900 border rounded-xl p-4 cursor-pointer transition-all ${selected?.domain===s.domain?"border-cyan-500/50 bg-cyan-500/5":"border-zinc-800 hover:border-zinc-700"}`}>
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0">
                  <p className="text-sm font-bold text-white font-mono truncate">{s.domain}</p>
                  {s.wp_version && <p className="text-xs text-zinc-500 mt-0.5">WP {s.wp_version}</p>}
                  <div className="flex items-center gap-2 mt-1.5">
                    <StatusBadge status={s.active?"active":"inactive"}/>
                    {s.ssl && <span className="text-xs text-emerald-400 font-mono">🔒 SSL</span>}
                  </div>
                </div>
                <Btn small variant="danger" onClick={e=>{e.stopPropagation();deleteSite(s.domain);}}>×</Btn>
              </div>
            </div>
          ))}
        </div>

        {/* Site detail */}
        <div className="lg:col-span-2">
          {!selected ? (
            <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-12 text-center">
              <p className="text-zinc-600 font-mono text-sm">Select a site to manage</p>
            </div>
          ) : (
            <div className="space-y-4">
              {/* Toolbar */}
              <div className="bg-zinc-900 border border-zinc-800 rounded-xl p-4">
                <div className="flex items-center justify-between mb-3">
                  <div>
                    <p className="text-sm font-bold text-white font-mono">{selected.domain}</p>
                    <p className="text-xs text-zinc-500 mt-0.5">DB: <span className="text-zinc-400">{selected.db_name}</span></p>
                  </div>
                  <div className="flex flex-wrap gap-2 justify-end">
                    <Btn small variant="ghost" onClick={updateCore} className={loading?"opacity-50":""}>↑ Update Core</Btn>
                    <Btn small variant="ghost" onClick={()=>setMaintenance(true)}>🔧 Maintenance ON</Btn>
                    <Btn small variant="ghost" onClick={()=>setMaintenance(false)}>✓ Maintenance OFF</Btn>
                    <Btn small variant="ghost" onClick={cacheFlush}>⚡ Flush Cache</Btn>
                    <Btn small variant="ghost" onClick={()=>setSrModal(true)}>⇄ Search & Replace</Btn>
                    <a href={`http://${selected.domain}/wp-admin`} target="_blank" rel="noreferrer">
                      <Btn small variant="primary">↗ WP Admin</Btn>
                    </a>
                  </div>
                </div>
              </div>

              {/* Tabs */}
              <div className="flex gap-1 bg-zinc-900 border border-zinc-800 rounded-xl p-1">
                {["plugins","themes"].map(t=>(
                  <button key={t} onClick={()=>setTab(t)}
                    className={`flex-1 text-xs font-mono py-2 rounded-lg transition-all cursor-pointer capitalize ${tab===t?"bg-cyan-500/10 text-cyan-400 border border-cyan-500/20":"text-zinc-500 hover:text-zinc-300"}`}>
                    {t}
                  </button>
                ))}
              </div>

              {/* Plugins */}
              {tab === "plugins" && (
                <div className="space-y-3">
                  <div className="flex justify-end"><Btn small onClick={()=>setPluginModal(true)}>+ Install Plugin</Btn></div>
                  <Table cols={["Plugin","Version","Status"]} actions rows={(plugins||[]).map((p,i)=>(
                    <TR key={i} cells={[
                      <span className="text-white">{p.name}</span>,
                      <span className="text-zinc-500">{p.version}</span>,
                      <StatusBadge status={p.status==="active"?"active":"inactive"}/>
                    ]} actions={[
                      p.status==="active"
                        ? <Btn key="d" small variant="ghost" onClick={()=>togglePlugin(p.name,"deactivate")}>Deactivate</Btn>
                        : <Btn key="a" small variant="success" onClick={()=>togglePlugin(p.name,"activate")}>Activate</Btn>,
                      <Btn key="u" small variant="ghost" onClick={()=>togglePlugin(p.name,"update")}>↑ Update</Btn>,
                      <Btn key="x" small variant="danger" onClick={()=>togglePlugin(p.name,"delete")}>Delete</Btn>,
                    ]}/>
                  ))}/>
                  {plugins.length===0 && <EmptyState icon="◍" text="No plugins installed"/>}
                </div>
              )}

              {/* Themes */}
              {tab === "themes" && (
                <div className="space-y-3">
                  <div className="flex justify-end"><Btn small onClick={()=>setThemeModal(true)}>+ Install Theme</Btn></div>
                  <Table cols={["Theme","Version","Status"]} actions rows={(themes||[]).map((t,i)=>(
                    <TR key={i} cells={[
                      <span className="text-white">{t.name}</span>,
                      <span className="text-zinc-500">{t.version}</span>,
                      <StatusBadge status={t.status==="active"?"active":"inactive"}/>
                    ]} actions={[
                      t.status!=="active" && <Btn key="a" small variant="success" onClick={()=>toggleTheme(t.name,"activate")}>Activate</Btn>,
                      <Btn key="u" small variant="ghost" onClick={()=>toggleTheme(t.name,"update")}>↑ Update</Btn>,
                      t.status!=="active" && <Btn key="x" small variant="danger" onClick={()=>toggleTheme(t.name,"delete")}>Delete</Btn>,
                    ].filter(Boolean)}/>
                  ))}/>
                  {themes.length===0 && <EmptyState icon="◍" text="No themes installed"/>}
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Create Site Modal */}
      {createModal && <Modal title="Install WordPress" onClose={()=>setCreateModal(false)} wide>
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <Field label="Domain" value={form.domain} onChange={e=>setForm(f({domain:e.target.value}))} placeholder="myblog.com"/>
            <Field label="Site Title" value={form.site_title} onChange={e=>setForm(f({site_title:e.target.value}))} placeholder="My Blog"/>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <Field label="WP Admin Username" value={form.admin_user} onChange={e=>setForm(f({admin_user:e.target.value}))} placeholder="admin"/>
            <Field label="WP Admin Password" type="password" value={form.admin_pass} onChange={e=>setForm(f({admin_pass:e.target.value}))} placeholder="••••••••"/>
          </div>
          <Field label="WP Admin Email" value={form.admin_email} onChange={e=>setForm(f({admin_email:e.target.value}))} placeholder="admin@myblog.com"/>
          <div className="border-t border-zinc-800 pt-3">
            <p className="text-xs text-zinc-500 font-mono mb-3">Database (leave blank to auto-generate)</p>
            <div className="grid grid-cols-3 gap-3">
              <Field label="DB Name" value={form.db_name} onChange={e=>setForm(f({db_name:e.target.value}))} placeholder="wp_myblog"/>
              <Field label="DB User" value={form.db_user} onChange={e=>setForm(f({db_user:e.target.value}))} placeholder="wp_user"/>
              <Field label="DB Password" type="password" value={form.db_pass} onChange={e=>setForm(f({db_pass:e.target.value}))} placeholder="auto-generate"/>
            </div>
          </div>
          <Select label="PHP Version" value={form.php} onChange={e=>setForm(f({php:e.target.value}))} options={[{v:"8.2",l:"PHP 8.2 (recommended)"},{v:"8.1",l:"PHP 8.1"},{v:"8.3",l:"PHP 8.3"}]}/>
          <div className="bg-zinc-800 rounded-lg p-3 text-xs text-zinc-500 font-mono">
            This will: create a MariaDB database, download WordPress, configure wp-config.php, run WP install, and set up an Nginx vhost.
          </div>
          <div className="flex justify-end gap-2">
            <Btn variant="ghost" onClick={()=>setCreateModal(false)}>Cancel</Btn>
            <Btn onClick={createSite} className={loading?"opacity-50 cursor-not-allowed":""}>
              {loading ? "Installing…" : "Install WordPress"}
            </Btn>
          </div>
        </div>
      </Modal>}

      {/* Plugin Install Modal */}
      {pluginModal && <Modal title="Install Plugin" onClose={()=>setPluginModal(false)}>
        <div className="space-y-4">
          <Field label="Plugin slug (from wordpress.org)" value={pluginForm.name} onChange={e=>setPluginForm({...pluginForm,name:e.target.value})} placeholder="woocommerce"/>
          <div>
            <p className="text-xs text-zinc-500 font-mono mb-2">Popular plugins</p>
            <div className="flex flex-wrap gap-1.5">
              {popularPlugins.map(p=>(
                <button key={p} onClick={()=>setPluginForm({...pluginForm,name:p})}
                  className={`text-xs font-mono px-2 py-1 rounded border cursor-pointer transition-colors ${pluginForm.name===p?"border-cyan-500 text-cyan-400 bg-cyan-500/10":"border-zinc-700 text-zinc-500 hover:text-zinc-300 hover:border-zinc-600"}`}>{p}</button>
              ))}
            </div>
          </div>
          <label className="flex items-center gap-2 cursor-pointer">
            <input type="checkbox" checked={pluginForm.activate} onChange={e=>setPluginForm({...pluginForm,activate:e.target.checked})} className="accent-cyan-500"/>
            <span className="text-xs font-mono text-zinc-400">Activate after install</span>
          </label>
          <div className="flex justify-end gap-2">
            <Btn variant="ghost" onClick={()=>setPluginModal(false)}>Cancel</Btn>
            <Btn onClick={installPlugin} className={loading?"opacity-50":""}>
              {loading?"Installing…":"Install Plugin"}
            </Btn>
          </div>
        </div>
      </Modal>}

      {/* Theme Install Modal */}
      {themeModal && <Modal title="Install Theme" onClose={()=>setThemeModal(false)}>
        <div className="space-y-4">
          <Field label="Theme slug (from wordpress.org)" value={themeForm.name} onChange={e=>setThemeForm({...themeForm,name:e.target.value})} placeholder="astra"/>
          <div>
            <p className="text-xs text-zinc-500 font-mono mb-2">Popular themes</p>
            <div className="flex flex-wrap gap-1.5">
              {popularThemes.map(t=>(
                <button key={t} onClick={()=>setThemeForm({...themeForm,name:t})}
                  className={`text-xs font-mono px-2 py-1 rounded border cursor-pointer transition-colors ${themeForm.name===t?"border-cyan-500 text-cyan-400 bg-cyan-500/10":"border-zinc-700 text-zinc-500 hover:text-zinc-300 hover:border-zinc-600"}`}>{t}</button>
              ))}
            </div>
          </div>
          <label className="flex items-center gap-2 cursor-pointer">
            <input type="checkbox" checked={themeForm.activate} onChange={e=>setThemeForm({...themeForm,activate:e.target.checked})} className="accent-cyan-500"/>
            <span className="text-xs font-mono text-zinc-400">Activate after install</span>
          </label>
          <div className="flex justify-end gap-2">
            <Btn variant="ghost" onClick={()=>setThemeModal(false)}>Cancel</Btn>
            <Btn onClick={installTheme} className={loading?"opacity-50":""}>
              {loading?"Installing…":"Install Theme"}
            </Btn>
          </div>
        </div>
      </Modal>}

      {/* Search & Replace Modal */}
      {srModal && <Modal title="Search & Replace in Database" onClose={()=>setSrModal(false)}>
        <div className="space-y-4">
          <div className="bg-amber-500/10 border border-amber-500/30 rounded-lg px-3 py-2 text-xs font-mono text-amber-400">
            ⚠ This runs directly on the database. Use for URL migrations (e.g. http → https).
          </div>
          <Field label="Search for" value={srForm.search} onChange={e=>setSrForm({...srForm,search:e.target.value})} placeholder="http://myblog.com"/>
          <Field label="Replace with" value={srForm.replace} onChange={e=>setSrForm({...srForm,replace:e.target.value})} placeholder="https://myblog.com"/>
          <div className="flex justify-end gap-2">
            <Btn variant="ghost" onClick={()=>setSrModal(false)}>Cancel</Btn>
            <Btn onClick={searchReplace} className={loading?"opacity-50":""}>
              {loading?"Running…":"Run Search & Replace"}
            </Btn>
          </div>
        </div>
      </Modal>}
    </div>
  );
}

// ── App Shell ──────────────────────────────────────────────────────────────
export default function App() {
  const [auth, setAuth] = useState(!!localStorage.getItem("sp_token"));
  const [active, setActive] = useState("dashboard");
  const [collapsed, setCollapsed] = useState(false);

  if (!auth) return <Login onLogin={()=>setAuth(true)}/>;

  const logout = () => { localStorage.removeItem("sp_token"); setAuth(false); };

  const panels = { dashboard:<Dashboard/>, users:<UsersPanel/>, webserver:<WebServerPanel/>, wordpress:<WordPressPanel/>, databases:<DatabasePanel/>, filemanager:<FileManagerPanel/>, email:<EmailPanel/>, dns:<DNSPanel/>, cron:<CronPanel/>, ftp:<FTPPanel/> };

  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-200 flex" style={{fontFamily:"'JetBrains Mono','Fira Code',monospace"}}>
      <style>{`
        @import url('https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;700&display=swap');
        *{box-sizing:border-box;}
        ::-webkit-scrollbar{width:4px;height:4px;}
        ::-webkit-scrollbar-track{background:#09090b;}
        ::-webkit-scrollbar-thumb{background:#27272a;border-radius:2px;}
      `}</style>

      {/* Sidebar */}
      <div className={`${collapsed?"w-14":"w-56"} transition-all duration-200 bg-zinc-900 border-r border-zinc-800 flex flex-col shrink-0`}>
        <div className="p-4 border-b border-zinc-800 flex items-center gap-2">
          <div className="w-7 h-7 bg-cyan-500 rounded-lg flex items-center justify-center text-black font-bold text-sm shrink-0">⬡</div>
          {!collapsed && <span className="text-sm font-bold text-white tracking-widest">BLOGRON</span>}
        </div>
        <nav className="flex-1 p-2 space-y-0.5 overflow-y-auto">
          {NAV.map(item=>(
            <button key={item.id} onClick={()=>setActive(item.id)} className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-xs font-mono transition-all cursor-pointer ${active===item.id?"bg-cyan-500/10 text-cyan-400 border border-cyan-500/20":"text-zinc-500 hover:text-zinc-300 hover:bg-zinc-800 border border-transparent"}`}>
              <span className="text-base shrink-0">{item.icon}</span>
              {!collapsed && <span>{item.label}</span>}
            </button>
          ))}
        </nav>
        <div className="p-3 border-t border-zinc-800 space-y-1">
          {!collapsed && <button onClick={logout} className="w-full text-left text-xs font-mono text-zinc-600 hover:text-rose-400 transition-colors px-3 py-2 cursor-pointer">Sign Out</button>}
          <button onClick={()=>setCollapsed(!collapsed)} className="w-full flex items-center justify-center text-zinc-600 hover:text-zinc-400 cursor-pointer text-sm py-1">
            {collapsed?"▶":"◀"}
          </button>
        </div>
      </div>

      {/* Main */}
      <div className="flex-1 flex flex-col overflow-hidden">
        <header className="bg-zinc-900/80 border-b border-zinc-800 px-6 py-3 flex items-center justify-between backdrop-blur-sm shrink-0">
          <div className="flex items-center gap-3">
            <div className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"/>
            <span className="text-xs text-zinc-500 font-mono">{NAV.find(n=>n.id===active)?.label}</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-xs text-zinc-700 font-mono">root@server</span>
            <button onClick={logout} className="text-xs font-mono text-zinc-600 hover:text-rose-400 transition-colors cursor-pointer">logout</button>
          </div>
        </header>
        <main className="flex-1 overflow-y-auto p-6">{panels[active]}</main>
      </div>
    </div>
  );
}
