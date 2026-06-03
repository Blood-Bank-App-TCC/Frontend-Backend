import { CheckCircle2 } from "lucide-react";
import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { bankDarahController } from "../../controllers/bankDarahController";
import { apiErrorMessage } from "../../models/apiClient";
import type { EmergencyRequest, LiveResponse } from "../../models/types";
import { formatDateTime } from "../../models/status";
import Metric from "../components/Metric";
import StatusBadge from "../components/StatusBadge";
import PageHeader from "../layout/PageHeader";

export default function MonitorPage() {
  const { id = "" } = useParams();
  const [responses, setResponses] = useState<LiveResponse[]>([]);
  const [requests, setRequests] = useState<EmergencyRequest[]>([]);
  const [closing, setClosing] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  async function loadRequests() {
    setRequests(await bankDarahController.requests());
  }

  useEffect(() => {
    loadRequests().catch((err) => setError(apiErrorMessage(err, "Data request gagal dimuat.")));
    const load = () =>
      bankDarahController
        .liveResponses(id)
        .then(setResponses)
        .catch((err) => setError(apiErrorMessage(err, "Respons pendonor gagal dimuat.")));
    load();
    const timer = window.setInterval(load, 2500);
    return () => window.clearInterval(timer);
  }, [id]);

  const request = requests.find((item) => item.id === id);
  const canClose = request?.status === "ACTIVE" || request?.status === "PENDING";
  const counts = {
    accepted: responses.filter((item) => item.status === "ACCEPTED").length,
    way: responses.filter((item) => item.status === "ON_THE_WAY").length,
    checked: responses.filter((item) => item.status === "CHECKED_IN").length,
    declined: responses.filter((item) => item.status === "DECLINED").length,
  };

  async function closeRequest() {
    setClosing(true);
    setError("");
    setMessage("");
    try {
      const closed = await bankDarahController.closeRequest(id);
      setRequests((items) => items.map((item) => (item.id === closed.id ? closed : item)));
      setMessage("Broadcast dan request berhasil ditutup.");
    } catch (err) {
      setError(apiErrorMessage(err, "Request gagal ditutup."));
    } finally {
      setClosing(false);
    }
  }

  return (
    <>
      <PageHeader
        eyebrow="Live response dashboard"
        title={request ? `Monitor ${request.hospitalName}` : "Monitor Broadcast"}
        action={
          <button className="btn danger" disabled={!canClose || closing} onClick={closeRequest} type="button">
            <CheckCircle2 size={18} />
            {closing ? "Menutup..." : "Tutup & Selesaikan"}
          </button>
        }
      />
      {error && <p className="alert danger">{error}</p>}
      {message && <p className="alert success">{message}</p>}
      {request && (
        <section className="emergency-summary">
          <div>
            <span>Status</span>
            <strong>{request.status}</strong>
          </div>
          <div>
            <span>Kebutuhan</span>
            <strong>{request.quantityNeeded} kantong</strong>
          </div>
          <div>
            <span>Golongan</span>
            <strong>
              {request.bloodType} / {request.productType}
            </strong>
          </div>
          <div>
            <span>Broadcast</span>
            <strong>{request.broadcastId || "-"}</strong>
          </div>
        </section>
      )}
      <section className="metric-grid">
        <Metric label="Siap Donor" value={counts.accepted} tone="success" />
        <Metric label="Menuju PMI" value={counts.way} tone="info" />
        <Metric label="Selesai" value={counts.checked} tone="success" />
        <Metric label="Tidak Bisa" value={counts.declined} tone="warning" />
      </section>
      <section className="panel">
        <div className="panel-header">
          <h2>Respons Pendonor</h2>
          <span className="live-dot">Auto-update</span>
        </div>
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Nama</th>
                <th>Golongan</th>
                <th>Jarak</th>
                <th>Status</th>
                <th>Waktu</th>
              </tr>
            </thead>
            <tbody>
              {responses.map((response) => (
                <tr key={response.id}>
                  <td>{response.donorName}</td>
                  <td>{response.bloodType}</td>
                  <td>{response.distanceKm.toFixed(1)} km</td>
                  <td>
                    <StatusBadge response={response.status} />
                  </td>
                  <td>{formatDateTime(response.respondedAt)}</td>
                </tr>
              ))}
              {responses.length === 0 && (
                <tr>
                  <td colSpan={5}>Belum ada respons masuk.</td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </section>
    </>
  );
}
