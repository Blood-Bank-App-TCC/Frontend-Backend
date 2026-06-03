import { Siren } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { bankDarahController } from "../../controllers/bankDarahController";
import { apiErrorMessage } from "../../models/apiClient";
import type { BloodStock, EmergencyRequest } from "../../models/types";
import { bloodTypes, stockLevel } from "../../models/status";
import BloodTypeCard from "../components/BloodTypeCard";
import { RequestTable } from "../components/DataTables";
import Metric from "../components/Metric";
import PageHeader from "../layout/PageHeader";

export default function DashboardPage() {
  const location = useLocation();
  const navigate = useNavigate();
  const [stock, setStock] = useState<BloodStock[]>([]);
  const [requests, setRequests] = useState<EmergencyRequest[]>([]);
  const [error, setError] = useState("");
  const [toast, setToast] = useState("");

  useEffect(() => {
    let active = true;

    async function load() {
      try {
        const [stockData, requestData] = await Promise.all([bankDarahController.stock(), bankDarahController.requests()]);
        if (!active) return;
        setStock(stockData);
        setRequests(requestData);
        setError("");
      } catch (err) {
        if (active) {
          setError(apiErrorMessage(err, "Dashboard gagal memuat data backend."));
        }
      }
    }

    load();
    const intervalId = window.setInterval(load, 5000);

    return () => {
      active = false;
      window.clearInterval(intervalId);
    };
  }, []);

  useEffect(() => {
    if (location.state && (location.state as { broadcastSuccess?: boolean }).broadcastSuccess) {
      setToast("Broadcast berhasil dikirim.");
      navigate(`${location.pathname}${location.hash}`, { replace: true, state: null });
      window.setTimeout(() => setToast(""), 3500);
    }
  }, [location, navigate]);

  useEffect(() => {
    if (location.hash === "#permintaan-aktif") {
      window.requestAnimationFrame(() => {
        document.getElementById("permintaan-aktif")?.scrollIntoView({ behavior: "smooth", block: "start" });
      });
    }
  }, [location.hash, requests.length]);

  const grouped = useMemo(
    () =>
      bloodTypes.map((bloodType) => ({
        bloodType,
        items: stock.filter((item) => item.bloodType === bloodType),
      })),
    [stock]
  );
  const critical = stock.filter((item) => stockLevel(item) === "critical").length;
  const low = stock.filter((item) => stockLevel(item) === "low").length;
  const activeRequests = requests.filter((request) => request.status === "ACTIVE" || request.status === "PENDING");

  return (
    <>
      <PageHeader
        eyebrow="Operasi stok real-time"
        title="Dashboard Stok Darah"
        action={
          <Link className="btn primary" to="/emergency/new">
            <Siren size={18} />
            Permintaan Baru
          </Link>
        }
      />
      {error && <p className="alert danger">{error}</p>}
      {toast && (
        <div className="toast success" role="status">
          {toast}
        </div>
      )}
      <section className="metric-grid">
        <Metric label="Total Kantong" value={stock.reduce((sum, item) => sum + item.quantity, 0)} tone="info" />
        <Metric label="Stok Kritis" value={critical} tone="danger" />
        <Metric label="Stok Menipis" value={low} tone="warning" />
        <Metric label="Request Aktif" value={activeRequests.length} tone="success" />
      </section>
      <section className="blood-grid">
        {grouped.map((group) => (
          <BloodTypeCard bloodType={group.bloodType} items={group.items} key={group.bloodType} />
        ))}
      </section>
      <section className="panel" id="permintaan-aktif">
        <div className="panel-header">
          <h2>Permintaan Aktif</h2>
          <Link to="/reports">Lihat laporan</Link>
        </div>
        <RequestTable requests={activeRequests} />
      </section>
    </>
  );
}
