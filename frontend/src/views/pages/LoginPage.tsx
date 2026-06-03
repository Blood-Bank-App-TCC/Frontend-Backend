import { Activity, ShieldCheck } from "lucide-react";
import { type FormEvent, useState } from "react";
import { useNavigate } from "react-router-dom";
import { bankDarahController } from "../../controllers/bankDarahController";
import { apiErrorMessage } from "../../models/apiClient";
import { useAuthStore } from "../../models/authStore";

export default function LoginPage() {
  const navigate = useNavigate();
  const setSession = useAuthStore((state) => state.setSession);
  const [username, setUsername] = useState("operator");
  const [password, setPassword] = useState("pmi123");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setError("");
    setLoading(true);
    try {
      const session = await bankDarahController.login(username, password);
      setSession(session.token, session.user);
      navigate("/dashboard", { replace: true });
    } catch (err) {
      setError(apiErrorMessage(err, "Username atau password tidak sesuai."));
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="login-page">
      <section className="login-panel">
        <div className="brand large">
          <span className="brand-mark">
            <Activity size={28} />
          </span>
          <span>
            <strong>Bank Darah PMI</strong>
            <small>Emergency donor operations</small>
          </span>
        </div>
        <form className="form-stack" onSubmit={submit}>
          <label>
            Username
            <input value={username} onChange={(event) => setUsername(event.target.value)} required />
          </label>
          <label>
            Password
            <input type="password" value={password} onChange={(event) => setPassword(event.target.value)} required />
          </label>
          {error && <p className="form-error">{error}</p>}
          <button className="btn primary wide" disabled={loading} type="submit">
            <ShieldCheck size={18} />
            {loading ? "Memeriksa..." : "Masuk"}
          </button>
        </form>
        <div className="login-footnote">
          <span>Demo admin</span>
          <strong>operator / pmi123</strong>
        </div>
      </section>
    </main>
  );
}
