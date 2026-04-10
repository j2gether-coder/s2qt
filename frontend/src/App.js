import { useState } from "react";
import "./App.css";
import { RunPipeline } from "../wailsjs/go/main/App";

export default function App() {
  return (
    <div style={{ padding: "40px", fontSize: "30px", color: "black" }}>
      S2QT TEST
    </div>
  );
}

export default function App() {
  const [url, setUrl] = useState("");
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState("대기 중");
  const [result, setResult] = useState("");
  const [log, setLog] = useState("");
  const [files, setFiles] = useState({
    videoFile: "",
    wavFile: "",
    transcriptFile: "",
    markdownFile: "",
  });

  const handleRun = async () => {
    const trimmed = url.trim();
    if (!trimmed) {
      setMessage("URL을 입력해 주세요.");
      return;
    }

    try {
      setLoading(true);
      setMessage("처리 중...");
      setResult("");
      setLog("");
      setFiles({
        videoFile: "",
        wavFile: "",
        transcriptFile: "",
        markdownFile: "",
      });

      const res = await RunPipeline(trimmed);

      setMessage(res.message || "완료");
      setResult(res.transcriptText || "");
      setLog(res.log || "");
      setFiles({
        videoFile: res.videoFile || "",
        wavFile: res.wavFile || "",
        transcriptFile: res.transcriptFile || "",
        markdownFile: res.markdownFile || "",
      });
    } catch (err) {
      setMessage(`오류: ${String(err)}`);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="page">
      <header className="header">
        <h1>S2QT 영상 전사</h1>
        <p>URL 입력 → 다운로드 → WAV 변환 → Whisper 전사</p>
      </header>

      <section className="card">
        <label className="label">동영상 URL</label>
        <div className="row">
          <input
            className="input"
            type="text"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="https://..."
            disabled={loading}
          />
          <button className="button" onClick={handleRun} disabled={loading}>
            {loading ? "처리 중..." : "실행"}
          </button>
        </div>
        <div className="status">{message}</div>
      </section>

      <section className="grid">
        <div className="card">
          <h2>결과 파일</h2>
          <div className="fileitem"><strong>Video:</strong> {files.videoFile || "-"}</div>
          <div className="fileitem"><strong>WAV:</strong> {files.wavFile || "-"}</div>
          <div className="fileitem"><strong>TXT:</strong> {files.transcriptFile || "-"}</div>
          <div className="fileitem"><strong>MD:</strong> {files.markdownFile || "-"}</div>
        </div>

        <div className="card">
          <h2>실행 로그</h2>
          <textarea className="textarea logbox" value={log} readOnly />
        </div>
      </section>

      <section className="card">
        <h2>전사 결과</h2>
        <textarea className="textarea resultbox" value={result} readOnly />
      </section>
    </div>
  );
}