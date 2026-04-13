"use client";

import { useState } from "react";
import { Shield, AlertTriangle, CheckCircle2, Info, ChevronDown, ChevronUp, Loader2, Upload, Terminal } from "lucide-react";

const EXAMPLE_MANIFEST = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: example-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: example-app
  template:
    metadata:
      labels:
        app: example-app
    spec:
      containers:
      - name: app
        image: nginx:latest
        ports:
        - containerPort: 80
        securityContext:
          privileged: true`;

type Severity = "critical" | "high" | "medium" | "low" | "info";
type Category = "helm_lint" | "yaml_lint" | "deprecated_api" | "security_policy" | "reliability" | "vulnerability";

interface Finding {
  id: string;
  category: Category;
  severity: Severity;
  resource: string;
  message: string;
  remediation: string;
  docs_url?: string;
  cve?: string;
  cvss_score?: number;
}

interface ScoreBreakdown {
  security: number;
  api_stability: number;
  reliability: number;
  lint_clean: number;
  overall: number;
  grade: string;
}

interface ScanReport {
  scan_id: string;
  created_at: string;
  source: string;
  score: ScoreBreakdown;
  findings: Finding[];
  summary: { critical: number; high: number; medium: number; low: number; info: number; total: number };
}

const SEV_STYLES: Record<Severity, string> = {
  critical: "bg-red-900/40 text-red-300 border border-red-700/50",
  high:     "bg-orange-900/40 text-orange-300 border border-orange-700/50",
  medium:   "bg-yellow-900/40 text-yellow-300 border border-yellow-700/50",
  low:      "bg-slate-800/60 text-slate-300 border border-slate-600/50",
  info:     "bg-blue-900/30 text-blue-300 border border-blue-700/40",
};

const SEV_ICON: Record<Severity, JSX.Element> = {
  critical: <AlertTriangle className="w-4 h-4 text-red-400" />,
  high:     <AlertTriangle className="w-4 h-4 text-orange-400" />,
  medium:   <AlertTriangle className="w-4 h-4 text-yellow-400" />,
  low:      <Info className="w-4 h-4 text-slate-400" />,
  info:     <Info className="w-4 h-4 text-blue-400" />,
};

const GRADE_COLOR: Record<string, string> = {
  A: "text-emerald-400", B: "text-green-400", C: "text-yellow-400", D: "text-orange-400", F: "text-red-500",
};

const CAT_LABEL: Record<Category, string> = {
  helm_lint:       "Helm Lint",
  yaml_lint:       "YAML Lint",
  deprecated_api:  "Deprecated API",
  security_policy: "Security",
  reliability:     "Reliability",
  vulnerability:   "CVE",
};

function ScoreRing({ score, grade }: { score: number; grade: string }) {
  const r = 52;
  const circ = 2 * Math.PI * r;
  const dash = (score / 100) * circ;
  const gradeColor = GRADE_COLOR[grade] ?? "text-slate-400";
  return (
    <div className="relative flex items-center justify-center w-36 h-36">
      <svg className="absolute" width="144" height="144" viewBox="0 0 144 144">
        <circle cx="72" cy="72" r={r} fill="none" stroke="#1e2d45" strokeWidth="10" />
        <circle
          cx="72" cy="72" r={r} fill="none"
          stroke={score >= 75 ? "#10b981" : score >= 60 ? "#f59e0b" : "#ef4444"}
          strokeWidth="10"
          strokeDasharray={`${dash} ${circ}`}
          strokeLinecap="round"
          transform="rotate(-90 72 72)"
        />
      </svg>
      <div className="text-center z-10">
        <div className={`text-3xl font-bold ${gradeColor}`}>{grade}</div>
        <div className="text-sm text-slate-400">{score.toFixed(1)}/100</div>
      </div>
    </div>
  );
}

function FindingRow({ f }: { f: Finding }) {
  const [open, setOpen] = useState(false);
  return (
    <div className={`rounded-lg px-4 py-3 mb-2 ${SEV_STYLES[f.severity]} cursor-pointer`} onClick={() => setOpen(!open)}>
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-start gap-2 flex-1 min-w-0">
          {SEV_ICON[f.severity]}
          <div className="min-w-0">
            <div className="flex items-center gap-2 flex-wrap">
              <span className="text-xs font-mono opacity-60">{f.resource}</span>
              <span className="text-xs px-1.5 py-0.5 rounded bg-white/10">{CAT_LABEL[f.category]}</span>
              {f.cve && <span className="text-xs px-1.5 py-0.5 rounded bg-red-900/60 text-red-300">{f.cve}</span>}
            </div>
            <p className="text-sm mt-0.5">{f.message}</p>
          </div>
        </div>
        {open ? <ChevronUp className="w-4 h-4 shrink-0 opacity-50 mt-0.5" /> : <ChevronDown className="w-4 h-4 shrink-0 opacity-50 mt-0.5" />}
      </div>
      {open && f.remediation && (
        <div className="mt-3 pt-3 border-t border-white/10 text-sm opacity-80">
          <span className="font-medium">Fix: </span>{f.remediation}
          {f.docs_url && (
            <a href={f.docs_url} target="_blank" rel="noopener noreferrer"
              className="ml-2 text-blue-400 hover:underline text-xs" onClick={e => e.stopPropagation()}>
              docs →
            </a>
          )}
        </div>
      )}
    </div>
  );
}

export default function Home() {
  const [manifest, setManifest] = useState(EXAMPLE_MANIFEST);
  const [source, setSource] = useState("example.yaml");
  const [scanning, setScanning] = useState(false);
  const [report, setReport] = useState<ScanReport | null>(null);
  const [error, setError] = useState("");

  const handleScan = async () => {
    if (!manifest.trim()) { setError("Paste a manifest first"); return; }
    setScanning(true); setError(""); setReport(null);
    try {
      const res = await fetch("/api/scan", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ manifest, source }),
      });
      if (!res.ok) throw new Error(`Scanner returned ${res.status}`);
      setReport(await res.json());
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Scan failed");
    } finally {
      setScanning(false);
    }
  };

  const handleFile = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setSource(file.name);
    const reader = new FileReader();
    reader.onload = ev => setManifest(ev.target?.result as string);
    reader.readAsText(file);
  };

  return (
    <main className="min-h-screen bg-[#080c14]">
      {/* Header */}
      <header className="border-b border-[#1e2d45] px-6 py-4 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <span className="text-2xl">🦅</span>
          <div>
            <h1 className="text-lg font-bold text-white leading-none">ShrikeOps</h1>
            <p className="text-xs text-slate-500">Kubernetes Helm Safety Gate</p>
          </div>
        </div>
        <div className="flex items-center gap-2 text-xs text-slate-500">
          <Shield className="w-3.5 h-3.5" />
          <span>Pluto · Polaris · kube-score · OSV</span>
        </div>
      </header>

      <div className="max-w-6xl mx-auto px-6 py-10 grid grid-cols-1 lg:grid-cols-2 gap-8">
        {/* Input Panel */}
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-sm font-semibold text-slate-300 flex items-center gap-2">
              <Terminal className="w-4 h-4 text-blue-400" /> Manifest / Chart YAML
            </h2>
            <label className="flex items-center gap-1.5 text-xs text-slate-400 cursor-pointer hover:text-blue-400 transition-colors">
              <Upload className="w-3.5 h-3.5" />
              Upload file
              <input type="file" accept=".yaml,.yml" className="hidden" onChange={handleFile} />
            </label>
          </div>

          <textarea
            className="w-full h-[480px] bg-[#0d1320] border border-[#1e2d45] rounded-xl px-4 py-3 font-mono text-sm text-slate-300 resize-none focus:outline-none focus:border-blue-500/60 placeholder:text-slate-600"
            placeholder="Paste your Kubernetes manifest or Helm template YAML here..."
            value={manifest}
            onChange={e => setManifest(e.target.value)}
          />

          <div className="flex items-center gap-3">
            <input
              className="flex-1 bg-[#0d1320] border border-[#1e2d45] rounded-lg px-3 py-2 text-sm text-slate-300 focus:outline-none focus:border-blue-500/60"
              placeholder="Source label (e.g. my-chart.yaml)"
              value={source}
              onChange={e => setSource(e.target.value)}
            />
            <button
              onClick={handleScan}
              disabled={scanning}
              className="flex items-center gap-2 bg-blue-600 hover:bg-blue-500 disabled:opacity-50 disabled:cursor-not-allowed text-white font-semibold px-6 py-2 rounded-lg transition-colors text-sm"
            >
              {scanning ? <Loader2 className="w-4 h-4 animate-spin" /> : <Shield className="w-4 h-4" />}
              {scanning ? "Scanning…" : "Scan"}
            </button>
          </div>

          {error && (
            <div className="bg-red-900/30 border border-red-700/50 text-red-300 rounded-lg px-4 py-3 text-sm">
              {error}
            </div>
          )}
        </div>

        {/* Results Panel */}
        <div className="space-y-6">
          {!report && !scanning && (
            <div className="h-full flex flex-col items-center justify-center text-center text-slate-600 gap-4 py-20">
              <Shield className="w-16 h-16 opacity-20" />
              <div>
                <p className="text-base font-medium text-slate-500">No scan yet</p>
                <p className="text-sm mt-1">Paste a manifest and click Scan to analyse</p>
              </div>
              <div className="mt-4 text-xs space-y-1 text-slate-600">
                <p>✓ YAML lint &nbsp;·&nbsp; ✓ Deprecated APIs &nbsp;·&nbsp; ✓ Security policies</p>
                <p>✓ Reliability scoring &nbsp;·&nbsp; ✓ CVE lookup via OSV.dev</p>
              </div>
            </div>
          )}

          {scanning && (
            <div className="flex flex-col items-center justify-center py-24 gap-4 text-slate-400">
              <Loader2 className="w-10 h-10 animate-spin text-blue-500" />
              <p className="text-sm">Running Pluto · Polaris · kube-score…</p>
            </div>
          )}

          {report && (
            <>
              {/* Score Card */}
              <div className="bg-[#0d1320] border border-[#1e2d45] rounded-xl p-6">
                <div className="flex items-center gap-6">
                  <ScoreRing score={report.score.overall} grade={report.score.grade} />
                  <div className="flex-1 space-y-2">
                    <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-sm">
                      {[
                        ["Security", report.score.security, "40%"],
                        ["API Stability", report.score.api_stability, "20%"],
                        ["Reliability", report.score.reliability, "25%"],
                        ["Lint", report.score.lint_clean, "15%"],
                      ].map(([label, val, weight]) => (
                        <div key={String(label)} className="flex items-center justify-between">
                          <span className="text-slate-400">{label} <span className="text-slate-600 text-xs">({weight})</span></span>
                          <span className={`font-mono font-semibold ${Number(val) >= 75 ? "text-emerald-400" : Number(val) >= 50 ? "text-yellow-400" : "text-red-400"}`}>
                            {Number(val).toFixed(1)}
                          </span>
                        </div>
                      ))}
                    </div>
                  </div>
                </div>

                {/* Summary pills */}
                <div className="mt-4 pt-4 border-t border-[#1e2d45] flex flex-wrap gap-2">
                  {([["critical","red"],["high","orange"],["medium","yellow"],["low","slate"],["info","blue"]] as const).map(([sev, color]) => (
                    report.summary[sev] > 0 && (
                      <span key={sev} className={`text-xs px-2 py-1 rounded-full bg-${color}-900/40 text-${color}-300 border border-${color}-700/40`}>
                        {report.summary[sev]} {sev}
                      </span>
                    )
                  ))}
                  {report.summary.total === 0 && (
                    <span className="text-xs flex items-center gap-1 text-emerald-400">
                      <CheckCircle2 className="w-3.5 h-3.5" /> All checks passed
                    </span>
                  )}
                </div>
              </div>

              {/* Findings */}
              {report.findings.length > 0 && (
                <div>
                  <h3 className="text-sm font-semibold text-slate-400 mb-3">
                    Findings <span className="text-slate-600">({report.findings.length})</span>
                  </h3>
                  <div className="max-h-[420px] overflow-y-auto pr-1 space-y-0">
                    {report.findings.map(f => <FindingRow key={f.id} f={f} />)}
                  </div>
                </div>
              )}

              <p className="text-xs text-slate-600">Scan ID: {report.scan_id} · {new Date(report.created_at).toLocaleString()}</p>
            </>
          )}
        </div>
      </div>
    </main>
  );
}
